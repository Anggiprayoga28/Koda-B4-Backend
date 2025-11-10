package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderController struct{}

// @Summary Get all orders
// @Description Get paginated orders (Admin)
// @Tags Admin - Orders
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} models.PaginationResponse
// @Router /admin/orders [get]
func (ctrl *OrderController) GetAllOrders(c *gin.Context) {
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

	c.JSON(200, gin.H{
		"success": true, "message": "Orders retrieved", "data": orders,
		"meta": gin.H{"page": page, "limit": limit, "total_items": total, "total_pages": int(math.Ceil(float64(total) / float64(limit)))},
	})
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
// @Accept json
// @Produce json
// @Param request body models.CreateOrderRequest true "Order Request"
// @Success 201 {object} models.Response
// @Router /orders [post]
func (ctrl *OrderController) CreateOrder(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	var price int
	err := models.DB.QueryRow(context.Background(), "SELECT price FROM products WHERE id=$1", req.ProductID).Scan(&price)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Product not found"})
		return
	}

	total := price * req.Quantity
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
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Param request body models.UpdateOrderStatusRequest true "Status Request"
// @Success 200 {object} models.Response
// @Router /admin/orders/{id}/status [patch]
func (ctrl *OrderController) UpdateOrderStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req models.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	models.DB.Exec(context.Background(), "UPDATE orders SET status=$1, updated_at=$2 WHERE id=$3", req.Status, time.Now(), id)
	c.JSON(200, gin.H{"success": true, "message": "Status updated"})
}

// @Summary Get dashboard stats
// @Description Get dashboard statistics (Admin)
// @Tags Admin - Dashboard
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Response
// @Router /admin/dashboard [get]
func (ctrl *OrderController) GetDashboard(c *gin.Context) {
	var total, pending, shipping, completed, revenue int
	models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*), COUNT(CASE WHEN status='pending' THEN 1 END), COUNT(CASE WHEN status='shipping' THEN 1 END), COUNT(CASE WHEN status='done' THEN 1 END), COALESCE(SUM(total),0) FROM orders").Scan(
		&total, &pending, &shipping, &completed, &revenue)

	c.JSON(200, gin.H{
		"success": true, "message": "Dashboard retrieved",
		"data": gin.H{
			"total_orders": total, "pending_orders": pending,
			"shipping_orders": shipping, "completed_orders": completed, "total_revenue": revenue,
		},
	})
}
