package utils

import (
	"coffee-shop/config"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var allowedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

func UploadFile(c *gin.Context, fileHeader *multipart.FileHeader, subDir string) (string, error) {
	if fileHeader.Size > config.AppConfig.MaxUploadSize {
		return "", errors.New("file size exceeds maximum allowed size")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedImageExtensions[ext] {
		return "", errors.New("invalid file type. Only images are allowed")
	}

	uploadPath := filepath.Join(config.AppConfig.UploadDir, subDir)
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(fileHeader.Filename, " ", "_"), "")
	if len(filename) > 255 {
		filename = fmt.Sprintf("%d%s", time.Now().Unix(), ext)
	}

	filePath := filepath.Join(uploadPath, filename)

	if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
		return "", err
	}

	return filepath.Join(subDir, filename), nil
}

func DeleteFile(filePath string) error {
	fullPath := filepath.Join(config.AppConfig.UploadDir, filePath)
	if _, err := os.Stat(fullPath); err == nil {
		return os.Remove(fullPath)
	}
	return nil
}
