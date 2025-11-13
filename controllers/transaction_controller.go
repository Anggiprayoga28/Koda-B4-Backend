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

// @Summary Create transaction
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

	tx, err := models.DB.Begin(ctx)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`SELECT 
			ci.id, 
			ci.product_id, 
			p.name, 
			p.price, 
			ci.quantity, 
			p.stock,
			ci.size_id,
			ci.temperature_id,
			ci.variant_id,
			p.is_flash_sale
		FROM cart_items ci 
		JOIN products p ON ci.product_id = p.id 
		WHERE ci.user_id = $1
		FOR UPDATE`,
		userID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Query error: %v", err)})
		return
	}
	defer rows.Close()

	type CartItem struct {
		CartID      int
		ProductID   int
		Name        string
		Price       int
		Qty         int
		Stock       int
		SizeID      *int
		TempID      *int
		VariantID   *int
		IsFlashSale bool
	}

	items := []CartItem{}
	for rows.Next() {
		var i CartItem
		err = rows.Scan(&i.CartID, &i.ProductID, &i.Name, &i.Price, &i.Qty, &i.Stock,
			&i.SizeID, &i.TempID, &i.VariantID, &i.IsFlashSale)
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Scan error: %v", err)})
			return
		}

		if i.Stock < i.Qty {
			c.JSON(400, gin.H{"success": false, "message": fmt.Sprintf("Insufficient stock for %s", i.Name)})
			return
		}
		items = append(items, i)
	}
	rows.Close()

	if len(items) == 0 {
		c.JSON(400, gin.H{"success": false, "message": "Cart is empty"})
		return
	}

	for idx := range items {
		sizeAdj := 0
		tempPrice := 0
		variantPrice := 0

		if items[idx].SizeID != nil && *items[idx].SizeID > 0 {
			tx.QueryRow(ctx, "SELECT COALESCE(price_adjustment, 0) FROM product_sizes WHERE id=$1", *items[idx].SizeID).Scan(&sizeAdj)
		}
		if items[idx].TempID != nil && *items[idx].TempID > 0 {
			tx.QueryRow(ctx, "SELECT COALESCE(price, 0) FROM product_temperatures WHERE id=$1", *items[idx].TempID).Scan(&tempPrice)
		}
		if items[idx].VariantID != nil && *items[idx].VariantID > 0 {
			tx.QueryRow(ctx, "SELECT COALESCE(price, 0) FROM product_variants WHERE id=$1", *items[idx].VariantID).Scan(&variantPrice)
		}

		items[idx].Price += sizeAdj + tempPrice + variantPrice
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
		tx.QueryRow(ctx, "SELECT u.email, COALESCE(p.full_name,''), COALESCE(p.address,'') FROM users u LEFT JOIN user_profiles p ON u.id=p.user_id WHERE u.id=$1", userID).Scan(&pe, &pn, &pa)

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
		subtotal += i.Price * i.Qty
	}
	total := subtotal + deliveryFee

	orderNum := fmt.Sprintf("ORD-%d", time.Now().Unix())
	now := time.Now()

	var statusID int
	err = tx.QueryRow(ctx, "SELECT id FROM order_status WHERE name='pending' LIMIT 1").Scan(&statusID)
	if err != nil {
		statusID = 1
	}

	var orderID int
	err = tx.QueryRow(ctx,
		"INSERT INTO orders (order_number, user_id, status_id, delivery_address, delivery_method_id, subtotal, delivery_fee, tax_amount, total, payment_method_id, order_date, created_at, updated_at) VALUES ($1,$2,$3,$4,1,$5,$6,0,$7,$8,$9,$10,$11) RETURNING id",
		orderNum, userID, statusID, req.Address, subtotal, deliveryFee, total, req.PaymentMethod, now, now, now).Scan(&orderID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Failed to create order: %v", err)})
		return
	}

	for _, i := range items {
		var sizeID, tempID interface{}
		if i.SizeID != nil && *i.SizeID > 0 {
			sizeID = *i.SizeID
		}
		if i.TempID != nil && *i.TempID > 0 {
			tempID = *i.TempID
		}

		_, err = tx.Exec(ctx,
			"INSERT INTO order_items (order_id, product_id, quantity, size_id, temperature_id, unit_price, is_flash_sale, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
			orderID, i.ProductID, i.Qty, sizeID, tempID, i.Price, i.IsFlashSale, now)
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Failed to create order items: %v", err)})
			return
		}

		_, err = tx.Exec(ctx, "UPDATE products SET stock=stock-$1, updated_at=$2 WHERE id=$3", i.Qty, now, i.ProductID)
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Failed to update stock: %v", err)})
			return
		}
	}

	_, err = tx.Exec(ctx, "DELETE FROM cart_items WHERE user_id=$1", userID)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Failed to clear cart: %v", err)})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(500, gin.H{"success": false, "message": fmt.Sprintf("Failed to commit: %v", err)})
		return
	}

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
