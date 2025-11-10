package controllers

import (
	"coffee-shop/models"
	"context"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

// @Summary Get all users
// @Description Get paginated users (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} models.PaginationResponse
// @Router /admin/users [get]
func (ctrl *UserController) GetAllUsers(c *gin.Context) {
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

	c.JSON(200, gin.H{
		"success": true, "message": "Users retrieved", "data": users,
		"meta": gin.H{"page": page, "limit": limit, "total_items": total, "total_pages": int(math.Ceil(float64(total) / float64(limit)))},
	})
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

// @Summary Create user
// @Description Create new user (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "User Request"
// @Success 201 {object} models.Response
// @Router /admin/users [post]
func (ctrl *UserController) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email=$1", req.Email).Scan(&exists)
	if exists > 0 {
		c.JSON(400, gin.H{"success": false, "message": "Email already exists"})
		return
	}

	hash, _ := hashPassword(req.Password)
	now := time.Now()

	var userID int
	models.DB.QueryRow(context.Background(),
		"INSERT INTO users (email, password, role, created_at, updated_at) VALUES ($1,$2,$3,$4,$5) RETURNING id",
		req.Email, hash, req.Role, now, now).Scan(&userID)

	models.DB.Exec(context.Background(),
		"INSERT INTO user_profiles (user_id, full_name, phone, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)",
		userID, req.FullName, req.Phone, now, now)

	c.JSON(201, gin.H{
		"success": true, "message": "User created",
		"data": gin.H{"id": userID, "email": req.Email, "role": req.Role, "full_name": req.FullName},
	})
}

// @Summary Update user
// @Description Update user (Admin)
// @Tags Admin - Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body models.UpdateUserRequest true "User Request"
// @Success 200 {object} models.Response
// @Router /admin/users/{id} [patch]
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	if req.Email != "" {
		var exists int
		models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email=$1 AND id!=$2", req.Email, id).Scan(&exists)
		if exists > 0 {
			c.JSON(400, gin.H{"success": false, "message": "Email already exists"})
			return
		}
	}

	models.DB.Exec(context.Background(), "UPDATE users SET email=$1, role=$2, updated_at=$3 WHERE id=$4",
		req.Email, req.Role, time.Now(), id)

	models.DB.Exec(context.Background(),
		"UPDATE user_profiles SET full_name=$1, phone=$2, address=$3, updated_at=$4 WHERE user_id=$5",
		req.FullName, req.Phone, req.Address, time.Now(), id)

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

	var photoURL string
	models.DB.QueryRow(context.Background(), "SELECT photo_url FROM user_profiles WHERE user_id=$1", id).Scan(&photoURL)
	deleteFile(photoURL)

	models.DB.Exec(context.Background(), "DELETE FROM user_profiles WHERE user_id=$1", id)
	models.DB.Exec(context.Background(), "DELETE FROM users WHERE id=$1", id)

	c.JSON(200, gin.H{"success": true, "message": "User deleted"})
}
