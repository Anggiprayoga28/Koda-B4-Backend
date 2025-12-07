package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderController struct{}

func (ctrl *OrderController) getPaginationParams(c *gin.Context, defaultLimit int) (page, limit, offset int) {
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

func (ctrl *OrderController) generateLinks(c *gin.Context, page, limit, totalPages int) models.PaginationLinks {
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
		links.Prev = prevURL
	}

	if page < totalPages {
		nextURL := makeURL(page + 1)
		links.Next = nextURL
	}

	return links
}

func (ctrl *OrderController) buildResponse(c *gin.Context, message string, data interface{}, page, limit, totalItems int) models.HATEOASResponse {
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

	links := ctrl.generateLinks(c, page, limit, totalPages)

	return models.HATEOASResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
		Links:   links,
	}
}

// @Summary Get all orders
// @Description Get all orders with pagination (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param status query string false "Filter by status"
// @Param search query string false "Search by order number"
// @Success 200 {object} models.HATEOASResponse
// @Router /admin/orders [get]
func (ctrl *OrderController) GetAllOrders(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)

	status := c.Query("status")
	search := c.Query("search")

	var total int
	countQuery := "SELECT COUNT(*) FROM orders"
	countArgs := []interface{}{}

	whereConditions := []string{}
	if status != "" && status != "All" {
		whereConditions = append(whereConditions, "status = $1")
		countArgs = append(countArgs, status)
	}

	if len(whereConditions) > 0 {
		countQuery += " WHERE " + strings.Join(whereConditions, " AND ")
		err := models.DB.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
		if err != nil {
			fmt.Println("Count error:", err)
			c.JSON(500, gin.H{"success": false, "message": "Failed to count orders"})
			return
		}
	} else {
		err := models.DB.QueryRow(context.Background(), countQuery).Scan(&total)
		if err != nil {
			fmt.Println("Count error:", err)
			c.JSON(500, gin.H{"success": false, "message": "Failed to count orders"})
			return
		}
	}

	fmt.Println("Total orders in DB:", total)

	query := `
		SELECT 
			o.id,
			o.user_id,
			o.subtotal,
			o.created_at
		FROM orders o
	`

	queryArgs := []interface{}{}
	argIndex := 1

	whereConditions = []string{}

	if status != "" && status != "All" {
		whereConditions = append(whereConditions, fmt.Sprintf("o.status = $%d", argIndex))
		queryArgs = append(queryArgs, status)
		argIndex++
	}

	if search != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("CAST(o.id AS TEXT) LIKE $%d", argIndex))
		queryArgs = append(queryArgs, "%"+search+"%")
		argIndex++
	}

	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	queryArgs = append(queryArgs, limit, offset)

	fmt.Println("Executing query:", query)
	fmt.Println("With args:", queryArgs)

	rows, err := models.DB.Query(context.Background(), query, queryArgs...)
	if err != nil {
		fmt.Println("Query execution error:", err)
		c.JSON(500, gin.H{
			"success": false,
			"message": "Query error: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	orders := []gin.H{}
	rowCount := 0

	for rows.Next() {
		var id, userID, subtotal int
		var createdAt time.Time

		err := rows.Scan(&id, &userID, &subtotal, &createdAt)
		if err != nil {
			fmt.Println("Scan error on row", rowCount, ":", err)
			continue
		}

		rowCount++
		fmt.Println("Scanned order:", id, userID, subtotal)

		orders = append(orders, gin.H{
			"id":               id,
			"order_id":         id,
			"orderId":          id,
			"order_number":     fmt.Sprintf("ORD-%d", id),
			"orderNumber":      fmt.Sprintf("ORD-%d", id),
			"user_id":          userID,
			"userId":           userID,
			"status":           "pending",
			"subtotal":         subtotal,
			"total":            subtotal,
			"customer_name":    "Customer",
			"customerName":     "Customer",
			"delivery_address": "N/A",
			"deliveryAddress":  "N/A",
			"created_at":       createdAt,
			"createdAt":        createdAt,
		})
	}

	fmt.Println("Total rows scanned:", rowCount)
	fmt.Println("Orders array length:", len(orders))

	response := ctrl.buildResponse(c, "Orders retrieved successfully", orders, page, limit, total)
	c.JSON(200, response)
}

// @Summary Get order by ID
// @Description Get order details (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.ErrorResponse
// @Router /admin/orders/{id} [get]
func (ctrl *OrderController) GetOrderByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if id <= 0 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid order ID",
		})
		return
	}

	query := "SELECT id, user_id, subtotal, created_at FROM orders WHERE id = $1"

	var orderID, userID, subtotal int
	var createdAt time.Time

	err := models.DB.QueryRow(context.Background(), query, id).Scan(
		&orderID, &userID, &subtotal, &createdAt,
	)

	if err != nil {
		c.JSON(404, gin.H{
			"success": false,
			"message": "Order not found",
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Order retrieved successfully",
		"data": gin.H{
			"id":           orderID,
			"order_id":     orderID,
			"orderId":      orderID,
			"order_number": fmt.Sprintf("ORD-%d", orderID),
			"orderNumber":  fmt.Sprintf("ORD-%d", orderID),
			"user_id":      userID,
			"userId":       userID,
			"status":       "pending",
			"subtotal":     subtotal,
			"total":        subtotal,
			"created_at":   createdAt,
			"createdAt":    createdAt,
		},
	})
}

// @Summary Create order
// @Description Create new order (Customer)
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param order body object true "Order data"
// @Success 201 {object} models.Response
// @Router /orders [post]
func (ctrl *OrderController) CreateOrder(c *gin.Context) {
	userID := c.GetInt("user_id")

	if userID == 0 {
		c.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}

	c.JSON(201, gin.H{
		"success": true,
		"message": "Order created successfully",
		"data": gin.H{
			"order_id": 1,
			"message":  "Please use /transactions/checkout endpoint instead",
		},
	})
}

// @Summary Update order status
// @Description Update order status (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Order ID"
// @Param status formData string true "New status"
// @Success 200 {object} models.Response
// @Router /admin/orders/{id}/status [patch]
func (ctrl *OrderController) UpdateOrderStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	status := strings.TrimSpace(c.PostForm("status"))

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid order ID"})
		return
	}

	if status == "" {
		c.JSON(400, gin.H{"success": false, "message": "Status is required"})
		return
	}

	var exists int
	err := models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM orders WHERE id=$1", id).Scan(&exists)
	if err != nil || exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "Order not found"})
		return
	}

	_, err = models.DB.Exec(context.Background(),
		"UPDATE orders SET status=$1, updated_at=$2 WHERE id=$3",
		status, time.Now(), id)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to update order status"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Order status updated successfully",
		"data": gin.H{
			"id":     id,
			"status": status,
		},
	})
}

// @Summary Delete order
// @Description Delete order (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} models.Response
// @Router /admin/orders/{id} [delete]
func (ctrl *OrderController) DeleteOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid order ID"})
		return
	}

	var exists int
	err := models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM orders WHERE id=$1", id).Scan(&exists)
	if err != nil || exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "Order not found"})
		return
	}

	ctx := context.Background()
	tx, err := models.DB.Begin(ctx)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	_, _ = tx.Exec(ctx, "DELETE FROM order_items WHERE order_id=$1", id)

	_, err = tx.Exec(ctx, "DELETE FROM orders WHERE id=$1", id)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to delete order"})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to commit transaction"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Order deleted successfully",
		"data": gin.H{
			"id": id,
		},
	})
}
