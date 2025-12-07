package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderDetailController struct{}

// @Summary Get order detail
// @Description Get complete order information with items
// @Tags History
// @Security BearerAuth
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.ErrorResponse
// @Router /orders/{id}/detail [get]
func (ctrl *OrderDetailController) GetOrderDetail(c *gin.Context) {
	userID := c.GetInt("user_id")
	orderID, _ := strconv.Atoi(c.Param("id"))
	fmt.Println(userID)
	fmt.Println(orderID)

	if orderID <= 0 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid order ID",
		})
		return
	}

	var orderNumber, deliveryAddress, deliveryMethod, paymentMethod, statusName, statusDisplay string
	var orderDate time.Time
	var subtotal, deliveryFee, taxAmount, total int
	var phone, fullName string

	err := models.DB.QueryRow(context.Background(),
		`SELECT 
			o.order_number,
			o.order_date,
			o.delivery_address,
			COALESCE(dm.name, 'Dine In') as delivery_method,
			COALESCE(pm.name, 'Cash') as payment_method,
			o.subtotal,
			o.delivery_fee,
			o.tax_amount,
			o.total,
			os.name as status_name,
			os.display_name as status_display,
			COALESCE(up.phone, '') as phone,
			COALESCE(up.full_name, u.email) as full_name
		FROM orders o
		JOIN users u ON o.user_id = u.id
		LEFT JOIN user_profiles up ON u.id = up.user_id
		LEFT JOIN delivery_methods dm ON o.delivery_method_id = dm.id
		LEFT JOIN payment_methods pm ON o.payment_method_id = pm.id
		JOIN order_status os ON o.status_id = os.id
		WHERE o.id = $1 AND o.user_id = $2`,
		orderID, userID).Scan(
		&orderNumber, &orderDate, &deliveryAddress, &deliveryMethod,
		&paymentMethod, &subtotal, &deliveryFee, &taxAmount, &total,
		&statusName, &statusDisplay, &phone, &fullName)

	if err != nil {
		c.JSON(404, gin.H{
			"success": false,
			"message": "Order not found",
		})
		return
	}

	rows, err := models.DB.Query(context.Background(),
		`SELECT 
			oi.product_id,
			p.name,
			oi.quantity,
			COALESCE(ps.name,'') as size_name,
			COALESCE(pt.name,'') as temperature_name,
			oi.unit_price,
			COALESCE(p.image_url, '') as image_url,
			COALESCE(oi.is_flash_sale, false) as is_flash_sale,
			COALESCE(dm.name, 'Dine In') as delivery_method
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		LEFT JOIN product_sizes ps ON oi.size_id = ps.id
		LEFT JOIN product_temperatures pt ON oi.temperature_id = pt.id
		LEFT JOIN orders o ON oi.order_id = o.id
		LEFT JOIN delivery_methods dm ON o.delivery_method_id = dm.id
		WHERE oi.order_id = $1`, orderID)

	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to get order items",
		})
		return
	}
	defer rows.Close()

	items := []gin.H{}
	for rows.Next() {
		var productID, quantity, unitPrice int
		var productName, sizeName, tempName, imageURL, itemDeliveryMethod string
		var isFlashSale bool

		rows.Scan(&productID, &productName, &quantity, &sizeName, &tempName, &unitPrice, &imageURL, &isFlashSale, &itemDeliveryMethod)

		items = append(items, gin.H{
			"productId":      productID,
			"name":           productName,
			"quantity":       quantity,
			"size":           sizeName,
			"temperature":    tempName,
			"unitPrice":      unitPrice,
			"totalPrice":     unitPrice * quantity,
			"imageUrl":       imageURL,
			"isFlashSale":    isFlashSale,
			"deliveryMethod": itemDeliveryMethod,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Order detail retrieved successfully",
		"data": gin.H{
			"orderNumber":    orderNumber,
			"orderDate":      orderDate.Format("02 January 2006 at 03:04 PM"),
			"fullName":       fullName,
			"phone":          phone,
			"address":        deliveryAddress,
			"deliveryMethod": deliveryMethod,
			"paymentMethod":  paymentMethod,
			"status":         statusName,
			"statusDisplay":  statusDisplay,
			"subtotal":       subtotal,
			"deliveryFee":    deliveryFee,
			"taxAmount":      taxAmount,
			"total":          total,
			"items":          items,
		},
	})
}
