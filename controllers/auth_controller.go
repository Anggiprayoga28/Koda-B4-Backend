package controllers

import (
	"coffee-shop/models"
	"context"
	"crypto/rand"
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
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct{}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func verifyPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

func generateOTP(length int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func isValidPassword(password string) bool {
	return len(password) >= 6
}

func isValidPhone(phone string) bool {
	re := regexp.MustCompile(`^[0-9+\-]{8,20}$`)
	return re.MatchString(phone)
}

func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "very-secret-key-change-me"
	}
	return secret
}

func generateToken(userID int, email, role string, expiry time.Duration) (string, error) {
	secret := getJWTSecret()

	if expiry <= 0 {
		expiry = time.Hour
	}

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
	if file == nil {
		return "", errors.New("no file provided")
	}

	uploadDir := filepath.Join("uploads", subDir)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
	path := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, path); err != nil {
		return "", err
	}

	return "/" + path, nil
}

func deleteFile(path string) {
	if path == "" {
		return
	}
	_ = os.Remove(strings.TrimPrefix(path, "/"))
}

// Register godoc
// @Summary Register user
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.RegisterRequest true "Register payload"
// @Success 201 {object} models.Response
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/register [post]
func (ctrl *AuthController) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid request payload",
			Error:   err.Error(),
		})
		return
	}

	email := strings.TrimSpace(req.Email)
	password := req.Password
	fullName := strings.TrimSpace(req.FullName)
	phone := strings.TrimSpace(req.Phone)
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "customer"
	}

	if !isValidEmail(email) {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid email format",
		})
		return
	}

	if !isValidPassword(password) {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Password must be at least 6 characters",
		})
		return
	}

	if len(fullName) < 3 {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Full name must be at least 3 characters",
		})
		return
	}

	if phone != "" && !isValidPhone(phone) {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid phone number",
		})
		return
	}

	if role != "customer" && role != "admin" {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Role must be 'customer' or 'admin'",
		})
		return
	}

	var exists int
	if err := models.DB.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM users WHERE email=$1", email,
	).Scan(&exists); err != nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Failed to check existing user",
		})
		return
	}
	if exists > 0 {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Email already exists",
		})
		return
	}

	hashed, _ := hashPassword(password)
	now := time.Now()

	var userID int
	err := models.DB.QueryRow(
		context.Background(),
		`INSERT INTO users (email, password, role, created_at, updated_at) 
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		email, hashed, role, now, now,
	).Scan(&userID)

	if err != nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Registration failed",
			Error:   err.Error(),
		})
		return
	}

	_, err = models.DB.Exec(
		context.Background(),
		`INSERT INTO user_profiles (user_id, full_name, phone, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, fullName, phone, now, now,
	)

	if err != nil {
		c.JSON(201, models.Response{
			Success: true,
			Message: "User registered successfully (profile pending)",
			Data: gin.H{
				"id":    userID,
				"email": email,
				"role":  role,
			},
		})
		return
	}

	c.JSON(201, models.Response{
		Success: true,
		Message: "User registered successfully",
		Data: gin.H{
			"id":    userID,
			"email": email,
			"role":  role,
		},
	})
}

// Login godoc
// @Summary Login user
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "Login payload"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid request payload",
			Error:   err.Error(),
		})
		return
	}

	email := strings.TrimSpace(req.Email)
	password := req.Password

	if !isValidEmail(email) {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid email format",
		})
		return
	}

	if password == "" {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Password is required",
		})
		return
	}

	var (
		id   int
		hash string
		role string
	)

	err := models.DB.QueryRow(
		context.Background(),
		"SELECT id, password, role FROM users WHERE email=$1",
		email,
	).Scan(&id, &hash, &role)

	if err != nil {
		c.JSON(401, models.ErrorResponse{
			Success: false,
			Message: "Invalid email or password",
		})
		return
	}

	if !verifyPassword(hash, password) {
		c.JSON(401, models.ErrorResponse{
			Success: false,
			Message: "Invalid email or password",
		})
		return
	}

	var (
		fullName string
		phone    string
		address  string
		photoURL string
	)

	profileErr := models.DB.QueryRow(
		context.Background(),
		`SELECT COALESCE(up.full_name, ''), COALESCE(up.phone, ''), 
		        COALESCE(up.address, ''), COALESCE(up.photo_url, '')
		 FROM users u
		 LEFT JOIN user_profiles up ON u.id = up.user_id
		 WHERE u.id = $1`,
		id,
	).Scan(&fullName, &phone, &address, &photoURL)

	if profileErr != nil {
		fullName = ""
		phone = ""
		address = ""
		photoURL = ""
	}

	token, err := generateToken(id, email, role, time.Hour)
	if err != nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Failed to generate token",
		})
		return
	}

	c.JSON(200, models.Response{
		Success: true,
		Message: "Login successful",
		Data: gin.H{
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

// ForgotPassword godoc
// @Summary Request OTP for password reset
// @Tags Auth
// @Accept json
// @Produce json
// @Param email body struct{Email string `json:"email"`} true "Email"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/forgot-password [post]
func (ctrl *AuthController) ForgotPassword(c *gin.Context) {
	var payload struct {
		Email string `json:"email" form:"email" binding:"required,email"`
	}
	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid email",
			Error:   err.Error(),
		})
		return
	}

	email := strings.TrimSpace(payload.Email)

	var userID int
	err := models.DB.QueryRow(context.Background(),
		"SELECT id FROM users WHERE email=$1",
		email,
	).Scan(&userID)
	if err != nil {
		c.JSON(200, models.Response{
			Success: true,
			Message: "If that email exists, an OTP has been sent",
		})
		return
	}

	otp, err := generateOTP(6)
	if err != nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Failed to generate OTP",
		})
		return
	}

	if models.RedisClient == nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "OTP service unavailable",
		})
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", strings.ToLower(email))
	if err := models.RedisClient.Set(ctx, key, otp, 5*time.Minute).Err(); err != nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Failed to store OTP",
		})
		return
	}

	emailService, err := models.NewEmailService()
	if err != nil {
		fmt.Printf("[OTP Generated - SMTP Not Configured]\n")
		fmt.Printf("Email: %s\nOTP: %s\nExpires: 5 minutes\n", email, otp)
	} else {
		_ = emailService.SendOTPEmail(email, otp)
	}

	c.JSON(200, models.Response{
		Success: true,
		Message: "If that email exists, an OTP has been sent",
	})
}

// VerifyOTP godoc
// @Summary Verify OTP and reset password
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body struct{
// @Param body body struct{Email string `json:"email"`; OTP string `json:"otp"`; NewPassword string `json:"new_password"`} true "Verify payload"
// @Success 200 {object} models.Response
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /auth/verify-otp [post]
func (ctrl *AuthController) VerifyOTP(c *gin.Context) {
	var payload struct {
		Email       string `json:"email" form:"email" binding:"required,email"`
		OTP         string `json:"otp" form:"otp" binding:"required"`
		NewPassword string `json:"new_password" form:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "Invalid request payload",
			Error:   err.Error(),
		})
		return
	}

	email := strings.TrimSpace(payload.Email)
	otp := strings.TrimSpace(payload.OTP)

	if models.RedisClient == nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "OTP service unavailable",
		})
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("otp:%s", strings.ToLower(email))

	stored, err := models.RedisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			c.JSON(400, models.ErrorResponse{
				Success: false,
				Message: "OTP is invalid or expired",
			})
			return
		}
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Failed to verify OTP",
		})
		return
	}

	if stored != otp {
		c.JSON(400, models.ErrorResponse{
			Success: false,
			Message: "OTP is invalid or expired",
		})
		return
	}

	hashed, _ := hashPassword(payload.NewPassword)

	_, err = models.DB.Exec(
		context.Background(),
		"UPDATE users SET password=$1, updated_at=$2 WHERE email=$3",
		hashed, time.Now(), email,
	)
	if err != nil {
		c.JSON(500, models.ErrorResponse{
			Success: false,
			Message: "Failed to reset password",
		})
		return
	}

	_ = models.RedisClient.Del(ctx, key).Err()

	c.JSON(200, models.Response{
		Success: true,
		Message: "Password reset successfully",
	})
}
