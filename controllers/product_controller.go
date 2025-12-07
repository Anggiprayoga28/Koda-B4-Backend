package controllers

import (
	"coffee-shop/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ProductController struct{}

func (ctrl *ProductController) getPaginationParams(c *gin.Context, defaultLimit int) (page, limit, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", fmt.Sprintf("%d", defaultLimit)))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 100 {
		limit = 100
	}
	offset = (page - 1) * limit
	return
}

func (ctrl *ProductController) generateProductLinks(c *gin.Context, page, limit, total int) models.PaginationLinks {
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}

	host := c.Request.Host
	path := c.Request.URL.Path
	queryParams := c.Request.URL.Query()

	makeURL := func(pageNum int) string {
		newParams := url.Values{}
		for key, values := range queryParams {
			if key != "page" {
				for _, value := range values {
					newParams.Add(key, value)
				}
			}
		}
		newParams.Set("page", strconv.Itoa(pageNum))
		newParams.Set("limit", strconv.Itoa(limit))
		return fmt.Sprintf("%s://%s%s?%s", scheme, host, path, newParams.Encode())
	}

	totalPages := 0
	if limit > 0 && total >= 0 {
		totalPages = (total + limit - 1) / limit
	}

	links := models.PaginationLinks{
		Self: makeURL(page),
	}
	if page > 1 {
		links.Prev = makeURL(page - 1)
	}
	if page < totalPages {
		links.Next = makeURL(page + 1)
	}
	return links
}

func (ctrl *ProductController) buildProductResponse(c *gin.Context, message string, data interface{}, page, limit, total int) models.HATEOASResponse {
	totalPages := 0
	if limit > 0 && total >= 0 {
		totalPages = (total + limit - 1) / limit
	}

	meta := models.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: total,
		TotalPages: totalPages,
	}

	links := ctrl.generateProductLinks(c, page, limit, total)

	return models.HATEOASResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
		Links:   links,
	}
}

func getProductCacheKey(page, limit int, params url.Values) string {
	filtered := url.Values{}

	for key, values := range params {
		if key == "page" || key == "limit" {
			continue
		}
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			filtered.Add(key, v)
		}
	}

	encoded := filtered.Encode()
	if encoded == "" {
		return fmt.Sprintf("products_list_p%d_l%d", page, limit)
	}

	return fmt.Sprintf("products_list_p%d_l%d_%s", page, limit, encoded)
}

func invalidateProductCache() {
	if models.RedisClient == nil {
		return
	}

	ctx := context.Background()
	iter := models.RedisClient.Scan(ctx, 0, "products_*", 0).Iterator()
	for iter.Next(ctx) {
		if err := models.RedisClient.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete cache key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("Failed to scan cache keys: %v", err)
	}
}

// @Summary Get all products
// @Tags Products
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} models.HATEOASResponse
// @Router /products [get]
func (ctrl *ProductController) GetAllProducts(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)

	cacheKey := getProductCacheKey(page, limit, c.Request.URL.Query())
	ctx := context.Background()

	if models.RedisClient != nil {
		cached, err := models.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			log.Println("Serving products from cache")
			c.Data(200, "application/json", []byte(cached))
			return
		}
	}

	rows, err := models.DB.Query(ctx,
		`SELECT id, name, description, category_id, price, stock, 
		        COALESCE(image_url, ''), COALESCE(cloudinary_id, ''),
		        COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), 
		        COALESCE(is_buy1get1, false), is_active, created_at, updated_at
		 FROM products
		 WHERE is_active = TRUE
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		log.Printf("Error querying products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to retrieve products"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.CategoryID,
			&p.Price, &p.Stock, &p.ImageURL, &p.CloudinaryID,
			&p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning product: %v", err)
			continue
		}
		products = append(products, p)
	}

	var total int
	if err := models.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM products WHERE is_active = TRUE`,
	).Scan(&total); err != nil {
		log.Printf("Error counting products: %v", err)
		total = len(products)
	}

	response := ctrl.buildProductResponse(c, "Products retrieved successfully", products, page, limit, total)

	if models.RedisClient != nil {
		jsonData, _ := json.Marshal(response)
		models.RedisClient.Set(ctx, cacheKey, jsonData, 5*time.Minute)
	}

	c.JSON(200, response)
}

// @Summary Filter products
// @Tags Products
// @Produce json
// @Param search query string false "Search by name"
// @Param category_id query int false "Category ID"
// @Param min_price query int false "Minimum price"
// @Param max_price query int false "Maximum price"
// @Param is_flash_sale query bool false "Flash sale only"
// @Param is_favorite query bool false "Favorites only"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} models.HATEOASResponse
// @Router /products/filter [get]
func (ctrl *ProductController) FilterProducts(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)
	ctx := context.Background()

	cacheKey := getProductCacheKey(page, limit, c.Request.URL.Query())

	if models.RedisClient != nil {
		cached, _ := models.RedisClient.Get(ctx, cacheKey).Result()
		if cached != "" {
			c.Data(200, "application/json", []byte(cached))
			return
		}
	}

	query := `SELECT id, name, description, category_id, price, stock, 
	                 COALESCE(image_url, ''), COALESCE(cloudinary_id, ''),
	                 COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), 
	                 COALESCE(is_buy1get1, false), is_active, created_at, updated_at
	          FROM products WHERE is_active = TRUE`

	args := []interface{}{}
	argIdx := 1

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if categoryID, err := strconv.Atoi(c.Query("category_id")); err == nil && categoryID > 0 {
		query += fmt.Sprintf(" AND category_id = $%d", argIdx)
		args = append(args, categoryID)
		argIdx++
	}

	if minPrice, err := strconv.Atoi(c.Query("min_price")); err == nil && minPrice > 0 {
		query += fmt.Sprintf(" AND price >= $%d", argIdx)
		args = append(args, minPrice)
		argIdx++
	}
	if maxPrice, err := strconv.Atoi(c.Query("max_price")); err == nil && maxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", argIdx)
		args = append(args, maxPrice)
		argIdx++
	}

	if isFlashSale, err := strconv.ParseBool(c.Query("is_flash_sale")); err == nil && isFlashSale {
		query += fmt.Sprintf(" AND is_flash_sale = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if isFavorite, err := strconv.ParseBool(c.Query("is_favorite")); err == nil && isFavorite {
		query += fmt.Sprintf(" AND is_favorite = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := models.DB.Query(ctx, query, args...)
	if err != nil {
		log.Printf("Error filtering products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to filter products"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID,
			&p.Price, &p.Stock, &p.ImageURL, &p.CloudinaryID,
			&p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		products = append(products, p)
	}

	countQuery := strings.Replace(query, "SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(cloudinary_id, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at", "SELECT COUNT(*)", 1)
	countQuery = countQuery[:strings.Index(countQuery, "LIMIT")]

	var total int
	models.DB.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total)

	response := ctrl.buildProductResponse(c, "Products filtered successfully", products, page, limit, total)

	if models.RedisClient != nil {
		jsonData, _ := json.Marshal(response)
		models.RedisClient.Set(ctx, cacheKey, jsonData, 5*time.Minute)
	}

	c.JSON(200, response)
}

// @Summary Get favorite products
// @Tags Products
// @Produce json
// @Success 200 {object} models.Response
// @Router /products/favorite [get]
func (ctrl *ProductController) GetFavoriteProducts(c *gin.Context) {
	ctx := context.Background()

	rows, err := models.DB.Query(ctx,
		`SELECT id, name, description, category_id, price, stock, 
		        COALESCE(image_url, ''), COALESCE(cloudinary_id, ''),
		        COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), 
		        COALESCE(is_buy1get1, false), is_active, created_at, updated_at
		 FROM products
		 WHERE is_active = TRUE AND is_favorite = TRUE
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to retrieve favorites"})
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID,
			&p.Price, &p.Stock, &p.ImageURL, &p.CloudinaryID,
			&p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		products = append(products, p)
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Favorite products retrieved",
		"data":    products,
	})
}

// @Summary Get product by ID
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.ErrorResponse
// @Router /products/{id} [get]
func (ctrl *ProductController) GetProductByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ctx := context.Background()

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	var p models.Product
	err := models.DB.QueryRow(ctx,
		`SELECT id, name, description, category_id, price, stock, 
		        COALESCE(image_url, ''), COALESCE(cloudinary_id, ''),
		        COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), 
		        COALESCE(is_buy1get1, false), is_active, created_at, updated_at
		 FROM products WHERE id = $1 AND is_active = TRUE`,
		id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID,
		&p.Price, &p.Stock, &p.ImageURL, &p.CloudinaryID,
		&p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Product retrieved",
		"data":    p,
	})
}

// @Summary Create product
// @Description Create new product (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Product name"
// @Param description formData string false "Description"
// @Param category_id formData int true "Category ID"
// @Param price formData int true "Price"
// @Param stock formData int true "Stock"
// @Param is_flash_sale formData bool false "Flash sale"
// @Param is_favorite formData bool false "Favorite"
// @Param is_buy1get1 formData bool false "Buy 1 Get 1"
// @Param image formData file false "Product image"
// @Success 201 {object} models.Response
// @Router /admin/products [post]
func (ctrl *ProductController) CreateProduct(c *gin.Context) {
	ctx := context.Background()

	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	categoryID, _ := strconv.Atoi(c.PostForm("category_id"))
	price, _ := strconv.Atoi(c.PostForm("price"))
	stock, _ := strconv.Atoi(c.PostForm("stock"))
	isFlashSale, _ := strconv.ParseBool(c.DefaultPostForm("is_flash_sale", "false"))
	isFavorite, _ := strconv.ParseBool(c.DefaultPostForm("is_favorite", "false"))
	isBuy1Get1, _ := strconv.ParseBool(c.DefaultPostForm("is_buy1get1", "false"))

	if len(name) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Product name must be at least 3 characters"})
		return
	}
	if categoryID <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid category_id"})
		return
	}
	if price < 1000 {
		c.JSON(400, gin.H{"success": false, "message": "Price must be at least 1000"})
		return
	}
	if stock < 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid stock"})
		return
	}

	var imageURL, cloudinaryID string
	uploadedFile, fileHeader, fileErr := c.Request.FormFile("image")

	if fileErr == nil {
		defer uploadedFile.Close()

		cloudinaryService, err := models.NewCloudinaryService()
		if err != nil {
			log.Printf("Cloudinary not configured: %v", err)
			c.JSON(500, gin.H{"success": false, "message": "Image upload service not available"})
			return
		}

		if err := cloudinaryService.ValidateImageFile(fileHeader); err != nil {
			c.JSON(400, gin.H{"success": false, "message": err.Error()})
			return
		}

		imageURL, cloudinaryID, err = cloudinaryService.UploadImage(ctx, uploadedFile, fileHeader.Filename, "products")
		if err != nil {
			log.Printf("Upload failed: %v", err)
			c.JSON(500, gin.H{"success": false, "message": "Failed to upload image"})
			return
		}
	}

	now := time.Now()
	var productID int

	err := models.DB.QueryRow(ctx,
		`INSERT INTO products 
		 (name, description, category_id, price, stock, image_url, cloudinary_id, 
		  is_flash_sale, is_favorite, is_buy1get1, is_active, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE, $11, $12) 
		 RETURNING id`,
		name, description, categoryID, price, stock, imageURL, cloudinaryID,
		isFlashSale, isFavorite, isBuy1Get1, now, now,
	).Scan(&productID)

	if err != nil {
		if cloudinaryID != "" {
			cloudinaryService, _ := models.NewCloudinaryService()
			if cloudinaryService != nil {
				cloudinaryService.DeleteImage(ctx, cloudinaryID)
			}
		}
		log.Printf("Insert failed: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to create product"})
		return
	}

	invalidateProductCache()

	c.JSON(201, gin.H{
		"success": true,
		"message": "Product created successfully",
		"data": gin.H{
			"id":            productID,
			"name":          name,
			"price":         price,
			"stock":         stock,
			"image_url":     imageURL,
			"cloudinary_id": cloudinaryID,
		},
	})
}

// @Summary Update product
// @Description Update product (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Product ID"
// @Param name formData string false "Product name"
// @Param description formData string false "Description"
// @Param category_id formData int false "Category ID"
// @Param price formData int false "Price"
// @Param stock formData int false "Stock"
// @Param is_flash_sale formData bool false "Flash sale"
// @Param is_favorite formData bool false "Favorite"
// @Param is_buy1get1 formData bool false "Buy 1 Get 1"
// @Param is_active formData bool false "Active status"
// @Param image formData file false "Product image"
// @Success 200 {object} models.Response
// @Router /admin/products/{id} [patch]
func (ctrl *ProductController) UpdateProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ctx := context.Background()

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	var existing models.Product
	err := models.DB.QueryRow(ctx,
		`SELECT name, description, category_id, price, stock, 
		        COALESCE(image_url, ''), COALESCE(cloudinary_id, ''),
		        COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), 
		        COALESCE(is_buy1get1, false), is_active
		 FROM products WHERE id = $1`,
		id,
	).Scan(&existing.Name, &existing.Description, &existing.CategoryID,
		&existing.Price, &existing.Stock, &existing.ImageURL, &existing.CloudinaryID,
		&existing.IsFlashSale, &existing.IsFavorite, &existing.IsBuy1Get1, &existing.IsActive)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	name := strings.TrimSpace(c.DefaultPostForm("name", existing.Name))
	description := strings.TrimSpace(c.DefaultPostForm("description", existing.Description))
	categoryID, _ := strconv.Atoi(c.DefaultPostForm("category_id", strconv.Itoa(existing.CategoryID)))
	price, _ := strconv.Atoi(c.DefaultPostForm("price", strconv.Itoa(existing.Price)))
	stock, _ := strconv.Atoi(c.DefaultPostForm("stock", strconv.Itoa(existing.Stock)))
	isFlashSale, _ := strconv.ParseBool(c.DefaultPostForm("is_flash_sale", strconv.FormatBool(existing.IsFlashSale)))
	isFavorite, _ := strconv.ParseBool(c.DefaultPostForm("is_favorite", strconv.FormatBool(existing.IsFavorite)))
	isBuy1Get1, _ := strconv.ParseBool(c.DefaultPostForm("is_buy1get1", strconv.FormatBool(existing.IsBuy1Get1)))
	isActive, _ := strconv.ParseBool(c.DefaultPostForm("is_active", strconv.FormatBool(existing.IsActive)))

	if len(name) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Product name must be at least 3 characters"})
		return
	}
	if categoryID <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid category_id"})
		return
	}
	if price < 1000 {
		c.JSON(400, gin.H{"success": false, "message": "Price must be at least 1000"})
		return
	}
	if stock < 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid stock"})
		return
	}

	imageURL := existing.ImageURL
	cloudinaryID := existing.CloudinaryID

	uploadedFile, fileHeader, fileErr := c.Request.FormFile("image")
	if fileErr == nil {
		defer uploadedFile.Close()

		cloudinaryService, err := models.NewCloudinaryService()
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Image upload service not available"})
			return
		}

		if err := cloudinaryService.ValidateImageFile(fileHeader); err != nil {
			c.JSON(400, gin.H{"success": false, "message": err.Error()})
			return
		}

		newImageURL, newCloudinaryID, err := cloudinaryService.UploadImage(ctx, uploadedFile, fileHeader.Filename, "products")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to upload image"})
			return
		}

		if existing.CloudinaryID != "" {
			cloudinaryService.DeleteImage(ctx, existing.CloudinaryID)
		}

		imageURL = newImageURL
		cloudinaryID = newCloudinaryID
	}

	now := time.Now()
	_, err = models.DB.Exec(ctx,
		`UPDATE products 
		 SET name=$1, description=$2, category_id=$3, price=$4, stock=$5, 
		     image_url=$6, cloudinary_id=$7, is_flash_sale=$8, is_favorite=$9, 
		     is_buy1get1=$10, is_active=$11, updated_at=$12 
		 WHERE id=$13`,
		name, description, categoryID, price, stock, imageURL, cloudinaryID,
		isFlashSale, isFavorite, isBuy1Get1, isActive, now, id,
	)

	if err != nil {
		log.Printf("Update failed: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to update product"})
		return
	}

	invalidateProductCache()

	c.JSON(200, gin.H{
		"success": true,
		"message": "Product updated successfully",
		"data": gin.H{
			"id":            id,
			"name":          name,
			"price":         price,
			"stock":         stock,
			"image_url":     imageURL,
			"cloudinary_id": cloudinaryID,
		},
	})
}

// @Summary Delete product
// @Description Delete product permanently (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} models.Response
// @Router /admin/products/{id} [delete]
func (ctrl *ProductController) DeleteProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ctx := context.Background()

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	var cloudinaryID string
	err := models.DB.QueryRow(ctx,
		"SELECT COALESCE(cloudinary_id, '') FROM products WHERE id=$1", id,
	).Scan(&cloudinaryID)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	_, err = models.DB.Exec(ctx, "DELETE FROM products WHERE id=$1", id)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to delete product"})
		return
	}

	if cloudinaryID != "" {
		cloudinaryService, _ := models.NewCloudinaryService()
		if cloudinaryService != nil {
			cloudinaryService.DeleteImage(ctx, cloudinaryID)
		}
	}

	invalidateProductCache()

	c.JSON(200, gin.H{
		"success": true,
		"message": "Product deleted permanently",
	})
}
