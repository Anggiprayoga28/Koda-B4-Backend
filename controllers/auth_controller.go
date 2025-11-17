package controllers

import (
	"coffee-shop/models"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
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

func generateOTP() string {
	otp := ""
	for i := 0; i < 6; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(10))
		otp += num.String()
	}
	return otp
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
		c.JSON(400, gin.H{
			"success": false,
			"message": "Email, password, and full_name are required",
		})
		return
	}

	if !isValidEmail(email) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid email format",
		})
		return
	}

	if !isValidPassword(password) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Password must be at least 6 characters",
		})
		return
	}

	if len(fullName) < 3 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Full name must be at least 3 characters",
		})
		return
	}

	if !isValidPhone(phone) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid phone number",
		})
		return
	}

	if role != "customer" && role != "admin" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Role must be 'customer' or 'admin'",
		})
		return
	}

	var exists int
	models.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email=$1", email).Scan(&exists)
	if exists > 0 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Email already exists",
		})
		return
	}

	hash, _ := hashPassword(password)
	now := time.Now()

	var userID int
	err := models.DB.QueryRow(context.Background(),
		"INSERT INTO users (email, password, role, created_at, updated_at) VALUES ($1,$2,$3,$4,$5) RETURNING id",
		email, hash, role, now, now).Scan(&userID)

	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Registration failed",
		})
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
				"id":       userID,
				"email":    email,
				"role":     role,
				"fullName": fullName,
				"phone":    phone,
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
		c.JSON(400, gin.H{
			"success": false,
			"message": "Email and password are required",
		})
		return
	}

	if !isValidEmail(email) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid email format",
		})
		return
	}

	var id int
	var emailDB, passwordDB, role string
	err := models.DB.QueryRow(context.Background(),
		"SELECT id, email, password, role FROM users WHERE email=$1", email).Scan(&id, &emailDB, &passwordDB, &role)

	if err != nil || !verifyPassword(passwordDB, password) {
		c.JSON(401, gin.H{
			"success": false,
			"message": "Invalid credentials",
		})
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
				"id":       id,
				"email":    emailDB,
				"role":     role,
				"fullName": fullName,
				"phone":    phone,
				"address":  address,
				"photoUrl": photoURL,
			},
		},
	})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send OTP
// @Tags Authentication
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "Email"
// @Success 200 {object} models.Response
// @Router /auth/forgot-password [post]
func (ctrl *AuthController) ForgotPassword(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))

	if email == "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Email is required",
		})
		return
	}

	if !isValidEmail(email) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid email format",
		})
		return
	}

	var userID int
	err := models.DB.QueryRow(context.Background(),
		"SELECT id FROM users WHERE email=$1", email).Scan(&userID)

	if err != nil {
		c.JSON(404, gin.H{
			"success": false,
			"message": "Email not found",
		})
		return
	}

	otp := generateOTP()

	if models.RedisClient != nil {
		ctx := context.Background()
		redisKey := fmt.Sprintf("otp:%s", email)

		err = models.RedisClient.Set(ctx, redisKey, otp, 5*time.Minute).Err()
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to generate OTP",
			})
			return
		}

		fmt.Printf("[OTP Generated] Email: %s, OTP: %s, Expires: 5 minutes\n", email, otp)
	} else {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Redis not available",
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "OTP sent successfully",
		"data": gin.H{
			"otp":       otp,
			"expiresIn": "5 minutes",
		},
	})
}

// VerifyOTP godoc
// @Summary Verify OTP and reset password
// @Description Verify OTP code and immediately reset password
// @Tags Authentication
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "Email"
// @Param otp formData string true "OTP Code"
// @Param new_password formData string true "New Password"
// @Param confirm_password formData string true "Confirm Password"
// @Success 200 {object} models.Response
// @Router /auth/verify-otp [post]
func (ctrl *AuthController) VerifyOTP(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	otp := strings.TrimSpace(c.PostForm("otp"))
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if email == "" || otp == "" || newPassword == "" || confirmPassword == "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "All fields are required",
		})
		return
	}

	if !isValidEmail(email) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid email format",
		})
		return
	}

	if len(otp) != 6 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "OTP must be 6 digits",
		})
		return
	}

	if !isValidPassword(newPassword) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Password must be at least 6 characters",
		})
		return
	}

	if newPassword != confirmPassword {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Passwords do not match",
		})
		return
	}

	if models.RedisClient == nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Redis not available",
		})
		return
	}

	ctx := context.Background()
	redisKey := fmt.Sprintf("otp:%s", email)

	storedOTP, err := models.RedisClient.Get(ctx, redisKey).Result()
	if err != nil {
		c.JSON(404, gin.H{
			"success": false,
			"message": "OTP not found or expired",
		})
		return
	}

	if storedOTP != otp {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid OTP code",
		})
		return
	}

	models.RedisClient.Del(ctx, redisKey)

	var userID int
	err = models.DB.QueryRow(context.Background(),
		"SELECT id FROM users WHERE email=$1", email).Scan(&userID)
	if err != nil {
		c.JSON(404, gin.H{
			"success": false,
			"message": "User not found",
		})
		return
	}

	newHash, _ := hashPassword(newPassword)
	_, err = models.DB.Exec(context.Background(),
		"UPDATE users SET password=$1, updated_at=$2 WHERE id=$3",
		newHash, time.Now(), userID)

	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to reset password",
		})
		return
	}

	fmt.Printf("[Password Reset] Email: %s, User ID: %d\n", email, userID)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Password reset successfully. You can now login with your new password.",
	})
}
