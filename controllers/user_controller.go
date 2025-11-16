package controllers

import (
	"coffee-shop/models"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

func (ctrl *UserController) getPaginationParams(c *gin.Context, defaultLimit int) (page, limit, offset int) {
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

func (ctrl *UserController) generateLinks(c *gin.Context, page, limit, totalPages int) models.PaginationLinks {
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

func (ctrl *UserController) buildResponse(c *gin.Context, message string, data interface{}, page, limit, totalItems int) models.HATEOASResponse {
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

// @Summary Get all users
// @Description Get users
// @Tags Admin - Users
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} models.HATEOASResponse
// @Router /admin/users [get]
func (ctrl *UserController) GetAllUsers(c *gin.Context) {
	page, limit, offset := ctrl.getPaginationParams(c, 10)

	var total int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&total)

	rows, _ := models.DB.Query(context.Background(),
		`SELECT u.id, u.email, u.role, u.created_at, COALESCE(p.full_name,''), COALESCE(p.phone,''), 
		COALESCE(p.address,''), COALESCE(p.photo_url,'') 
		FROM users u LEFT JOIN user_profiles p ON u.id=p.user_id ORDER BY u.created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	defer rows.Close()

	users := []gin.H{}
	for rows.Next() {
		var id int
		var email, role, fullName, phone, address, photoURL string
		var createdAt time.Time
		rows.Scan(&id, &email, &role, &createdAt, &fullName, &phone, &address, &photoURL)
		users = append(users, gin.H{
			"id": id, "email": email, "role": role, "created_at": createdAt,
			"full_name": fullName, "phone": phone, "address": address, "photo_url": photoURL,
		})
	}

	response := ctrl.buildResponse(c, "Users retrieved successfully", users, page, limit, total)
	c.JSON(200, response)
}

// @Summary Get user by ID
// @Description Get user details (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.Response
// @Failure 404 {object} models.ErrorResponse
// @Router /admin/users/{id} [get]
func (ctrl *UserController) GetUserByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid user ID"})
		return
	}

	var email, role, fullName, phone, address, photoURL string
	var createdAt time.Time
	err := models.DB.QueryRow(context.Background(),
		`SELECT u.email, u.role, u.created_at, COALESCE(p.full_name,''), COALESCE(p.phone,''), 
		COALESCE(p.address,''), COALESCE(p.photo_url,'') 
		FROM users u LEFT JOIN user_profiles p ON u.id=p.user_id WHERE u.id=$1`,
		id).Scan(&email, &role, &createdAt, &fullName, &phone, &address, &photoURL)

	if err != nil {
		c.JSON(404, gin.H{"success": false, "message": "User not found"})
		return
	}

	c.JSON(200, gin.H{
		"success": true, "message": "User retrieved",
		"data": gin.H{
			"id": id, "email": email, "role": role, "created_at": createdAt,
			"full_name": fullName, "phone": phone, "address": address, "photo_url": photoURL,
		},
	})
}

func isValidEmailUser(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

func isValidPasswordUser(password string) bool {
	return len(password) >= 6
}

func isValidPhoneUser(phone string) bool {
	if phone == "" {
		return true
	}
	pattern := `^[\d+\-\s()]+$`
	match, _ := regexp.MatchString(pattern, phone)
	return match && len(phone) >= 10
}

// @Summary Create user
// @Description Create new user (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Param role formData string true "Role (admin/customer)"
// @Param full_name formData string false "Full Name"
// @Param phone formData string false "Phone"
// @Success 201 {object} models.Response
// @Router /admin/users [post]
func (ctrl *UserController) CreateUser(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	role := strings.TrimSpace(c.PostForm("role"))
	fullName := strings.TrimSpace(c.PostForm("full_name"))
	phone := strings.TrimSpace(c.PostForm("phone"))

	if email == "" || password == "" || role == "" {
		c.JSON(400, gin.H{"success": false, "message": "Email, password, and role are required"})
		return
	}

	if !isValidEmailUser(email) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid email format"})
		return
	}

	if !isValidPasswordUser(password) {
		c.JSON(400, gin.H{"success": false, "message": "Password must be at least 6 characters"})
		return
	}

	if role != "admin" && role != "customer" {
		c.JSON(400, gin.H{"success": false, "message": "Role must be 'admin' or 'customer'"})
		return
	}

	if fullName != "" && len(fullName) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Full name must be at least 3 characters"})
		return
	}

	if !isValidPhoneUser(phone) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid phone number"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email=$1", email).Scan(&exists)
	if exists > 0 {
		c.JSON(400, gin.H{"success": false, "message": "Email already exists"})
		return
	}

	hash, _ := hashPassword(password)
	now := time.Now()

	var userID int
	models.DB.QueryRow(context.Background(),
		"INSERT INTO users (email, password, role, created_at, updated_at) VALUES ($1,$2,$3,$4,$5) RETURNING id",
		email, hash, role, now, now).Scan(&userID)

	models.DB.Exec(context.Background(),
		"INSERT INTO user_profiles (user_id, full_name, phone, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)",
		userID, fullName, phone, now, now)

	c.JSON(201, gin.H{
		"success": true, "message": "User created",
		"data": gin.H{"id": userID, "email": email, "role": role, "full_name": fullName},
	})
}

// @Summary Update user
// @Description Update user (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "User ID"
// @Param email formData string false "Email"
// @Param role formData string false "Role"
// @Param full_name formData string false "Full Name"
// @Param phone formData string false "Phone"
// @Param address formData string false "Address"
// @Success 200 {object} models.Response
// @Router /admin/users/{id} [patch]
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid user ID"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE id=$1", id).Scan(&exists)
	if exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "User not found"})
		return
	}

	email := strings.TrimSpace(c.PostForm("email"))
	role := strings.TrimSpace(c.PostForm("role"))
	fullName := strings.TrimSpace(c.PostForm("full_name"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	address := strings.TrimSpace(c.PostForm("address"))

	if email != "" {
		if !isValidEmailUser(email) {
			c.JSON(400, gin.H{"success": false, "message": "Invalid email format"})
			return
		}

		var emailExists int
		models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email=$1 AND id!=$2", email, id).Scan(&emailExists)
		if emailExists > 0 {
			c.JSON(400, gin.H{"success": false, "message": "Email already exists"})
			return
		}

		models.DB.Exec(context.Background(), "UPDATE users SET email=$1, updated_at=$2 WHERE id=$3",
			email, time.Now(), id)
	}

	if role != "" {
		if role != "admin" && role != "customer" {
			c.JSON(400, gin.H{"success": false, "message": "Role must be 'admin' or 'customer'"})
			return
		}

		models.DB.Exec(context.Background(), "UPDATE users SET role=$1, updated_at=$2 WHERE id=$3",
			role, time.Now(), id)
	}

	if fullName != "" && len(fullName) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Full name must be at least 3 characters"})
		return
	}

	if !isValidPhoneUser(phone) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid phone number"})
		return
	}

	models.DB.Exec(context.Background(),
		"UPDATE user_profiles SET full_name=$1, phone=$2, address=$3, updated_at=$4 WHERE user_id=$5",
		fullName, phone, address, time.Now(), id)

	c.JSON(200, gin.H{"success": true, "message": "User updated"})
}

// @Summary Delete user
// @Description Delete user (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.Response
// @Router /admin/users/{id} [delete]
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if id <= 0 {
		c.JSON(400, gin.H{"success": false, "message": "Invalid user ID"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE id=$1", id).Scan(&exists)
	if exists == 0 {
		c.JSON(404, gin.H{"success": false, "message": "User not found"})
		return
	}

	var photoURL string
	models.DB.QueryRow(context.Background(), "SELECT photo_url FROM user_profiles WHERE user_id=$1", id).Scan(&photoURL)
	deleteFile(photoURL)

	models.DB.Exec(context.Background(), "DELETE FROM user_profiles WHERE user_id=$1", id)
	models.DB.Exec(context.Background(), "DELETE FROM users WHERE id=$1", id)

	c.JSON(200, gin.H{"success": true, "message": "User deleted"})
}
