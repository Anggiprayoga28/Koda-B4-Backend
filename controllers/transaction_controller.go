package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type TransactionController struct{}

// @Summary Create transaction from cart
// @Description Create order
// @Tags Transactions
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param email formData string false "Email"
// @Param full_name formData string false "Full Name"
// @Param address formData string false "Address"
// @Param delivery_method formData string false "Delivery method"
// @Param payment_method_id formData int false "Payment ID"
// @Success 201 {object} models.Response
// @Router /transactions/checkout [post]
func (ctrl *TransactionController) Checkout(c *gin.Context) {
	ctx := context.Background()
	userID := c.GetInt("user_id")

	rows, err := models.DB.Query(ctx,
		"SELECT ci.id, ci.product_id, p.name, p.price, ci.quantity, p.stock, COALESCE(ci.size_id,0), COALESCE(ci.temperature_id,0), COALESCE(ci.variant_id,0), COALESCE(ps.price_adjustment,0), COALESCE(pt.price,0), COALESCE(pv.price,0), COALESCE(p.is_flash_sale,false) FROM cart_items ci JOIN products p ON ci.product_id=p.id LEFT JOIN product_sizes ps ON ci.size_id=ps.id LEFT JOIN product_temperatures pt ON ci.temperature_id=pt.id LEFT JOIN product_variants pv ON ci.variant_id=pv.id WHERE ci.user_id=$1 AND p.is_active=true",
		userID)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to get cart"})
		return
	}
	defer rows.Close()

	items := []models.CartItem{}
	for rows.Next() {
		var i models.CartItem
		rows.Scan(&i.CartID, &i.ProductID, &i.Name, &i.Price, &i.Qty, &i.Stock,
			&i.SizeID, &i.TempID, &i.VariantID, &i.SizeAdj, &i.TempPrice, &i.VariantPrice, &i.IsFlashSale)

		if i.Stock < i.Qty {
			c.JSON(400, gin.H{"success": false, "message": fmt.Sprintf("Insufficient stock for %s", i.Name)})
			return
		}
		items = append(items, i)
	}

	if len(items) == 0 {
		c.JSON(400, gin.H{"success": false, "message": "Cart is empty"})
		return
	}

	req := models.CheckoutRequest{
		Email:          strings.TrimSpace(c.PostForm("email")),
		FullName:       strings.TrimSpace(c.PostForm("full_name")),
		Address:        strings.TrimSpace(c.PostForm("address")),
		DeliveryMethod: strings.ToLower(c.DefaultPostForm("delivery_method", "dine_in")),
	}
	fmt.Sscanf(c.PostForm("payment_method_id"), "%d", &req.PaymentMethod)
	if req.PaymentMethod == 0 {
		req.PaymentMethod = 1
	}

	if req.Email == "" || req.FullName == "" || req.Address == "" {
		var pe, pn, pa string
		models.DB.QueryRow(ctx, "SELECT u.email, COALESCE(p.full_name,''), COALESCE(p.address,'') FROM users u LEFT JOIN user_profiles p ON u.id=p.user_id WHERE u.id=$1", userID).Scan(&pe, &pn, &pa)

		if req.Email == "" {
			req.Email = pe
		}
		if req.FullName == "" {
			req.FullName = pn
		}
		if req.Address == "" {
			req.Address = pa
		}
	}

	if req.Email == "" || req.FullName == "" || req.Address == "" {
		c.JSON(400, gin.H{"success": false, "message": "Email, full name, and address are required"})
		return
	}

	if req.DeliveryMethod != "dine_in" && req.DeliveryMethod != "door_delivery" && req.DeliveryMethod != "pick_up" {
		c.JSON(400, gin.H{"success": false, "message": "Invalid delivery method"})
		return
	}

	deliveryFee := 0
	if req.DeliveryMethod == "door_delivery" {
		deliveryFee = 10000
	}

	subtotal := 0
	for _, i := range items {
		subtotal += (i.Price + i.SizeAdj + i.TempPrice + i.VariantPrice) * i.Qty
	}
	total := subtotal + deliveryFee

	orderNum := fmt.Sprintf("ORD-%d", time.Now().Unix())
	now := time.Now()

	var orderID int
	err = models.DB.QueryRow(ctx,
		"INSERT INTO orders (order_number, user_id, status, delivery_address, delivery_method_id, subtotal, delivery_fee, tax_amount, total, payment_method_id, order_date, created_at, updated_at) VALUES ($1,$2,'pending',$3,1,$4,$5,0,$6,$7,$8,$9,$10) RETURNING id",
		orderNum, userID, req.Address, subtotal, deliveryFee, total, req.PaymentMethod, now, now, now).Scan(&orderID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to create order"})
		return
	}

	for _, i := range items {
		unitPrice := i.Price + i.SizeAdj + i.TempPrice + i.VariantPrice

		models.DB.Exec(ctx,
			"INSERT INTO order_items (order_id, product_id, quantity, size_id, temperature_id, unit_price, is_flash_sale, created_at) VALUES ($1,$2,$3,NULLIF($4,0),NULLIF($5,0),$6,$7,$8)",
			orderID, i.ProductID, i.Qty, i.SizeID, i.TempID, unitPrice, i.IsFlashSale, now)

		models.DB.Exec(ctx, "UPDATE products SET stock=stock-$1, updated_at=$2 WHERE id=$3", i.Qty, now, i.ProductID)
	}

	models.DB.Exec(ctx, "DELETE FROM cart_items WHERE user_id=$1", userID)

	c.JSON(201, gin.H{
		"success": true,
		"message": "Order created successfully",
		"data": models.TransactionResponse{
			ID:             orderID,
			OrderNumber:    orderNum,
			Status:         "pending",
			Subtotal:       subtotal,
			DeliveryFee:    deliveryFee,
			Total:          total,
			Email:          req.Email,
			FullName:       req.FullName,
			Address:        req.Address,
			DeliveryMethod: req.DeliveryMethod,
		},
	})
}
