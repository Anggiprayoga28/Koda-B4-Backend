package controllers

import (
	"coffee-shop/models"
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

type CategoryController struct{}

// @Summary Get all categories
// @Description Get list of all categories
// @Tags categories
// @Produce json
// @Success 200 {object} models.Response
// @Router /categories [get]
func (ctrl *CategoryController) GetCategories(c *gin.Context) {
	rows, _ := models.DB.Query(context.Background(),
		"SELECT id, name, created_at FROM categories ORDER BY id")
	defer rows.Close()

	categories := []gin.H{}
	for rows.Next() {
		var cat models.Category
		rows.Scan(&cat.ID, &cat.Name, &cat.CreatedAt)
		categories = append(categories, gin.H{
			"id":         cat.ID,
			"name":       cat.Name,
			"created_at": cat.CreatedAt,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Categories retrieved",
		"data":    categories,
	})
}

// @Summary Get category by ID
// @Description Get a single category by ID
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.ErrorResponse
// @Router /categories/{id} [get]
func (ctrl *CategoryController) GetCategoryByID(c *gin.Context) {
	id := c.Param("id")

	var cat models.Category
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, name, created_at FROM categories WHERE id=$1",
		id).Scan(&cat.ID, &cat.Name, &cat.CreatedAt)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Category not found"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Category retrieved",
		"data":    cat,
	})
}

// @Summary Create new category
// @Description Create a new category (Admin only)
// @Tags categories
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Category name"
// @Security BearerAuth
// @Success 201 {object} models.Response
// @Router /admin/categories [post]
func (ctrl *CategoryController) CreateCategory(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))

	if name == "" {
		c.JSON(400, gin.H{"success": false, "message": "Name is required"})
		return
	}

	if len(name) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Category name must be at least 3 characters"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM categories WHERE name=$1", name).Scan(&exists)
	if exists > 0 {
		c.JSON(400, gin.H{"success": false, "message": "Category name already exists"})
		return
	}

	var categoryID int
	err := models.DB.QueryRow(context.Background(),
		"INSERT INTO categories (name) VALUES ($1) RETURNING id",
		name).Scan(&categoryID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to create category"})
		return
	}

	c.JSON(201, gin.H{
		"success": true,
		"message": "Category created successfully",
		"data": gin.H{
			"id":   categoryID,
			"name": name,
		},
	})
}

// @Summary Update category
// @Description Update an existing category (Admin only)
// @Tags categories
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Category ID"
// @Param name formData string true "Category name"
// @Security BearerAuth
// @Success 200 {object} models.Response
// @Router /admin/categories/{id} [patch]
func (ctrl *CategoryController) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	name := strings.TrimSpace(c.PostForm("name"))

	if name == "" {
		c.JSON(400, gin.H{"success": false, "message": "Name is required"})
		return
	}

	if len(name) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Category name must be at least 3 characters"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM categories WHERE id=$1", id).Scan(&exists)
	if exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "Category not found"})
		return
	}

	var nameExists int
	models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM categories WHERE name=$1 AND id!=$2", name, id).Scan(&nameExists)
	if nameExists > 0 {
		c.JSON(400, gin.H{"success": false, "message": "Category name already exists"})
		return
	}

	_, err := models.DB.Exec(context.Background(),
		"UPDATE categories SET name=$1 WHERE id=$2", name, id)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to update category"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Category updated successfully",
	})
}

// @Summary Delete category
// @Description Delete a category (Admin only)
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Security BearerAuth
// @Success 200 {object} models.Response
// @Router /admin/categories/{id} [delete]
func (ctrl *CategoryController) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	var exists int
	models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM categories WHERE id=$1", id).Scan(&exists)
	if exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "Category not found"})
		return
	}

	_, err := models.DB.Exec(context.Background(),
		"DELETE FROM categories WHERE id=$1", id)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to delete category"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Category deleted successfully",
	})
}
