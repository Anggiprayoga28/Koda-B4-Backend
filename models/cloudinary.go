package models

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryService() (*CloudinaryService, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	fmt.Printf(" Initializing Cloudinary...\n")
	fmt.Printf("   Cloud Name: %s\n", cloudName)
	fmt.Printf("   API Key: %s\n", apiKey)
	fmt.Printf("   API Secret: %s (length: %d)\n", strings.Repeat("*", len(apiSecret)), len(apiSecret))

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, errors.New("cloudinary credentials not configured")
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	fmt.Printf(" Cloudinary initialized successfully\n")
	return &CloudinaryService{cld: cld}, nil
}

func (s *CloudinaryService) ValidateImageFile(file *multipart.FileHeader) error {
	const maxSize = 10 * 1024 * 1024
	if file.Size > maxSize {
		return fmt.Errorf("file too large (max %dMB)", maxSize/(1024*1024))
	}

	if file.Size == 0 {
		return errors.New("file is empty")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".bmp":  true,
	}

	if !allowedExts[ext] {
		return fmt.Errorf("invalid file type '%s'. Allowed: jpg, jpeg, png, gif, webp, bmp", ext)
	}

	if strings.Contains(file.Filename, "..") || strings.Contains(file.Filename, "/") {
		return errors.New("invalid filename")
	}

	return nil
}

func (s *CloudinaryService) UploadImage(ctx context.Context, file multipart.File, filename, folder string) (string, string, error) {
	timestamp := time.Now().Unix()
	safeFilename := strings.ReplaceAll(filename, " ", "_")
	ext := filepath.Ext(safeFilename)
	safeFilename = strings.ReplaceAll(safeFilename, ext, "")
	publicID := fmt.Sprintf("%s/%d_%s", folder, timestamp, safeFilename)

	fmt.Printf(" Starting upload to Cloudinary...\n")
	fmt.Printf("   Filename: %s\n", filename)
	fmt.Printf("   Folder: %s\n", folder)
	fmt.Printf("   Public ID: %s\n", publicID)

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	uploadParams := uploader.UploadParams{
		PublicID:     publicID,
		Folder:       folder,
		ResourceType: "image",
	}

	fmt.Printf(" Calling Cloudinary Upload API...\n")

	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploadParams)

	if err != nil {
		fmt.Printf(" Cloudinary Upload Error: %v\n", err)
		return "", "", fmt.Errorf("failed to upload to cloudinary: %w", err)
	}

	fmt.Printf(" Upload Result Details:\n")
	if uploadResult == nil {
		fmt.Printf("   Result is NIL\n")
		return "", "", fmt.Errorf("cloudinary returned nil result")
	}

	fmt.Printf("   Public ID: %s\n", uploadResult.PublicID)
	fmt.Printf("   Secure URL: %s\n", uploadResult.SecureURL)
	fmt.Printf("   URL: %s\n", uploadResult.URL)
	fmt.Printf("   Format: %s\n", uploadResult.Format)
	fmt.Printf("   Width: %d\n", uploadResult.Width)
	fmt.Printf("   Height: %d\n", uploadResult.Height)
	fmt.Printf("   Bytes: %d\n", uploadResult.Bytes)

	if uploadResult.SecureURL == "" {
		fmt.Printf("   SecureURL is empty, trying URL...\n")
		if uploadResult.URL != "" {
			fmt.Printf("   Using URL instead: %s\n", uploadResult.URL)
			return uploadResult.URL, uploadResult.PublicID, nil
		}

		if uploadResult.PublicID != "" {
			cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
			constructedURL := fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s",
				cloudName, uploadResult.PublicID)
			fmt.Printf("   Constructed URL: %s\n", constructedURL)
			return constructedURL, uploadResult.PublicID, nil
		}

		fmt.Printf("   No URL available in result\n")
		return "", "", fmt.Errorf("cloudinary returned empty result - no URL available")
	}

	if uploadResult.PublicID == "" {
		fmt.Printf("   PublicID is empty, using parameter: %s\n", publicID)
		return uploadResult.SecureURL, publicID, nil
	}

	fmt.Printf(" Upload successful!\n")
	fmt.Printf("   Final URL: %s\n", uploadResult.SecureURL)
	fmt.Printf("   Final Public ID: %s\n", uploadResult.PublicID)

	return uploadResult.SecureURL, uploadResult.PublicID, nil
}

func (s *CloudinaryService) DeleteImage(ctx context.Context, publicID string) error {
	if publicID == "" {
		return nil
	}

	fmt.Printf(" Deleting image from Cloudinary: %s\n", publicID)

	result, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Not Found") {
			fmt.Printf(" Image not found (already deleted): %s\n", publicID)
			return nil
		}
		fmt.Printf(" Failed to delete image: %v\n", err)
		return fmt.Errorf("failed to delete image %s: %w", publicID, err)
	}

	if result.Result != "ok" && result.Result != "not found" {
		fmt.Printf(" Delete result: %s\n", result.Result)
		return fmt.Errorf("failed to delete image: %s", result.Result)
	}

	fmt.Printf(" Image deleted successfully: %s\n", publicID)
	return nil
}

func (s *CloudinaryService) UploadMultipleImages(ctx context.Context, files []*multipart.FileHeader, folder string) ([]map[string]string, error) {
	results := []map[string]string{}

	for i, fileHeader := range files {
		fmt.Printf(" Uploading file %d/%d: %s\n", i+1, len(files), fileHeader.Filename)

		file, err := fileHeader.Open()
		if err != nil {
			fmt.Printf(" Failed to open file: %v\n", err)
			for _, result := range results {
				s.DeleteImage(ctx, result["public_id"])
			}
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		url, publicID, err := s.UploadImage(ctx, file, fileHeader.Filename, folder)
		if err != nil {
			fmt.Printf(" Upload failed: %v\n", err)
			for _, result := range results {
				s.DeleteImage(ctx, result["public_id"])
			}
			return nil, err
		}

		results = append(results, map[string]string{
			"url":       url,
			"public_id": publicID,
		})
	}

	fmt.Printf(" All %d images uploaded successfully\n", len(results))
	return results, nil
}

func (s *CloudinaryService) UploadImageWithTimeout(ctx context.Context, file multipart.File, filename, folder string, timeout time.Duration) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return s.UploadImage(ctx, file, filename, folder)
}

func (s *CloudinaryService) TestConnection(ctx context.Context) error {
	fmt.Printf(" Testing Cloudinary connection...\n")

	if s.cld == nil {
		return errors.New("cloudinary client is nil")
	}

	fmt.Printf(" Cloudinary client is initialized\n")
	return nil
}
