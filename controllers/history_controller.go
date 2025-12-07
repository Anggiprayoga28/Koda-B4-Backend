package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type HistoryController struct{}

// @Summary Get order history
// @Description Get paginated order history with aggregated product images
// @Tags History
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Param start_date query string false "Filter by start date (format: 2006-01-02)"
// @Param end_date query string false "Filter by end date (format: 2006-01-02)"
// @Param month query string false "Filter by month (format: January 2023)"
// @Success 200 {object} models.PaginationResponse
// @Router /history [get]
func (ctrl *HistoryController) GetHistory(c *gin.Context) {
	userID := c.GetInt("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "4"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 4
	}
	offset := (page - 1) * limit

	statusFilter := strings.TrimSpace(c.Query("status"))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))
	monthFilter := strings.TrimSpace(c.Query("month"))

	whereConditions := []string{"o.user_id = $1"}
	args := []interface{}{userID}
	paramIndex := 2

	if statusFilter != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("os.name = $%d", paramIndex))
		args = append(args, statusFilter)
		paramIndex++
	}

	if startDate != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("DATE(o.order_date) >= $%d", paramIndex))
		args = append(args, startDate)
		paramIndex++
	}

	if endDate != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("DATE(o.order_date) <= $%d", paramIndex))
		args = append(args, endDate)
		paramIndex++
	}

	if monthFilter != "" {
		t, err := time.Parse("January 2006", monthFilter)
		if err == nil {
			year, month, _ := t.Date()
			firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
			lastDay := firstDay.AddDate(0, 1, -1)

			whereConditions = append(whereConditions,
				fmt.Sprintf("DATE(o.order_date) >= $%d AND DATE(o.order_date) <= $%d", paramIndex, paramIndex+1))
			args = append(args, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"))
			paramIndex += 2
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT o.id)
		FROM orders o
		JOIN order_status os ON o.status_id = os.id
		WHERE %s
	`, whereClause)

	var total int
	err := models.DB.QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": fmt.Sprintf("Count error: %v", err),
		})
		return
	}

	query := fmt.Sprintf(`
		SELECT 
			o.id,
			o.order_number,
			o.order_date,
			o.total,
			os.name as status,
			os.display_name as status_display,
			COALESCE(
				(
					SELECT json_agg(p.image_url ORDER BY oi.id)
					FROM order_items oi
					JOIN products p ON oi.product_id = p.id
					WHERE oi.order_id = o.id
					LIMIT 4
				),
				'[]'::json
			) as product_images,
			(
				SELECT COUNT(*)
				FROM order_items oi
				WHERE oi.order_id = o.id
			) as total_items
		FROM orders o
		JOIN order_status os ON o.status_id = os.id
		WHERE %s
		ORDER BY o.order_date DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, paramIndex, paramIndex+1)

	args = append(args, limit, offset)

	rows, err := models.DB.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": fmt.Sprintf("Query error: %v", err),
		})
		return
	}
	defer rows.Close()

	orders := []gin.H{}
	for rows.Next() {
		var (
			id            int
			orderNumber   string
			status        string
			statusDisplay string
			orderDate     time.Time
			totalAmount   int
			productImages []byte
			totalItems    int
		)

		err = rows.Scan(&id, &orderNumber, &orderDate, &totalAmount, &status, &statusDisplay, &productImages, &totalItems)
		if err != nil {
			continue
		}

		var images []string
		if len(productImages) > 0 {
			imagesStr := string(productImages)
			imagesStr = strings.Trim(imagesStr, "[]")
			if imagesStr != "" {
				parts := strings.Split(imagesStr, ",")
				for _, part := range parts {
					img := strings.Trim(strings.Trim(part, " "), "\"")
					if img != "null" && img != "" {
						images = append(images, img)
					}
				}
			}
		}

		if len(images) == 0 {
			images = []string{"https://via.placeholder.com/150"}
		}

		orders = append(orders, gin.H{
			"id":            id,
			"invoice":       orderNumber,
			"date":          orderDate.Format("02 January 2006"),
			"status":        status,
			"statusDisplay": statusDisplay,
			"total":         totalAmount,
			"productImages": images,
			"totalItems":    totalItems,
		})
	}

	if orders == nil {
		orders = []gin.H{}
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Order history retrieved",
		"data":    orders,
		"meta": gin.H{
			"page":       page,
			"limit":      limit,
			"totalItems": total,
			"totalPages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}
