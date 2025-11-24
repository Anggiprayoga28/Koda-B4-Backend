package controllers

import (
	"coffee-shop/models"
	"context"

	"github.com/gin-gonic/gin"
)

type PromoController struct{}

func (ctrl *PromoController) GetAllPromos(c *gin.Context) {
	rows, err := models.DB.Query(context.Background(),
		"SELECT id, title, description, code, bg_color, text_color FROM promos WHERE is_active=true ORDER BY created_at DESC")
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to get promos",
		})
		return
	}
	defer rows.Close()

	promos := []gin.H{}
	for rows.Next() {
		var id int
		var title, description, code, bgColor, textColor string

		if err := rows.Scan(&id, &title, &description, &code, &bgColor, &textColor); err != nil {
			continue
		}

		promos = append(promos, gin.H{
			"id":          id,
			"title":       title,
			"description": description,
			"code":        code,
			"bgColor":     bgColor,
			"textColor":   textColor,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Promos retrieved",
		"data":    promos,
	})
}
