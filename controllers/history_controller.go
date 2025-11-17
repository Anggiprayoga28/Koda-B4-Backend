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
// @Description Get paginated order history
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

	query := `
		SELECT 
			o.id,
			o.order_number,
			o.order_date,
			o.total,
			os.name as status,
			os.display_name as status_display,
			COALESCE(p.image_url, '') as product_image
		FROM orders o
		JOIN order_status os ON o.status_id = os.id
		LEFT JOIN order_items oi ON o.id = oi.order_id
		LEFT JOIN products p ON oi.product_id = p.id
		WHERE o.user_id = $1
	`
	args := []interface{}{userID}
	paramIndex := 2

	if statusFilter != "" {
		query += fmt.Sprintf(" AND os.name = $%d", paramIndex)
		args = append(args, statusFilter)
		paramIndex++
	}

	if startDate != "" {
		query += fmt.Sprintf(" AND DATE(o.order_date) >= $%d", paramIndex)
		args = append(args, startDate)
		paramIndex++
	}
	if endDate != "" {
		query += fmt.Sprintf(" AND DATE(o.order_date) <= $%d", paramIndex)
		args = append(args, endDate)
		paramIndex++
	}

	if monthFilter != "" {
		t, err := time.Parse("January 2006", monthFilter)
		if err == nil {
			year, month, _ := t.Date()
			firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
			lastDay := firstDay.AddDate(0, 1, -1)

			query += fmt.Sprintf(" AND DATE(o.order_date) >= $%d AND DATE(o.order_date) <= $%d", paramIndex, paramIndex+1)
			args = append(args, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"))
			paramIndex += 2
		}
	}

	query += " GROUP BY o.id, o.order_number, o.order_date, o.total, os.name, os.display_name, p.image_url"
	query += " ORDER BY o.order_date DESC"

	countQuery := `
		SELECT COUNT(DISTINCT o.id)
		FROM orders o
		JOIN order_status os ON o.status_id = os.id
		WHERE o.user_id = $1
	`
	countArgs := []interface{}{userID}
	countParamIndex := 2

	if statusFilter != "" {
		countQuery += fmt.Sprintf(" AND os.name = $%d", countParamIndex)
		countArgs = append(countArgs, statusFilter)
		countParamIndex++
	}
	if startDate != "" {
		countQuery += fmt.Sprintf(" AND DATE(o.order_date) >= $%d", countParamIndex)
		countArgs = append(countArgs, startDate)
		countParamIndex++
	}
	if endDate != "" {
		countQuery += fmt.Sprintf(" AND DATE(o.order_date) <= $%d", countParamIndex)
		countArgs = append(countArgs, endDate)
		countParamIndex++
	}
	if monthFilter != "" {
		t, err := time.Parse("January 2006", monthFilter)
		if err == nil {
			year, month, _ := t.Date()
			firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
			lastDay := firstDay.AddDate(0, 1, -1)

			countQuery += fmt.Sprintf(" AND DATE(o.order_date) >= $%d AND DATE(o.order_date) <= $%d", countParamIndex, countParamIndex+1)
			countArgs = append(countArgs, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"))
		}
	}

	var total int
	models.DB.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	args = append(args, limit, offset)

	rows, err := models.DB.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Query error: %v", err)})
		return
	}
	defer rows.Close()

	orders := []gin.H{}
	for rows.Next() {
		var id int
		var orderNumber, status, statusDisplay, productImage string
		var orderDate time.Time
		var totalAmount int

		err = rows.Scan(&id, &orderNumber, &orderDate, &totalAmount, &status, &statusDisplay, &productImage)
		if err != nil {
			continue
		}

		orders = append(orders, gin.H{
			"id":            id,
			"invoice":       orderNumber,
			"date":          orderDate.Format("02 January 2006"),
			"status":        status,
			"statusDisplay": statusDisplay,
			"total":         totalAmount,
			"imageProduct":  productImage,
		})
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
