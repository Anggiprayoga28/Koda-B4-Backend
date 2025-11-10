package controllers

import (
	"coffee-shop/models"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/matthewhartstonge/argon2"
)

type AuthController struct{}

func hashPassword(password string) (string, error) {
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(password))
	return string(encoded), err
}

func verifyPassword(hash, password string) bool {
	ok, _ := argon2.VerifyEncoded([]byte(password), []byte(hash))
	return ok
}

func generateToken(userID int, email, role string) (string, error) {
	secret := getEnv("JWT_SECRET", "secret")
	expiry, _ := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(expiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func uploadFile(c *gin.Context, file *multipart.FileHeader, subDir string) (string, error) {
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	if file.Size > 5*1024*1024 {
		return "", errors.New("file too large (max 5MB)")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := false
	for _, e := range allowedExts {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", errors.New("invalid file type")
	}

	uploadDir := filepath.Join(getEnv("UPLOAD_DIR", "./uploads"), subDir)
	os.MkdirAll(uploadDir, os.ModePerm)

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), strings.ReplaceAll(file.Filename, " ", "_"))
	fullPath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		return "", err
	}

	return filepath.Join(subDir, filename), nil
}

func deleteFile(path string) {
	if path != "" {
		os.Remove(filepath.Join(getEnv("UPLOAD_DIR", "./uploads"), path))
	}
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// Register godoc
// @Summary Register new user
// @Description Register a new customer account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Register Request"
// @Success 201 {object} models.Response
// @Failure 400 {object} models.ErrorResponse
// @Router /auth/register [post]
func (ctrl *AuthController) Register(c *gin.Context) {
	var req models.RegisterRequest
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
	err := models.DB.QueryRow(context.Background(),
		"INSERT INTO users (email, password, role, created_at, updated_at) VALUES ($1,$2, $5,$3,$4) RETURNING id",
		req.Email, hash, now, now, req.Role).Scan(&userID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Registration failed"})
		return
	}

	models.DB.Exec(context.Background(),
		"INSERT INTO user_profiles (user_id, full_name, phone, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)",
		userID, req.FullName, req.Phone, now, now)

	token, _ := generateToken(userID, req.Email, req.Role)

	c.JSON(201, gin.H{
		"success": true,
		"message": "Registration successful",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":        userID,
				"email":     req.Email,
				"role":      req.Role,
				"full_name": req.FullName,
				"phone":     req.Phone,
			},
		},
	})
}

// Login godoc
// @Summary User login
// @Description Login with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login Request"
// @Success 200 {object} models.Response
// @Failure 401 {object} models.ErrorResponse
// @Router /auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	var id int
	var email, password, role string
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, email, password, role FROM users WHERE email=$1", req.Email).Scan(&id, &email, &password, &role)

	if err != nil || !verifyPassword(password, req.Password) {
		c.JSON(401, gin.H{"success": false, "message": "Invalid credentials"})
		return
	}

	token, _ := generateToken(id, email, role)

	var fullName, phone, address, photoURL string
	models.DB.QueryRow(context.Background(),
		"SELECT COALESCE(full_name,''), COALESCE(phone,''), COALESCE(address,''), COALESCE(photo_url,'') FROM user_profiles WHERE user_id=$1",
		id).Scan(&fullName, &phone, &address, &photoURL)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Login successful",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":        id,
				"email":     email,
				"role":      role,
				"full_name": fullName,
				"phone":     phone,
				"address":   address,
				"photo_url": photoURL,
			},
		},
	})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get current user profile
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Response
// @Router /auth/profile [get]
func (ctrl *AuthController) GetProfile(c *gin.Context) {
	userID := c.GetInt("user_id")

	var email, role, fullName, phone, address, photoURL string
	models.DB.QueryRow(context.Background(),
		`SELECT u.email, u.role, COALESCE(p.full_name,''), COALESCE(p.phone,''), 
		COALESCE(p.address,''), COALESCE(p.photo_url,'') 
		FROM users u LEFT JOIN user_profiles p ON u.id=p.user_id WHERE u.id=$1`,
		userID).Scan(&email, &role, &fullName, &phone, &address, &photoURL)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Profile retrieved",
		"data": gin.H{
			"id":        userID,
			"email":     email,
			"role":      role,
			"full_name": fullName,
			"phone":     phone,
			"address":   address,
			"photo_url": photoURL,
		},
	})
}

// UpdateProfile godoc
// @Summary Update profile
// @Description Update user profile information
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.UpdateProfileRequest true "Update Request"
// @Success 200 {object} models.Response
// @Router /auth/profile [patch]
func (ctrl *AuthController) UpdateProfile(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	models.DB.Exec(context.Background(),
		"UPDATE user_profiles SET full_name=$1, phone=$2, address=$3, updated_at=$4 WHERE user_id=$5",
		req.FullName, req.Phone, req.Address, time.Now(), userID)

	c.JSON(200, gin.H{"success": true, "message": "Profile updated"})
}

// UpdateProfilePhoto godoc
// @Summary Update profile photo
// @Description Upload profile photo
// @Tags Authentication
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param photo formData file true "Photo file"
// @Success 200 {object} models.Response
// @Router /auth/profile/photo [post]
func (ctrl *AuthController) UpdateProfilePhoto(c *gin.Context) {
	userID := c.GetInt("user_id")

	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Photo required"})
		return
	}

	photoURL, err := uploadFile(c, file, "profiles")
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}

	var oldPhoto string
	models.DB.QueryRow(context.Background(), "SELECT photo_url FROM user_profiles WHERE user_id=$1", userID).Scan(&oldPhoto)
	deleteFile(oldPhoto)

	models.DB.Exec(context.Background(),
		"UPDATE user_profiles SET photo_url=$1, updated_at=$2 WHERE user_id=$3",
		photoURL, time.Now(), userID)

	c.JSON(200, gin.H{"success": true, "message": "Photo updated", "data": gin.H{"photo_url": photoURL}})
}

// ChangePassword godoc
// @Summary Change password
// @Description Change user password
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.ChangePasswordRequest true "Password Request"
// @Success 200 {object} models.Response
// @Router /auth/change-password [post]
func (ctrl *AuthController) ChangePassword(c *gin.Context) {
	userID := c.GetInt("user_id")

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	var currentHash string
	models.DB.QueryRow(context.Background(), "SELECT password FROM users WHERE id=$1", userID).Scan(&currentHash)

	if !verifyPassword(currentHash, req.OldPassword) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid old password"})
		return
	}

	newHash, _ := hashPassword(req.NewPassword)
	models.DB.Exec(context.Background(), "UPDATE users SET password=$1, updated_at=$2 WHERE id=$3",
		newHash, time.Now(), userID)

	c.JSON(200, gin.H{"success": true, "message": "Password changed"})
}
