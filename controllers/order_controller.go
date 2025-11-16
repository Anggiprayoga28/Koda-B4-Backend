package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"net/url"
	"strconv"
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
		links.Prev = &prevURL
	}

	if page < totalPages {
		nextURL := makeURL(page + 1)
		links.Next = &nextURL
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
// @Description Get paginated orders (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} models.HATEOASResponse
// @Router /admin/orders [get]
func (ctrl *OrderController) GetAllOrders(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)

	var total int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM orders").Scan(&total)

	rows, _ := models.DB.Query(context.Background(),
		"SELECT id, order_number, user_id, status, total, created_at FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)
	defer rows.Close()

	orders := []gin.H{}
	for rows.Next() {
		var o models.Order
		rows.Scan(&o.ID, &o.OrderNumber, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
		orders = append(orders, gin.H{
			"id": o.ID, "order_number": o.OrderNumber, "user_id": o.UserID,
			"status": o.Status, "total": o.Total, "created_at": o.CreatedAt,
		})
	}

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
		c.JSON(400, gin.H{"success": false, "message": "Invalid order ID"})
		return
	}

	var o models.Order
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, order_number, user_id, status, total, created_at FROM orders WHERE id=$1",
		id).Scan(&o.ID, &o.OrderNumber, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Order not found"})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "Order retrieved", "data": o})
}

// @Summary Create order
// @Description Create new order
// @Tags Orders
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param product_id formData int true "Product ID"
// @Param quantity formData int true "Quantity"
// @Success 201 {object} models.Response
// @Router /orders [post]
func (ctrl *OrderController) CreateOrder(c *gin.Context) {
	userID := c.GetInt("user_id")

	productIDStr := c.PostForm("product_id")
	quantityStr := c.PostForm("quantity")

	if productIDStr == "" || quantityStr == "" {
		c.JSON(400, gin.H{"success": false, "message": "Product ID and quantity are required"})
		return
	}

	productID, err := strconv.Atoi(productIDStr)
	if err != nil || productID <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid product ID"})
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid quantity"})
		return
	}

	if quantity > 100 {
		c.JSON(400, gin.H{"success": false, "message": "Maximum quantity is 100"})
		return
	}

	var price, stock int
	var isActive bool
	err = models.DB.QueryRow(context.Background(),
		"SELECT price, stock, is_active FROM products WHERE id=$1",
		productID).Scan(&price, &stock, &isActive)

	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Product not found"})
		return
	}

	if !isActive {
		c.JSON(400, gin.H{"success": false, "message": "Product is not available"})
		return
	}

	if stock < quantity {
		c.JSON(400, gin.H{"success": false, "message": "Insufficient stock"})
		return
	}

	total := price * quantity
	orderNumber := fmt.Sprintf("ORD-%d", time.Now().Unix())
	now := time.Now()

	var orderID int
	models.DB.QueryRow(context.Background(),
		"INSERT INTO orders (order_number, user_id, status, delivery_address, delivery_method_id, subtotal, delivery_fee, tax_amount, total, payment_method_id, order_date, created_at, updated_at) VALUES ($1,$2,'pending','Default',1,$3,0,0,$3,1,$4,$5,$6) RETURNING id",
		orderNumber, userID, total, now, now, now).Scan(&orderID)

	c.JSON(201, gin.H{
		"success": true, "message": "Order created",
		"data": gin.H{"id": orderID, "order_number": orderNumber, "status": "pending", "total": total},
	})
}

// @Summary Update order status
// @Description Update order status (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Order ID"
// @Param status formData string true "Status"
// @Success 200 {object} models.Response
// @Router /admin/orders/{id}/status [patch]
func (ctrl *OrderController) UpdateOrderStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid order ID"})
		return
	}

	status := c.PostForm("status")
	if status == "" {
		c.JSON(400, gin.H{"success": false, "message": "Status is required"})
		return
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"shipping":  true,
		"done":      true,
		"cancelled": true,
	}
	if !validStatuses[status] {
		c.JSON(400, gin.H{"success": false, "message": "Invalid status. Must be: pending, shipping, done, or cancelled"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM orders WHERE id=$1", id).Scan(&exists)
	if exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "Order not found"})
		return
	}

	models.DB.Exec(context.Background(), "UPDATE orders SET status=$1, updated_at=$2 WHERE id=$3", status, time.Now(), id)
	c.JSON(200, gin.H{"success": true, "message": "Status updated"})
}
