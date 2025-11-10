package controllers

import (
	"coffee-shop/models"
	"context"
	"math"
	"strconv"
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
	offset := (page - 1) * limit

	var total int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM products WHERE is_active=true").Scan(&total)

	rows, _ := models.DB.Query(context.Background(),
		"SELECT id, name, description, category_id, price, stock, is_active, created_at, updated_at FROM products WHERE is_active=true ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	defer rows.Close()

	products := []gin.H{}
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		products = append(products, gin.H{
			"id": p.ID, "name": p.Name, "description": p.Description,
			"category_id": p.CategoryID, "price": p.Price, "stock": p.Stock,
			"is_active": p.IsActive, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		})
	}

	c.JSON(200, gin.H{
		"success": true, "message": "Products retrieved", "data": products,
		"meta": gin.H{
			"page": page, "limit": limit, "total_items": total,
			"total_pages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
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
		"SELECT id, name, description, category_id, price, stock, is_active, created_at, updated_at FROM products WHERE id=$1",
		id).Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)

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
// @Accept json
// @Produce json
// @Param request body models.CreateProductRequest true "Product Request"
// @Success 201 {object} models.Response
// @Router /admin/products [post]
func (ctrl *ProductController) CreateProduct(c *gin.Context) {
	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	now := time.Now()
	var id int
	models.DB.QueryRow(context.Background(),
		"INSERT INTO products (name, description, category_id, price, stock, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,true,$6,$7) RETURNING id",
		req.Name, req.Description, req.CategoryID, req.Price, req.Stock, now, now).Scan(&id)

	c.JSON(201, gin.H{
		"success": true, "message": "Product created",
		"data": gin.H{
			"id": id, "name": req.Name, "description": req.Description,
			"category_id": req.CategoryID, "price": req.Price, "stock": req.Stock,
		},
	})
}

// @Summary Update product
// @Description Update product (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param request body models.UpdateProductRequest true "Product Request"
// @Success 200 {object} models.Response
// @Router /admin/products/{id} [patch]
func (ctrl *ProductController) UpdateProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req models.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	models.DB.Exec(context.Background(),
		"UPDATE products SET name=$1, description=$2, category_id=$3, price=$4, stock=$5, is_active=$6, updated_at=$7 WHERE id=$8",
		req.Name, req.Description, req.CategoryID, req.Price, req.Stock, req.IsActive, time.Now(), id)

	c.JSON(200, gin.H{"success": true, "message": "Product updated"})
}

// @Summary Delete product
// @Description Soft delete product (Admin)
// @Tags Admin - Products
// @Security BearerAuth
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} models.Response
// @Router /admin/products/{id} [delete]
func (ctrl *ProductController) DeleteProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	models.DB.Exec(context.Background(), "UPDATE products SET is_active=false WHERE id=$1", id)
	c.JSON(200, gin.H{"success": true, "message": "Product deleted"})
}
