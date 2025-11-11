package controllers

import (
	"coffee-shop/models"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ProductController struct{}

// @Summary Get all categories
// @Description Get list of all categories
// @Tags Categories
// @Produce json
// @Success 200 {object} models.Response
// @Router /categories [get]
func (ctrl *ProductController) GetAllCategories(c *gin.Context) {
	rows, _ := models.DB.Query(context.Background(), "SELECT id, name, is_active, created_at FROM categories ORDER BY name")
	defer rows.Close()

	categories := []gin.H{}
	for rows.Next() {
		var id int
		var name string
		var isActive bool
		var createdAt time.Time
		rows.Scan(&id, &name, &isActive, &createdAt)
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
// @Success 200 {object} models.PaginationResponse
// @Router /products [get]
func (ctrl *ProductController) GetAllProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	cacheKey := getProductCacheKey(page, limit)
	ctx := context.Background()

	if models.RedisClient != nil {
		cached, err := models.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			c.Data(200, "application/json", []byte(cached))
			return
		}
	}

	offset := (page - 1) * limit

	var total int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM products WHERE is_active=true").Scan(&total)

	rows, _ := models.DB.Query(context.Background(),
		"SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE is_active=true ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	response := gin.H{
		"success": true, "message": "Products retrieved", "data": products,
		"meta": gin.H{
			"page": page, "limit": limit, "total_items": total,
			"total_pages": int(math.Ceil(float64(total) / float64(limit))),
		},
	}

	if models.RedisClient != nil {
		jsonData, _ := json.Marshal(response)
		models.RedisClient.Set(ctx, cacheKey, string(jsonData), 5*time.Minute)
	}

	c.JSON(200, response)
}

// @Summary Filter products
// @Description Filter products by search, category, sort, and price range
// @Tags Products
// @Produce json
// @Param search query string false "Search by product name"
// @Param category query []string false "Filter by category"
// @Param sort_name query string false "Sort by name" Enums(asc, desc)
// @Param sort_price query string false "Sort by price" Enums(asc, desc)
// @Param sort query string false "Filter by type"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Success 200 {object} models.Response
// @Router /products/filter [get]
func (ctrl *ProductController) FilterProducts(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	category := strings.TrimSpace(c.Query("category"))
	sortBy := strings.TrimSpace(c.Query("sort"))
	sortName := strings.TrimSpace(c.Query("sort_name"))
	sortPrice := strings.TrimSpace(c.Query("sort_price"))
	minPrice, _ := strconv.Atoi(c.Query("min_price"))
	maxPrice, _ := strconv.Atoi(c.Query("max_price"))

	query := "SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE is_active=true"
	args := []interface{}{}
	paramIndex := 1

	if search != "" {
		query += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", paramIndex)
		args = append(args, "%"+search+"%")
		paramIndex++
	}

	if category != "" {
		if category == "favorite" {
			query += " AND is_favorite=true"
		} else {
			query += fmt.Sprintf(" AND category_id IN (SELECT id FROM categories WHERE LOWER(name)=LOWER($%d))", paramIndex)
			args = append(args, category)
			paramIndex++
		}
	}

	if minPrice > 0 {
		query += fmt.Sprintf(" AND price >= $%d", paramIndex)
		args = append(args, minPrice)
		paramIndex++
	}

	if maxPrice > 0 {
		query += fmt.Sprintf(" AND price <= $%d", paramIndex)
		args = append(args, maxPrice)
		paramIndex++
	}

	if sortBy == "buy1get1" {
		query += " AND is_buy1get1=true"
	} else if sortBy == "flash_sale" {
		query += " AND is_flash_sale=true"
	}

	orderBy := ""
	if sortName != "" {
		if sortName == "asc" {
			orderBy = " ORDER BY name ASC"
		} else if sortName == "desc" {
			orderBy = " ORDER BY name DESC"
		}
	} else if sortPrice != "" {
		if sortPrice == "asc" {
			orderBy = " ORDER BY price ASC"
		} else if sortPrice == "desc" {
			orderBy = " ORDER BY price DESC"
		}
	} else {
		orderBy = " ORDER BY created_at DESC"
	}

	query += orderBy

	rows, _ := models.DB.Query(context.Background(), query, args...)
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Products filtered",
		"data":    products,
		"total":   len(products),
	})
}

// @Summary Get favorite products
// @Description Get list of favorite products (limited to 4)
// @Tags Products
// @Produce json
// @Success 200 {object} models.Response
// @Router /products/favorite [get]
func (ctrl *ProductController) GetFavoriteProducts(c *gin.Context) {
	rows, _ := models.DB.Query(context.Background(),
		"SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE is_active=true AND is_favorite=true ORDER BY created_at DESC LIMIT 4")
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"image_url": p.ImageURL, "is_flash_sale": p.IsFlashSale,
			"is_favorite": p.IsFavorite, "is_buy1get1": p.IsBuy1Get1,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	c.JSON(200, gin.H{"success": true, "message": "Favorite products retrieved", "data": products})
}

// @Summary Get product by ID
// @Description Get product details
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.ErrorResponse
// @Router /products/{id} [get]
func (ctrl *ProductController) GetProductByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var p models.Product
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active, created_at, updated_at FROM products WHERE id=$1",
		id).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL, &p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Product retrieved", "data": p})
}

// @Summary Create product
// @Description Create new product (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Product name"
// @Param description formData string false "Product description"
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
	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	categoryIDStr := c.PostForm("category_id")
	priceStr := c.PostForm("price")
	stockStr := c.PostForm("stock")
	isFlashSale, _ := strconv.ParseBool(c.DefaultPostForm("is_flash_sale", "false"))
	isFavorite, _ := strconv.ParseBool(c.DefaultPostForm("is_favorite", "false"))
	isBuy1Get1, _ := strconv.ParseBool(c.DefaultPostForm("is_buy1get1", "false"))

	if name == "" || categoryIDStr == "" || priceStr == "" {
		c.JSON(400, gin.H{"success": false, "message": "Name, category_id, and price are required"})
		return
	}

	if len(name) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Product name must be at least 3 characters"})
		return
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil || categoryID <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid category_id"})
		return
	}

	price, err := strconv.Atoi(priceStr)
	if err != nil || price < 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid price"})
		return
	}

	if price < 1000 {
		c.JSON(400, gin.H{"success": false, "message": "Price must be at least 1000"})
		return
	}

	stock := 0
	if stockStr != "" {
		stock, err = strconv.Atoi(stockStr)
		if err != nil || stock < 0 {
			c.JSON(400, gin.H{"success": false, "message": "Invalid stock"})
			return
		}
	}

	var categoryExists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM categories WHERE id=$1", categoryID).Scan(&categoryExists)
	if categoryExists == 0 {
		c.JSON(400, gin.H{"success": false, "message": "Category not found"})
		return
	}

	imageURL := ""
	file, err := c.FormFile("image")
	if err == nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
		if !allowedExts[ext] {
			c.JSON(400, gin.H{"success": false, "message": "Invalid file type. Only jpg, jpeg, png, gif, webp allowed"})
			return
		}

		if file.Size > 5*1024*1024 {
			c.JSON(400, gin.H{"success": false, "message": "File size too large. Maximum 5MB"})
			return
		}

		uploadDir := "./uploads/products"
		os.MkdirAll(uploadDir, os.ModePerm)

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to save image: " + err.Error()})
			return
		}
		imageURL = "/uploads/products/" + filename
	}

	now := time.Now()
	var id int
	err = models.DB.QueryRow(context.Background(),
		"INSERT INTO products (name, description, category_id, price, stock, image_url, is_flash_sale, is_favorite, is_buy1get1, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true,$10,$11) RETURNING id",
		name, description, categoryID, price, stock, imageURL, isFlashSale, isFavorite, isBuy1Get1, now, now).Scan(&id)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to create product: " + err.Error()})
		return
	}

	invalidateProductCache()

	c.JSON(201, gin.H{
		"success": true, "message": "Product created successfully",
		"data": gin.H{
			"id": id, "name": name, "description": description,
			"category_id": categoryID, "price": price, "stock": stock,
			"image_url": imageURL, "is_flash_sale": isFlashSale,
			"is_favorite": isFavorite, "is_buy1get1": isBuy1Get1,
			"is_active": true,
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

	var existingProduct models.Product
	err := models.DB.QueryRow(context.Background(),
		"SELECT name, description, category_id, price, stock, COALESCE(image_url, ''), COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), is_active FROM products WHERE id=$1",
		id).Scan(&existingProduct.Name, &existingProduct.Description, &existingProduct.CategoryID,
		&existingProduct.Price, &existingProduct.Stock, &existingProduct.ImageURL,
		&existingProduct.IsFlashSale, &existingProduct.IsFavorite, &existingProduct.IsBuy1Get1, &existingProduct.IsActive)

	if err != nil {
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
	file, err := c.FormFile("image")
	if err == nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
		if !allowedExts[ext] {
			c.JSON(400, gin.H{"success": false, "message": "Invalid file type. Only jpg, jpeg, png, gif, webp allowed"})
			return
		}

		if file.Size > 5*1024*1024 {
			c.JSON(400, gin.H{"success": false, "message": "File size too large. Maximum 5MB"})
			return
		}

		uploadDir := "./uploads/products"
		os.MkdirAll(uploadDir, os.ModePerm)

		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, savePath); err == nil {
			if existingProduct.ImageURL != "" {
				oldPath := "." + existingProduct.ImageURL
				os.Remove(oldPath)
			}
			imageURL = "/uploads/products/" + filename
		}
	}

	_, err = models.DB.Exec(context.Background(),
		"UPDATE products SET name=$1, description=$2, category_id=$3, price=$4, stock=$5, image_url=$6, is_flash_sale=$7, is_favorite=$8, is_buy1get1=$9, is_active=$10, updated_at=$11 WHERE id=$12",
		name, description, categoryID, price, stock, imageURL, isFlashSale, isFavorite, isBuy1Get1, isActive, time.Now(), id)

	if err != nil {
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

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	var imageURL string
	err := models.DB.QueryRow(context.Background(),
		"SELECT COALESCE(image_url, '') FROM products WHERE id=$1", id).Scan(&imageURL)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	_, err = models.DB.Exec(context.Background(), "DELETE FROM products WHERE id=$1", id)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to delete product"})
		return
	}

	if imageURL != "" {
		oldPath := "." + imageURL
		os.Remove(oldPath)
	}

	invalidateProductCache()

	c.JSON(200, gin.H{"success": true, "message": "Product deleted permanently"})
}
