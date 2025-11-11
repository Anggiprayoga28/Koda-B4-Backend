package controllers

import (
	"coffee-shop/models"
	"context"
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
		sizes = append(sizes, gin.H{"id": sid, "name": name, "price_adjustment": adj})
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
			"rating": rating, "text": text, "user": name, "created_at": created,
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
			"id": rid, "name": rname, "price": rprice, "image_url": rimg, "is_flash_sale": rflash,
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
			"total_reviews":   totalReviews,
			"average_rating":  avgRating,
			"reviews":         reviews,
			"recommendations": recs,
		},
	})
}

// @Summary Add to cart with variants
// @Description Add product to cart with size and temperature options
// @Tags Cart
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param product_id formData int true "Product ID"
// @Param quantity formData int true "Quantity"
// @Param size_id formData int false "Size ID"
// @Param temperature_id formData int false "Temperature ID"
// @Success 201 {object} models.Response
// @Router /cart [post]
func (ctrl *ProductDetailController) AddToCart(c *gin.Context) {
	userID := c.GetInt("user_id")
	productID, _ := strconv.Atoi(c.PostForm("product_id"))
	quantity, _ := strconv.Atoi(c.PostForm("quantity"))
	sizeID, _ := strconv.Atoi(c.PostForm("size_id"))
	tempID, _ := strconv.Atoi(c.PostForm("temperature_id"))

	if productID <= 0 || quantity <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid product or quantity"})
		return
	}

	var existingID, existingQty int
	err := models.DB.QueryRow(context.Background(),
		`SELECT id, quantity FROM cart_items 
		WHERE user_id=$1 AND product_id=$2 AND COALESCE(size_id,0)=$3 AND COALESCE(temperature_id,0)=$4`,
		userID, productID, sizeID, tempID).Scan(&existingID, &existingQty)

	if err == nil {
		models.DB.Exec(context.Background(),
			"UPDATE cart_items SET quantity=$1 WHERE id=$2", existingQty+quantity, existingID)
	} else {
		models.DB.Exec(context.Background(),
			`INSERT INTO cart_items (user_id, product_id, quantity, size_id, temperature_id, created_at, updated_at) 
			VALUES ($1,$2,$3,NULLIF($4,0),NULLIF($5,0),NOW(),NOW())`,
			userID, productID, quantity, sizeID, tempID)
	}

	c.JSON(201, gin.H{"success": true, "message": "Added to cart"})
}

// @Summary Get user cart
// @Description Get all cart items for current user
// @Tags Cart
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Response
// @Router /cart [get]
func (ctrl *ProductDetailController) GetCart(c *gin.Context) {
	userID := c.GetInt("user_id")

	rows, _ := models.DB.Query(context.Background(),
		`SELECT ci.id, ci.product_id, p.name, p.price, ci.quantity, 
		COALESCE(ps.name,''), COALESCE(ps.price_adjustment,0),
		COALESCE(pt.name,''), COALESCE(p.image_url,'')
		FROM cart_items ci
		JOIN products p ON ci.product_id=p.id
		LEFT JOIN product_sizes ps ON ci.size_id=ps.id
		LEFT JOIN product_temperatures pt ON ci.temperature_id=pt.id
		WHERE ci.user_id=$1`, userID)
	defer rows.Close()

	items := []gin.H{}
	total := 0
	for rows.Next() {
		var id, pid, price, qty, adj int
		var pname, sname, tname, img string
		rows.Scan(&id, &pid, &pname, &price, &qty, &sname, &adj, &tname, &img)

		itemPrice := (price + adj) * qty
		total += itemPrice

		items = append(items, gin.H{
			"id": id, "product_id": pid, "name": pname, "price": price,
			"quantity": qty, "size": sname, "temperature": tname,
			"price_adjustment": adj, "subtotal": itemPrice, "image_url": img,
		})
	}

	c.JSON(200, gin.H{
		"success": true, "message": "Cart retrieved",
		"data": gin.H{"items": items, "total": total},
	})
}
