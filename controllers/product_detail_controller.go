package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductDetailController struct{}

// @Summary Get product detail with variants
// @Description Get complete product information including sizes, temperatures, and recommendations
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} models.Response
// @Router /products/{id}/detail [get]
func (ctrl *ProductDetailController) GetProductDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var p models.Product
	err := models.DB.QueryRow(context.Background(),
		`SELECT id, name, description, category_id, price, stock, COALESCE(image_url, ''), 
		COALESCE(is_flash_sale, false), COALESCE(is_favorite, false), COALESCE(is_buy1get1, false), 
		is_active, created_at, updated_at FROM products WHERE id=$1 AND is_active=true`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Stock, &p.ImageURL,
			&p.IsFlashSale, &p.IsFavorite, &p.IsBuy1Get1, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "Product not found"})
		return
	}

	images := []gin.H{}
	imgRows, _ := models.DB.Query(context.Background(),
		"SELECT image_url, display_order FROM product_images WHERE product_id=$1 ORDER BY display_order", id)
	defer imgRows.Close()
	for imgRows.Next() {
		var url string
		var order int
		imgRows.Scan(&url, &order)
		images = append(images, gin.H{"url": url, "order": order})
	}

	sizes := []gin.H{}
	sizeRows, _ := models.DB.Query(context.Background(),
		"SELECT id, name, price_adjustment FROM product_sizes WHERE is_active=true ORDER BY price_adjustment")
	defer sizeRows.Close()
	for sizeRows.Next() {
		var sid int
		var name string
		var adj int
		sizeRows.Scan(&sid, &name, &adj)
		sizes = append(sizes, gin.H{"id": sid, "name": name, "priceAdjustment": adj})
	}

	temps := []gin.H{}
	tempRows, _ := models.DB.Query(context.Background(),
		"SELECT id, name FROM product_temperatures WHERE is_active=true")
	defer tempRows.Close()
	for tempRows.Next() {
		var tid int
		var name string
		tempRows.Scan(&tid, &name)
		temps = append(temps, gin.H{"id": tid, "name": name})
	}

	var totalReviews int
	var avgRating float64
	models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*), COALESCE(AVG(rating), 0) FROM product_reviews WHERE product_id=$1", id).
		Scan(&totalReviews, &avgRating)

	reviews := []gin.H{}
	revRows, _ := models.DB.Query(context.Background(),
		`SELECT pr.rating, pr.review_text, pr.created_at, up.full_name 
		FROM product_reviews pr 
		LEFT JOIN user_profiles up ON pr.user_id=up.user_id 
		WHERE pr.product_id=$1 ORDER BY pr.created_at DESC LIMIT 5`, id)
	defer revRows.Close()
	for revRows.Next() {
		var rating int
		var text, name string
		var created interface{}
		revRows.Scan(&rating, &text, &created, &name)
		reviews = append(reviews, gin.H{
			"rating": rating, "text": text, "user": name, "createdAt": created,
		})
	}

	recs := []gin.H{}
	recRows, _ := models.DB.Query(context.Background(),
		`SELECT id, name, price, COALESCE(image_url, ''), COALESCE(is_flash_sale, false) 
		FROM products WHERE category_id=$1 AND id!=$2 AND is_active=true LIMIT 3`,
		p.CategoryID, id)
	defer recRows.Close()
	for recRows.Next() {
		var rid, rprice int
		var rname, rimg string
		var rflash bool
		recRows.Scan(&rid, &rname, &rprice, &rimg, &rflash)
		recs = append(recs, gin.H{
			"id": rid, "name": rname, "price": rprice, "imageUrl": rimg, "isFlashSale": rflash,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Product detail retrieved",
		"data": gin.H{
			"product":         p,
			"images":          images,
			"sizes":           sizes,
			"temperatures":    temps,
			"totalReviews":    totalReviews,
			"averageRating":   avgRating,
			"reviews":         reviews,
			"recommendations": recs,
		},
	})
}

// Create cart
// @Summary Add to cart
// @Description Add product to cart
// @Tags Cart
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param product_id formData int true "Product ID"
// @Param quantity formData int true "Quantity"
// @Param size_id formData int false "Size ID"
// @Param temperature_id formData int false "Temperature ID"
// @Param variant_id formData int false "Variant ID"
// @Success 201 {object} models.Response
// @Router /cart [post]
func (ctrl *ProductDetailController) AddToCart(c *gin.Context) {
	ctx := context.Background()
	userID := c.GetInt("user_id")

	productIDStr := c.PostForm("product_id")
	quantityStr := c.PostForm("quantity")
	sizeIDStr := c.PostForm("size_id")
	tempIDStr := c.PostForm("temperature_id")
	variantIDStr := c.PostForm("variant_id")

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

	var sizeID, tempID, variantID *int

	if sizeIDStr != "" {
		if val, err := strconv.Atoi(sizeIDStr); err == nil && val > 0 {
			sizeID = &val
		}
	}

	if tempIDStr != "" {
		if val, err := strconv.Atoi(tempIDStr); err == nil && val > 0 {
			tempID = &val
		}
	}

	if variantIDStr != "" {
		if val, err := strconv.Atoi(variantIDStr); err == nil && val > 0 {
			variantID = &val
		}
	}

	var productExists bool
	var stock int
	err = models.DB.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND is_active=true), COALESCE((SELECT stock FROM products WHERE id=$1), 0)",
		productID).Scan(&productExists, &stock)

	if err != nil || !productExists {
		c.JSON(400, gin.H{"success": false, "message": "Product not found or inactive"})
		return
	}

	if stock < quantity {
		c.JSON(400, gin.H{"success": false, "message": fmt.Sprintf("Insufficient stock. Available: %d", stock)})
		return
	}

	var existingID, existingQty int
	checkQuery := `
		SELECT id, quantity FROM cart_items 
		WHERE user_id=$1 
		AND product_id=$2 
		AND ($3::int IS NULL AND size_id IS NULL OR size_id = $3)
		AND ($4::int IS NULL AND temperature_id IS NULL OR temperature_id = $4)
		AND ($5::int IS NULL AND variant_id IS NULL OR variant_id = $5)
	`

	err = models.DB.QueryRow(ctx, checkQuery,
		userID, productID, sizeID, tempID, variantID).Scan(&existingID, &existingQty)

	if err == nil {
		newQty := existingQty + quantity

		if stock < newQty {
			c.JSON(400, gin.H{"success": false, "message": fmt.Sprintf("Insufficient stock. Available: %d, Current cart: %d", stock, existingQty)})
			return
		}

		_, err = models.DB.Exec(ctx,
			"UPDATE cart_items SET quantity=$1, updated_at=NOW() WHERE id=$2",
			newQty, existingID)

		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to update cart: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Cart updated successfully",
			"data": gin.H{
				"cartItemId": existingID,
				"quantity":   newQty,
			},
		})
		return
	}

	insertQuery := `
		INSERT INTO cart_items (user_id, product_id, quantity, size_id, temperature_id, variant_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`

	var newCartID int
	err = models.DB.QueryRow(ctx, insertQuery,
		userID, productID, quantity, sizeID, tempID, variantID).Scan(&newCartID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to add to cart: " + err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"success": true,
		"message": "Added to cart successfully",
		"data": gin.H{
			"cartItemId": newCartID,
			"quantity":   quantity,
		},
	})
}

// Get user cart
// @Summary Get user cart
// @Description Get all cart items for current user
// @Tags Cart
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Response
// @Router /cart [get]
func (ctrl *ProductDetailController) GetCart(c *gin.Context) {
	ctx := context.Background()
	userID := c.GetInt("user_id")

	rows, err := models.DB.Query(ctx,
		`SELECT 
			ci.id, 
			ci.product_id, 
			p.name, 
			p.price, 
			ci.quantity, 
			COALESCE(ps.name,'') as size_name, 
			COALESCE(ps.price_adjustment,0) as size_adj,
			COALESCE(pt.name,'') as temp_name, 
			COALESCE(pt.price,0) as temp_price,
			COALESCE(pv.name,'') as variant_name, 
			COALESCE(pv.price,0) as variant_price,
			COALESCE(p.image_url,'') as image_url,
			p.stock
		FROM cart_items ci
		JOIN products p ON ci.product_id=p.id
		LEFT JOIN product_sizes ps ON ci.size_id=ps.id
		LEFT JOIN product_temperatures pt ON ci.temperature_id=pt.id
		LEFT JOIN product_variants pv ON ci.variant_id=pv.id
		WHERE ci.user_id=$1 
		AND p.is_active=true
		ORDER BY ci.created_at DESC`, userID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Failed to retrieve cart: " + err.Error()})
		return
	}
	defer rows.Close()

	items := []gin.H{}
	subtotal := 0

	for rows.Next() {
		var id, pid, basePrice, qty, sizeAdj, tempPrice, variantPrice, stock int
		var pname, sizeName, tempName, variantName, img string

		err = rows.Scan(&id, &pid, &pname, &basePrice, &qty,
			&sizeName, &sizeAdj, &tempName, &tempPrice,
			&variantName, &variantPrice, &img, &stock)

		if err != nil {
			continue
		}

		itemPrice := (basePrice + sizeAdj + tempPrice + variantPrice) * qty
		subtotal += itemPrice

		items = append(items, gin.H{
			"id":               id,
			"productId":        pid,
			"name":             pname,
			"basePrice":        basePrice,
			"quantity":         qty,
			"size":             sizeName,
			"sizeAdjustment":   sizeAdj,
			"temperature":      tempName,
			"temperaturePrice": tempPrice,
			"variant":          variantName,
			"variantPrice":     variantPrice,
			"subtotal":         itemPrice,
			"imageUrl":         img,
			"stock":            stock,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Cart retrieved",
		"data": gin.H{
			"items":    items,
			"subtotal": subtotal,
		},
	})
}
