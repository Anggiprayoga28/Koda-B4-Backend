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

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, errors.New("cloudinary credentials not configured")
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &CloudinaryService{cld: cld}, nil
}

func (s *CloudinaryService) ValidateImageFile(file *multipart.FileHeader) error {
	if file.Size > 10*1024*1024 {
		return errors.New("file too large (max 10MB)")
	}

	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		return errors.New("invalid file type. Only jpg, jpeg, png, gif, webp allowed")
	}

	return nil
}

func (s *CloudinaryService) UploadImage(ctx context.Context, file multipart.File, filename, folder string) (string, string, error) {
	timestamp := time.Now().Unix()
	publicID := fmt.Sprintf("%s/%d_%s", folder, timestamp, strings.ReplaceAll(filename, " ", "_"))
	publicID = strings.TrimSuffix(publicID, filepath.Ext(publicID))

	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       publicID,
		Folder:         folder,
		ResourceType:   "image",
		Transformation: "q_auto,f_auto",
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to upload to cloudinary: %w", err)
	}

	return uploadResult.SecureURL, uploadResult.PublicID, nil
}

func (s *CloudinaryService) DeleteImage(ctx context.Context, publicID string) error {
	if publicID == "" {
		return nil
	}

	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})

	if err != nil {
		return fmt.Errorf("failed to delete from cloudinary: %w", err)
	}

	return nil
}

func (s *CloudinaryService) UploadMultipleImages(ctx context.Context, files []*multipart.FileHeader, folder string) ([]map[string]string, error) {
	results := []map[string]string{}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		url, publicID, err := s.UploadImage(ctx, file, fileHeader.Filename, folder)
		if err != nil {
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

	return results, nil
}
