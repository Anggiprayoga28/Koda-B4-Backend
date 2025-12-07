package libs

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func SaveUploadedFile(c *gin.Context, header *multipart.FileHeader, folder string) (string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp" {
		return "", fmt.Errorf("format image tidak didukung. Hanya .png, .jpg, .jpeg, .gif, .webp")
	}

	if header.Size > (5 * 1024 * 1024) {
		return "", fmt.Errorf("file terlalu besar (max 5MB)")
	}

	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return "", fmt.Errorf("gagal membuat folder: %v", err)
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	path := filepath.Join(folder, filename)

	if err := c.SaveUploadedFile(header, path); err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %v", err)
	}

	return path, nil
}
