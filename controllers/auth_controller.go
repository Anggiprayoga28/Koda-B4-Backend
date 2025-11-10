package controllers

import (
	"coffee-shop/models"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
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

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

func isValidPassword(password string) bool {
	return len(password) >= 6
}

func isValidPhone(phone string) bool {
	if phone == "" {
		return true
	}
	pattern := `^[\d+\-\s()]+$`
	match, _ := regexp.MatchString(pattern, phone)
	return match && len(phone) >= 10
}

// Register godoc
// @Summary Register new user
// @Description Register a new customer account
// @Tags Authentication
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Param full_name formData string true "Full Name"
// @Param phone formData string false "Phone"
// @Param role formData string false "Role"
// @Success 201 {object} models.Response
// @Failure 400 {object} models.ErrorResponse
// @Router /auth/register [post]
func (ctrl *AuthController) Register(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	fullName := strings.TrimSpace(c.PostForm("full_name"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	role := c.DefaultPostForm("role", "customer")

	if email == "" || password == "" || fullName == "" {
		c.JSON(400, gin.H{"success": false, "message": "Email, password, and full_name are required"})
		return
	}

	if !isValidEmail(email) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid email format"})
		return
	}

	if !isValidPassword(password) {
		c.JSON(400, gin.H{"success": false, "message": "Password must be at least 6 characters"})
		return
	}

	if len(fullName) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Full name must be at least 3 characters"})
		return
	}

	if !isValidPhone(phone) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid phone number"})
		return
	}

	if role != "customer" && role != "admin" {
		c.JSON(400, gin.H{"success": false, "message": "Role must be 'customer' or 'admin'"})
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
	err := models.DB.QueryRow(context.Background(),
		"INSERT INTO users (email, password, role, created_at, updated_at) VALUES ($1,$2,$3,$4,$5) RETURNING id",
		email, hash, role, now, now).Scan(&userID)

	if err != nil {
		c.JSON(500, gin.H{"success": false, "message": "Registration failed"})
		return
	}

	models.DB.Exec(context.Background(),
		"INSERT INTO user_profiles (user_id, full_name, phone, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)",
		userID, fullName, phone, now, now)

	token, _ := generateToken(userID, email, role)

	c.JSON(201, gin.H{
		"success": true,
		"message": "Registration successful",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":        userID,
				"email":     email,
				"role":      role,
				"full_name": fullName,
				"phone":     phone,
			},
		},
	})
}

// Login godoc
// @Summary User login
// @Description Login with email and password
// @Tags Authentication
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Success 200 {object} models.Response
// @Failure 401 {object} models.ErrorResponse
// @Router /auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")

	if email == "" || password == "" {
		c.JSON(400, gin.H{"success": false, "message": "Email and password are required"})
		return
	}

	if !isValidEmail(email) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid email format"})
		return
	}

	var id int
	var emailDB, passwordDB, role string
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, email, password, role FROM users WHERE email=$1", email).Scan(&id, &emailDB, &passwordDB, &role)

	if err != nil || !verifyPassword(passwordDB, password) {
		c.JSON(401, gin.H{"success": false, "message": "Invalid credentials"})
		return
	}

	token, _ := generateToken(id, emailDB, role)

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
				"email":     emailDB,
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
// @Accept multipart/form-data
// @Produce json
// @Param full_name formData string false "Full Name"
// @Param phone formData string false "Phone"
// @Param address formData string false "Address"
// @Success 200 {object} models.Response
// @Router /auth/profile [patch]
func (ctrl *AuthController) UpdateProfile(c *gin.Context) {
	userID := c.GetInt("user_id")

	fullName := strings.TrimSpace(c.PostForm("full_name"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	address := strings.TrimSpace(c.PostForm("address"))

	if fullName != "" && len(fullName) < 3 {
		c.JSON(400, gin.H{"success": false, "message": "Full name must be at least 3 characters"})
		return
	}

	if !isValidPhone(phone) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid phone number"})
		return
	}

	models.DB.Exec(context.Background(),
		"UPDATE user_profiles SET full_name=$1, phone=$2, address=$3, updated_at=$4 WHERE user_id=$5",
		fullName, phone, address, time.Now(), userID)

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
// @Accept multipart/form-data
// @Produce json
// @Param old_password formData string true "Old Password"
// @Param new_password formData string true "New Password"
// @Success 200 {object} models.Response
// @Router /auth/change-password [post]
func (ctrl *AuthController) ChangePassword(c *gin.Context) {
	userID := c.GetInt("user_id")

	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")

	if oldPassword == "" || newPassword == "" {
		c.JSON(400, gin.H{"success": false, "message": "Old and new password are required"})
		return
	}

	if !isValidPassword(newPassword) {
		c.JSON(400, gin.H{"success": false, "message": "New password must be at least 6 characters"})
		return
	}

	if oldPassword == newPassword {
		c.JSON(400, gin.H{"success": false, "message": "New password must be different from old password"})
		return
	}

	var currentHash string
	models.DB.QueryRow(context.Background(), "SELECT password FROM users WHERE id=$1", userID).Scan(&currentHash)

	if !verifyPassword(currentHash, oldPassword) {
		c.JSON(400, gin.H{"success": false, "message": "Invalid old password"})
		return
	}

	newHash, _ := hashPassword(newPassword)
	models.DB.Exec(context.Background(), "UPDATE users SET password=$1, updated_at=$2 WHERE id=$3",
		newHash, time.Now(), userID)

	c.JSON(200, gin.H{"success": true, "message": "Password changed"})
}
