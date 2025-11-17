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
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > 100 {
		limit = 100
	}

	offset = (page - 1) * limit
	return page, limit, offset
}

func (ctrl *ProductController) generateProductLinks(c *gin.Context, page, limit, totalPages int) models.PaginationLinks {
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

	links := models.PaginationLinks{
		Self: makeURL(page),
	}

	if page > 1 {
		prevURL := makeURL(page - 1)
		links.Prev = &prevURL
	}

	if page < totalPages {
		nextURL := makeURL(page + 1)
		links.Next = &nextURL
	}

	return links
}

func (ctrl *ProductController) buildProductResponse(c *gin.Context, message string, data interface{}, page, limit, totalItems int) models.HATEOASResponse {
	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + limit - 1) / limit
	}

	if page > totalPages && totalPages > 0 {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	meta := models.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	links := ctrl.generateProductLinks(c, page, limit, totalPages)

	return models.HATEOASResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
		Links:   links,
	}
}

// @Summary Get all categories
// @Description Get list of all categories
// @Tags Categories
// @Produce json
// @Success 200 {object} models.Response
// @Router /categories [get]
func (ctrl *ProductController) GetAllCategories(c *gin.Context) {
	rows, err := models.DB.Query(context.Background(), "SELECT id, name, is_active, created_at FROM categories ORDER BY name")
	if err != nil {
		log.Printf("Error querying categories: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to retrieve categories"})
		return
	}
	defer rows.Close()

	categories := []gin.H{}
	for rows.Next() {
		var id int
		var name string
		var isActive bool
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &isActive, &createdAt); err != nil {
			log.Printf("Error scanning category row: %v", err)
			continue
		}
		categories = append(categories, gin.H{"id": id, "name": name, "is_active": isActive, "created_at": createdAt})
	}

	c.JSON(200, gin.H{"success": true, "message": "Categories retrieved", "data": categories})
}

func getProductCacheKey(page, limit int) string {
	return fmt.Sprintf("products_list_p%d_l%d", page, limit)
}

func invalidateProductCache() {
	if models.RedisClient == nil {
		return
	}
	ctx := context.Background()
	iter := models.RedisClient.Scan(ctx, 0, "products_list_*", 0).Iterator()
	for iter.Next(ctx) {
		models.RedisClient.Del(ctx, iter.Val())
	}
}

// @Summary Get all products
// @Description Get paginated list of products
// @Tags Products
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} models.HATEOASResponse
// @Router /products [get]
func (ctrl *ProductController) GetAllProducts(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)

	cacheKey := getProductCacheKey(page, limit)
	ctx := context.Background()

	if models.RedisClient != nil {
		cached, err := models.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			log.Println("Serving products from cache")
			c.Data(200, "application/json", []byte(cached))
			return
		}
		log.Printf("Cache miss or error: %v", err)
	}

	var total int
	err := models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM products WHERE is_active=true").Scan(&total)
	if err != nil {
		log.Printf("Error counting products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to count products"})
		return
	}

	log.Printf("Total products: %d, Page: %d, Limit: %d, Offset: %d", total, page, limit, offset)

	rows, err := models.DB.Query(context.Background(),
		"SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE is_active=true ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)

	if err != nil {
		log.Printf("Error querying products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Failed to retrieve products: %v", err)})
		return
	}
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning product row: %v", err)
			continue
		}

		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v", err)
	}

	log.Printf("Retrieved %d products out of %d total", len(products), total)

	response := ctrl.buildProductResponse(c, "Products retrieved successfully", products, page, limit, total)

	if models.RedisClient != nil {
		jsonData, _ := json.Marshal(response)
		if err := models.RedisClient.Set(ctx, cacheKey, string(jsonData), 5*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache products: %v", err)
		}
	}

	c.JSON(200, response)
}

// @Summary Filter products
// @Description Filter products by search, category, sort, and price range with HATEOAS links
// @Tags Products
// @Produce json
// @Param search query string false "Search by product name"
// @Param category query []string false "Filter by category"
// @Param sort_name query string false "Sort by name" Enums(asc, desc)
// @Param sort_price query string false "Sort by price" Enums(asc, desc)
// @Param sort query string false "Filter by type"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} models.HATEOASResponse
// @Router /products/filter [get]
func (ctrl *ProductController) FilterProducts(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)

	search := strings.TrimSpace(c.Query("search"))
	categories := c.QueryArray("category")
	sortName := c.Query("sort_name")
	sortPrice := c.Query("sort_price")
	sortType := c.Query("sort")
	minPrice, _ := strconv.Atoi(c.Query("min_price"))
	maxPrice, _ := strconv.Atoi(c.Query("max_price"))

	query := "SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE is_active=true"
	countQuery := "SELECT COUNT(*) FROM products WHERE is_active=true"
	args := []interface{}{}
	argCount := 1

	if search != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE $%d", argCount)
		countQuery += fmt.Sprintf(" AND LOWER(name) LIKE $%d", argCount)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argCount++
	}

	if len(categories) > 0 {
		placeholders := []string{}
		for _, cat := range categories {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			args = append(args, cat)
			argCount++
		}
		query += " AND category_id IN (" + strings.Join(placeholders, ",") + ")"
		countQuery += " AND category_id IN (" + strings.Join(placeholders, ",") + ")"
	}

	if sortType != "" {
		switch sortType {
		case "flash_sale":
			query += " AND is_flash_sale=true"
			countQuery += " AND is_flash_sale=true"
		case "favorite":
			query += " AND is_favorite=true"
			countQuery += " AND is_favorite=true"
		case "buy1get1":
			query += " AND is_buy1get1=true"
			countQuery += " AND is_buy1get1=true"
		}
	}

	if minPrice > 0 {
		query += fmt.Sprintf(" AND price >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND price >= $%d", argCount)
		args = append(args, minPrice)
		argCount++
	}

	if maxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND price <= $%d", argCount)
		args = append(args, maxPrice)
		argCount++
	}

	if sortName != "" {
		if sortName == "asc" {
			query += " ORDER BY name ASC"
		} else if sortName == "desc" {
			query += " ORDER BY name DESC"
		}
	} else if sortPrice != "" {
		if sortPrice == "asc" {
			query += " ORDER BY price ASC"
		} else if sortPrice == "desc" {
			query += " ORDER BY price DESC"
		}
	} else {
		query += " ORDER BY created_at DESC"
	}

	var total int
	err := models.DB.QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("Error counting filtered products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to count products"})
		return
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := models.DB.Query(context.Background(), query, args...)
	if err != nil {
		log.Printf("Error querying filtered products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to retrieve products"})
		return
	}
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning product: %v", err)
			continue
		}

		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	response := ctrl.buildProductResponse(c, "Products filtered successfully", products, page, limit, total)
	c.JSON(200, response)
}

// @Summary Get favorite products
// @Description Get list of favorite products
// @Tags Products
// @Produce json
// @Success 200 {object} models.Response
// @Router /products/favorite [get]
func (ctrl *ProductController) GetFavoriteProducts(c *gin.Context) {
	rows, err := models.DB.Query(context.Background(),
		"SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE is_favorite=true AND is_active=true ORDER BY created_at DESC")

	if err != nil {
		log.Printf("Error querying favorite products: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to retrieve favorite products"})
		return
	}
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning product: %v", err)
			continue
		}

		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	c.JSON(200, gin.H{"success": true, "message": "Favorite products retrieved successfully", "data": products})
}

// @Summary Get product by ID
// @Description Get product details by ID
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} models.Response
// @Router /products/{id} [get]
func (ctrl *ProductController) GetProductByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var p models.Product
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE id=$1",
		id).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		log.Printf("Error finding product: %v", err)
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Product retrieved successfully",
		"data": gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		},
	})
}

// @Summary Create product
// @Description Create new product (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Product name"
// @Param description formData string true "Product description"
// @Param category_id formData int true "Category ID"
// @Param price formData int true "Product price"
// @Param stock formData int true "Product stock"
// @Param is_flash_sale formData bool false "Is flash sale"
// @Param is_favorite formData bool false "Is favorite"
// @Param is_buy1get1 formData bool false "Is buy 1 get 1"
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

	var imageURL, cloudinaryPublicID string

	file, fileHeader, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()

		cloudinaryService, err := models.NewCloudinaryService()
		if err != nil {
			log.Printf("Cloudinary not configured, skipping upload: %v", err)
		} else {
			if err := cloudinaryService.ValidateImageFile(fileHeader); err != nil {
				c.JSON(400, gin.H{"success": false, "message": err.Error()})
				return
			}

			imageURL, cloudinaryPublicID, err = cloudinaryService.UploadImage(ctx, file, fileHeader.Filename, "products")
			if err != nil {
				log.Printf("Error uploading to Cloudinary: %v", err)
				c.JSON(500, gin.H{"success": false, "message": "Failed to upload image"})
				return
			}

			log.Printf("Image uploaded to Cloudinary successfully: %s", imageURL)
		}
	} else {
		log.Printf("No image uploaded or error: %v", err)
	}

	now := time.Now()

	var hasCloudinaryColumn bool
	err = models.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='cloudinary_id')").Scan(&hasCloudinaryColumn)

	if err != nil {
		log.Printf("Error checking column existence: %v", err)
		hasCloudinaryColumn = false
	}

	var id int
	if hasCloudinaryColumn {
		insertQuery := `
			INSERT INTO products 
			(name, description, category_id, price, stock, image_url, cloudinary_id, is_flash_sale, is_favorite, is_buy1get1, is_active, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, $11, $12) 
			RETURNING id
		`
		err = models.DB.QueryRow(ctx, insertQuery,
			name, description, categoryID, price, stock, imageURL, cloudinaryPublicID,
			isFlashSale, isFavorite, isBuy1Get1, now, now).Scan(&id)
	} else {
		insertQuery := `
			INSERT INTO products 
			(name, description, category_id, price, stock, image_url, is_flash_sale, is_favorite, is_buy1get1, is_active, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $11) 
			RETURNING id
		`
		err = models.DB.QueryRow(ctx, insertQuery,
			name, description, categoryID, price, stock, imageURL,
			isFlashSale, isFavorite, isBuy1Get1, now, now).Scan(&id)
	}

	if err != nil {
		log.Printf("Error inserting product to database: %v", err)

		if cloudinaryPublicID != "" {
			cloudinaryService, _ := models.NewCloudinaryService()
			if cloudinaryService != nil {
				cloudinaryService.DeleteImage(ctx, cloudinaryPublicID)
			}
		}

		c.JSON(500, gin.H{"success": false, "message": "Failed to create product: " + err.Error()})
		return
	}

	log.Printf("Product created successfully with ID: %d", id)

	invalidateProductCache()

	responseData := gin.H{
		"id":            id,
		"name":          name,
		"description":   description,
		"category_id":   categoryID,
		"price":         price,
		"stock":         stock,
		"image_url":     imageURL,
		"is_flash_sale": isFlashSale,
		"is_favorite":   isFavorite,
		"is_buy1get1":   isBuy1Get1,
		"is_active":     true,
	}

	if hasCloudinaryColumn && cloudinaryPublicID != "" {
		responseData["cloudinary_id"] = cloudinaryPublicID
	}

	c.JSON(201, gin.H{
		"success": true,
		"message": "Product created successfully",
		"data":    responseData,
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
// @Param description formData string false "Product description"
// @Param category_id formData int false "Category ID"
// @Param price formData int false "Product price"
// @Param stock formData int false "Product stock"
// @Param is_flash_sale formData bool false "Is flash sale"
// @Param is_favorite formData bool false "Is favorite"
// @Param is_buy1get1 formData bool false "Is buy 1 get 1"
// @Param is_active formData bool false "Is active"
// @Param image formData file false "Product image"
// @Success 200 {object} models.Response
// @Router /admin/products/{id} [patch]
func (ctrl *ProductController) UpdateProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ctx := context.Background()

	var hasCloudinaryColumn bool
	err := models.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='cloudinary_id')").Scan(&hasCloudinaryColumn)

	if err != nil {
		log.Printf("Error checking column existence: %v", err)
		hasCloudinaryColumn = false
	}

	var existingProduct models.Product

	if hasCloudinaryColumn {
		err = models.DB.QueryRow(ctx,
			"SELECT name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(cloudinary_id, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active FROM products WHERE id=$1",
			id).Scan(&existingProduct.Name, &existingProduct.Description, &existingProduct.CategoryID,
			&existingProduct.Price, &existingProduct.Stock, &existingProduct.ImageURL, &existingProduct.CloudinaryID,
			&existingProduct.IsFlashSale, &existingProduct.IsFavorite, &existingProduct.IsBuy1Get1, &existingProduct.IsActive)
	} else {
		err = models.DB.QueryRow(ctx,
			"SELECT name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active FROM products WHERE id=$1",
			id).Scan(&existingProduct.Name, &existingProduct.Description, &existingProduct.CategoryID,
			&existingProduct.Price, &existingProduct.Stock, &existingProduct.ImageURL,
			&existingProduct.IsFlashSale, &existingProduct.IsFavorite, &existingProduct.IsBuy1Get1, &existingProduct.IsActive)
	}

	if err != nil {
		log.Printf("Error finding product: %v", err)
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	name := strings.TrimSpace(c.DefaultPostForm("name", existingProduct.Name))
	description := strings.TrimSpace(c.DefaultPostForm("description", existingProduct.Description))
	categoryID, _ := strconv.Atoi(c.DefaultPostForm("category_id", strconv.Itoa(existingProduct.CategoryID)))
	price, _ := strconv.Atoi(c.DefaultPostForm("price", strconv.Itoa(existingProduct.Price)))
	stock, _ := strconv.Atoi(c.DefaultPostForm("stock", strconv.Itoa(existingProduct.Stock)))
	isFlashSale, _ := strconv.ParseBool(c.DefaultPostForm("is_flash_sale", strconv.FormatBool(existingProduct.IsFlashSale)))
	isFavorite, _ := strconv.ParseBool(c.DefaultPostForm("is_favorite", strconv.FormatBool(existingProduct.IsFavorite)))
	isBuy1Get1, _ := strconv.ParseBool(c.DefaultPostForm("is_buy1get1", strconv.FormatBool(existingProduct.IsBuy1Get1)))
	isActive, _ := strconv.ParseBool(c.DefaultPostForm("is_active", strconv.FormatBool(existingProduct.IsActive)))

	if name != existingProduct.Name && len(name) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Product name must be at least 3 characters"})
		return
	}

	if categoryID <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid category_id"})
		return
	}

	if price < 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid price"})
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

	imageURL := existingProduct.ImageURL
	cloudinaryPublicID := existingProduct.CloudinaryID

	file, fileHeader, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()

		cloudinaryService, err := models.NewCloudinaryService()
		if err != nil {
			log.Printf("Cloudinary not configured, skipping upload: %v", err)
		} else {
			if err := cloudinaryService.ValidateImageFile(fileHeader); err != nil {
				c.JSON(400, gin.H{"success": false, "message": err.Error()})
				return
			}

			newImageURL, newCloudinaryID, err := cloudinaryService.UploadImage(ctx, file, fileHeader.Filename, "products")
			if err != nil {
				log.Printf("Error uploading to Cloudinary: %v", err)
				c.JSON(500, gin.H{"success": false, "message": "Failed to upload image"})
				return
			}

			if existingProduct.CloudinaryID != "" {
				if err := cloudinaryService.DeleteImage(ctx, existingProduct.CloudinaryID); err != nil {
					log.Printf("Warning: Failed to delete old image from Cloudinary: %v", err)
				}
			}

			imageURL = newImageURL
			cloudinaryPublicID = newCloudinaryID
			log.Printf("Image updated successfully: %s", imageURL)
		}
	}

	if hasCloudinaryColumn {
		_, err = models.DB.Exec(ctx,
			"UPDATE products SET name=$1, description=$2, category_id=$3, price=$4, stock=$5, image_url=$6, cloudinary_id=$7, is_flash_sale=$8, is_favorite=$9, is_buy1get1=$10, is_active=$11, updated_at=$12 WHERE id=$13",
			name, description, categoryID, price, stock, imageURL, cloudinaryPublicID, isFlashSale, isFavorite, isBuy1Get1, isActive, time.Now(), id)
	} else {
		_, err = models.DB.Exec(ctx,
			"UPDATE products SET name=$1, description=$2, category_id=$3, price=$4, stock=$5, image_url=$6, is_flash_sale=$7, is_favorite=$8, is_buy1get1=$9, is_active=$10, updated_at=$11 WHERE id=$12",
			name, description, categoryID, price, stock, imageURL, isFlashSale, isFavorite, isBuy1Get1, isActive, time.Now(), id)
	}

	if err != nil {
		log.Printf("Error updating product: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to update product"})
		return
	}

	invalidateProductCache()

	c.JSON(200, gin.H{"success": true, "message": "Product updated successfully"})
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

	var hasCloudinaryColumn bool
	err := models.DB.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='cloudinary_id')").Scan(&hasCloudinaryColumn)

	if err != nil {
		log.Printf("Error checking column existence: %v", err)
		hasCloudinaryColumn = false
	}

	var cloudinaryPublicID string

	if hasCloudinaryColumn {
		err = models.DB.QueryRow(ctx,
			"SELECT COALESCE(cloudinary_id, '') FROM products WHERE id=$1", id).Scan(&cloudinaryPublicID)
	} else {
		err = models.DB.QueryRow(ctx,
			"SELECT id FROM products WHERE id=$1", id).Scan(&id)
	}

	if err != nil {
		log.Printf("Error finding product to delete: %v", err)
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	_, err = models.DB.Exec(ctx, "DELETE FROM products WHERE id=$1", id)
	if err != nil {
		log.Printf("Error deleting product: %v", err)
		c.JSON(500, gin.H{"success": false, "message": "Failed to delete product"})
		return
	}

	if cloudinaryPublicID != "" {
		cloudinaryService, err := models.NewCloudinaryService()
		if err == nil {
			if err := cloudinaryService.DeleteImage(ctx, cloudinaryPublicID); err != nil {
				log.Printf("Warning: Failed to delete image from Cloudinary: %v", err)
			}
		}
	}

	invalidateProductCache()

	c.JSON(200, gin.H{"success": true, "message": "Product deleted permanently"})
}
