package controllers

import (
	"coffee-shop/models"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type ProfileController struct{}

// @Summary Get user profile
// @Description Get current user profile
// @Tags Profile
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Response
// @Router /profile [get]
func (ctrl *ProfileController) GetProfile(c *gin.Context) {
	userID := c.GetInt("user_id")

	var id int
	var email, role, fullName, phone, address, photoURL string
	var createdAt time.Time

	err := models.DB.QueryRow(context.Background(),
		`SELECT 
			u.id,
			u.email,
			u.role,
			u.created_at,
			COALESCE(p.full_name, '') as full_name,
			COALESCE(p.phone, '') as phone,
			COALESCE(p.address, '') as address,
			COALESCE(p.photo_url, '') as photo_url
		FROM users u 
		LEFT JOIN user_profiles p ON u.id = p.user_id 
		WHERE u.id = $1`,
		userID).Scan(&id, &email, &role, &createdAt, &fullName, &phone, &address, &photoURL)

	if err != nil {
		c.JSON(404, gin.H{
			"success": false,
			"message": "Profile not found",
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Profile retrieved successfully",
		"data": gin.H{
			"id":        id,
			"email":     email,
			"role":      role,
			"fullName":  fullName,
			"phone":     phone,
			"address":   address,
			"photoUrl":  photoURL,
			"createdAt": createdAt,
		},
	})
}

// @Summary Update profile
// @Description Update user profile information and change password
// @Tags Profile
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param full_name formData string false "Full Name"
// @Param phone formData string false "Phone"
// @Param address formData string false "Address"
// @Param photo formData file false "Profile photo"
// @Param old_password formData string false "Old Password"
// @Param new_password formData string false "New Password"
// @Param confirm_password formData string false "Confirm New Password"
// @Success 200 {object} models.Response
// @Router /profile [patch]
func (ctrl *ProfileController) UpdateProfile(c *gin.Context) {
	userID := c.GetInt("user_id")

	fullName := c.PostForm("full_name")
	phone := c.PostForm("phone")
	address := c.PostForm("address")
	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if fullName != "" && len(fullName) < 3 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Full name must be at least 3 characters",
		})
		return
	}

	if phone != "" && !isValidPhone(phone) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid phone number format",
		})
		return
	}

	if oldPassword != "" || newPassword != "" || confirmPassword != "" {
		if oldPassword == "" || newPassword == "" || confirmPassword == "" {
			c.JSON(400, gin.H{
				"success": false,
				"message": "All password fields are required for password change",
			})
			return
		}

		if !isValidPassword(newPassword) {
			c.JSON(400, gin.H{
				"success": false,
				"message": "New password must be at least 6 characters",
			})
			return
		}

		if newPassword != confirmPassword {
			c.JSON(400, gin.H{
				"success": false,
				"message": "New password and confirm password do not match",
			})
			return
		}

		if oldPassword == newPassword {
			c.JSON(400, gin.H{
				"success": false,
				"message": "New password must be different from old password",
			})
			return
		}

		var currentHash string
		err := models.DB.QueryRow(context.Background(),
			"SELECT password FROM users WHERE id=$1", userID).Scan(&currentHash)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to verify password",
			})
			return
		}

		if !verifyPassword(currentHash, oldPassword) {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid old password",
			})
			return
		}

		newHash, _ := hashPassword(newPassword)
		models.DB.Exec(context.Background(),
			"UPDATE users SET password=$1, updated_at=$2 WHERE id=$3",
			newHash, time.Now(), userID)
	}

	photoURL := ""
	file, err := c.FormFile("photo")
	if err == nil {
		uploadedPath, uploadErr := uploadFile(c, file, "profiles")
		if uploadErr != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": uploadErr.Error(),
			})
			return
		}

		photoURL = uploadedPath

		var oldPhoto string
		models.DB.QueryRow(context.Background(),
			"SELECT COALESCE(photo_url, '') FROM user_profiles WHERE user_id=$1", userID).Scan(&oldPhoto)
		if oldPhoto != "" {
			deleteFile(oldPhoto)
		}
	}

	now := time.Now()
	if photoURL != "" {
		models.DB.Exec(context.Background(),
			"UPDATE user_profiles SET full_name=$1, phone=$2, address=$3, photo_url=$4, updated_at=$5 WHERE user_id=$6",
			fullName, phone, address, photoURL, now, userID)
	} else {
		models.DB.Exec(context.Background(),
			"UPDATE user_profiles SET full_name=$1, phone=$2, address=$3, updated_at=$4 WHERE user_id=$5",
			fullName, phone, address, now, userID)
	}

	responseData := gin.H{
		"success": true,
		"message": "Profile updated successfully",
	}

	if photoURL != "" {
		responseData["data"] = gin.H{
			"photoUrl": photoURL,
		}
	}

	c.JSON(200, responseData)
}
