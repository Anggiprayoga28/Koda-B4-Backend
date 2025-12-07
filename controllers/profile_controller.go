package controllers

import (
	"coffee-shop/libs"
	"coffee-shop/models"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type ProfileController struct {
	uploadFolder string
}

func NewProfileController() *ProfileController {
	uploadFolder := "./uploads"
	os.MkdirAll(uploadFolder, os.ModePerm)

	return &ProfileController{
		uploadFolder: uploadFolder,
	}
}

type ProfileResponse struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	FullName  string    `json:"fullName"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	PhotoURL  string    `json:"photoUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type UpdateProfileRequest struct {
	FullName        string `form:"full_name" binding:"max=100"`
	Phone           string `form:"phone" binding:"max=20"`
	Address         string `form:"address" binding:"max=500"`
	OldPassword     string `form:"old_password" binding:"max=100"`
	NewPassword     string `form:"new_password" binding:"max=100"`
	ConfirmPassword string `form:"confirm_password" binding:"max=100"`
}

type UpdateProfileResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// @Summary Get user profile
// @Description Get current user profile
// @Tags Profile
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.Response
// @Router /profile [get]
func (ctrl *ProfileController) GetProfile(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(401, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	var profile ProfileResponse

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
		WHERE u.id = $1 AND u.deleted_at IS NULL`,
		userID).Scan(
		&profile.ID,
		&profile.Email,
		&profile.Role,
		&profile.CreatedAt,
		&profile.FullName,
		&profile.Phone,
		&profile.Address,
		&profile.PhotoURL,
	)

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
		"data":    profile,
	})
}

// @Summary Update profile
// @Description Update user profile information and upload photo to Cloudinary
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
	if userID == 0 {
		c.JSON(401, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid request data: " + err.Error(),
		})
		return
	}

	if req.FullName != "" && len(req.FullName) < 3 {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Full name must be at least 3 characters",
		})
		return
	}

	if req.Phone != "" && !isValidPhone(req.Phone) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid phone number format",
		})
		return
	}

	if req.OldPassword != "" || req.NewPassword != "" || req.ConfirmPassword != "" {
		if err := ctrl.handlePasswordChange(c, userID, req); err != nil {
			return
		}
	}

	photoURL, cloudinaryPublicID, shouldUpdatePhoto, err := ctrl.handlePhotoUpload(c, userID)
	if err != nil {
		return
	}

	if err := ctrl.updateProfileInDB(c, userID, req, photoURL, cloudinaryPublicID, shouldUpdatePhoto); err != nil {
		return
	}

	response := UpdateProfileResponse{
		Success: true,
		Message: "Profile updated successfully",
	}

	if photoURL != "" {
		response.Data = map[string]interface{}{
			"photoUrl": photoURL,
		}
	}

	c.JSON(200, response)
}

func (ctrl *ProfileController) handlePasswordChange(c *gin.Context, userID int, req UpdateProfileRequest) error {
	if req.OldPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "All password fields are required for password change",
		})
		return fmt.Errorf("missing password fields")
	}

	if !isValidPassword(req.NewPassword) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "New password must be at least 6 characters",
		})
		return fmt.Errorf("invalid password length")
	}

	if req.NewPassword != req.ConfirmPassword {
		c.JSON(400, gin.H{
			"success": false,
			"message": "New password and confirm password do not match",
		})
		return fmt.Errorf("password mismatch")
	}

	if req.OldPassword == req.NewPassword {
		c.JSON(400, gin.H{
			"success": false,
			"message": "New password must be different from old password",
		})
		return fmt.Errorf("same old and new password")
	}

	var currentHash string
	err := models.DB.QueryRow(context.Background(),
		"SELECT password FROM users WHERE id=$1 AND deleted_at IS NULL", userID).Scan(&currentHash)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to verify password",
		})
		return err
	}

	if !verifyPassword(currentHash, req.OldPassword) {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid old password",
		})
		return fmt.Errorf("invalid old password")
	}

	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to hash password",
		})
		return err
	}

	_, err = models.DB.Exec(context.Background(),
		"UPDATE users SET password=$1, updated_at=$2 WHERE id=$3",
		newHash, time.Now(), userID)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to update password",
		})
		return err
	}

	return nil
}

func (ctrl *ProfileController) handlePhotoUpload(c *gin.Context, userID int) (string, string, bool, error) {
	file, err := c.FormFile("photo")
	if err != nil || file == nil {
		fmt.Printf("[Controller] No photo file in request (err: %v)\n", err)
		return "", "", false, nil
	}

	fmt.Printf("[Controller] Photo upload - user_id=%d, file=%s, size=%d\n",
		userID, file.Filename, file.Size)

	if file.Size > (5 * 1024 * 1024) {
		fmt.Printf("[Controller] File too large: %d bytes\n", file.Size)
		c.JSON(400, gin.H{
			"success": false,
			"message": "File terlalu besar (max 5MB)",
		})
		return "", "", false, fmt.Errorf("file too large")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}
	isValid := false
	for _, allowed := range allowedExts {
		if ext == allowed {
			isValid = true
			break
		}
	}

	if !isValid {
		fmt.Printf("[Controller] Invalid file format: %s\n", ext)
		c.JSON(400, gin.H{
			"success": false,
			"message": "Format image salah. Hanya " + strings.Join(allowedExts, ", "),
		})
		return "", "", false, fmt.Errorf("invalid image format")
	}

	var oldCloudinaryPublicID string
	err = models.DB.QueryRow(context.Background(),
		"SELECT cloudinary_public_id FROM user_profiles WHERE user_id=$1",
		userID).Scan(&oldCloudinaryPublicID)

	if err != nil {
		fmt.Printf("[Controller] No existing profile found or error: %v\n", err)
		oldCloudinaryPublicID = ""
	} else {
		fmt.Printf("[Controller] Found old Cloudinary Public ID: %s\n", oldCloudinaryPublicID)
	}

	fmt.Printf("[Controller] Saving file temporarily to: %s\n", ctrl.uploadFolder)
	localPath, err := libs.SaveUploadedFile(c, file, ctrl.uploadFolder)
	if err != nil {
		fmt.Printf("[Controller] Failed to save file locally: %v\n", err)
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to save uploaded file: " + err.Error(),
		})
		return "", "", false, err
	}

	fmt.Printf("[Controller] File saved locally: %s\n", localPath)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		fmt.Printf("[Controller] Local file doesn't exist after save!\n")
		c.JSON(500, gin.H{
			"success": false,
			"message": "File was not saved correctly",
		})
		return "", "", false, fmt.Errorf("file not saved")
	}

	fmt.Printf("[Controller] Calling libs.UploadToCloudinary...\n")
	cloudinaryURL, err := libs.UploadToCloudinary(localPath)
	if err != nil {
		fmt.Printf("[Controller] Failed to upload to Cloudinary: %v\n", err)

		if _, err := os.Stat(localPath); err == nil {
			fmt.Printf("[Controller] Removing failed upload file: %s\n", localPath)
			os.Remove(localPath)
		}

		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to upload photo to Cloudinary: " + err.Error(),
		})
		return "", "", false, err
	}

	if cloudinaryURL == "" {
		fmt.Printf("[Controller] Cloudinary returned empty URL!\n")
		c.JSON(500, gin.H{
			"success": false,
			"message": "Cloudinary returned empty URL",
		})
		return "", "", false, fmt.Errorf("cloudinary returned empty url")
	}

	fmt.Printf("[Controller] Cloudinary URL received: %s\n", cloudinaryURL)

	cloudinaryPublicID := extractPublicIDFromURL(cloudinaryURL)
	fmt.Printf("[Controller] Extracted Public ID: %s\n", cloudinaryPublicID)

	if oldCloudinaryPublicID != "" && cloudinaryPublicID != "" {
		fmt.Printf("[Controller] Queueing old photo deletion: %s\n", oldCloudinaryPublicID)
		go func(oldID string) {
			time.Sleep(2 * time.Second)
			if err := libs.DeleteFromCloudinary(oldID); err != nil {
				fmt.Printf("[Controller] Failed to delete old photo: %v\n", err)
			} else {
				fmt.Printf("[Controller] Old photo deleted successfully\n")
			}
		}(oldCloudinaryPublicID)
	}

	return cloudinaryURL, cloudinaryPublicID, true, nil
}

func extractPublicIDFromURL(url string) string {
	if url == "" {
		return ""
	}

	fmt.Printf("[extractPublicID] Parsing URL: %s\n", url)

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return ""
	}

	uploadIndex := -1
	for i, part := range parts {
		if part == "upload" {
			uploadIndex = i
			break
		}
	}

	if uploadIndex == -1 || uploadIndex >= len(parts)-1 {
		return ""
	}

	publicIDParts := parts[uploadIndex+1:]

	publicIDWithExt := strings.Join(publicIDParts, "/")

	if dotIndex := strings.LastIndex(publicIDWithExt, "."); dotIndex != -1 {
		publicIDWithExt = publicIDWithExt[:dotIndex]
	}

	if qIndex := strings.Index(publicIDWithExt, "?"); qIndex != -1 {
		publicIDWithExt = publicIDWithExt[:qIndex]
	}

	fmt.Printf("[extractPublicID] Result: %s\n", publicIDWithExt)
	return publicIDWithExt
}

func (ctrl *ProfileController) updateProfileInDB(c *gin.Context, userID int, req UpdateProfileRequest,
	photoURL, cloudinaryPublicID string, shouldUpdatePhoto bool) error {

	ctx := context.Background()
	now := time.Now()

	tx, err := models.DB.Begin(ctx)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to start transaction",
		})
		return err
	}
	defer tx.Rollback(ctx)

	var profileExists bool
	err = tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM user_profiles WHERE user_id=$1)", userID).
		Scan(&profileExists)

	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to check profile existence",
		})
		return err
	}

	fmt.Printf("Profile exists: %v, shouldUpdatePhoto: %v\n", profileExists, shouldUpdatePhoto)
	fmt.Printf("Photo URL to save: %s\n", photoURL)
	fmt.Printf("Cloudinary Public ID to save: %s\n", cloudinaryPublicID)

	var result pgconn.CommandTag

	if profileExists {
		if shouldUpdatePhoto && photoURL != "" {
			fmt.Printf("Updating profile WITH photo for user_id=%d\n", userID)
			query := `UPDATE user_profiles SET 
				full_name = COALESCE(NULLIF($1, ''), full_name),
				phone = COALESCE(NULLIF($2, ''), phone),
				address = COALESCE(NULLIF($3, ''), address),
				photo_url = $4,
				cloudinary_public_id = $5,
				updated_at = $6
				WHERE user_id = $7`

			result, err = tx.Exec(ctx, query,
				req.FullName,
				req.Phone,
				req.Address,
				photoURL,
				cloudinaryPublicID,
				now,
				userID,
			)
		} else {
			fmt.Printf("Updating profile WITHOUT photo for user_id=%d\n", userID)
			query := `UPDATE user_profiles SET 
				full_name = COALESCE(NULLIF($1, ''), full_name),
				phone = COALESCE(NULLIF($2, ''), phone),
				address = COALESCE(NULLIF($3, ''), address),
				updated_at = $4
				WHERE user_id = $5`

			result, err = tx.Exec(ctx, query,
				req.FullName,
				req.Phone,
				req.Address,
				now,
				userID,
			)
		}
	} else {
		fmt.Printf("Inserting NEW profile for user_id=%d\n", userID)
		query := `INSERT INTO user_profiles (
			user_id, full_name, phone, address, 
			photo_url, cloudinary_public_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

		result, err = tx.Exec(ctx, query,
			userID,
			req.FullName,
			req.Phone,
			req.Address,
			photoURL,
			cloudinaryPublicID,
			now,
			now,
		)
	}

	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to update profile: " + err.Error(),
		})
		return err
	}

	fmt.Printf("Database update successful: %v rows affected\n", result.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		fmt.Printf("Failed to commit transaction: %v\n", err)
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to commit profile update",
		})
		return err
	}

	var savedPhotoURL string
	_ = models.DB.QueryRow(context.Background(),
		"SELECT COALESCE(photo_url, '') FROM user_profiles WHERE user_id=$1",
		userID).Scan(&savedPhotoURL)

	fmt.Printf("Verified photo_url in database: %s\n", savedPhotoURL)

	return nil
}
