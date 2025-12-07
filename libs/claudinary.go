package libs

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
)

func UploadToCloudinary(localPath string) (string, error) {
	godotenv.Load()
	fmt.Printf("[Cloudinary] Starting upload for: %s\n", localPath)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return "", fmt.Errorf("cile not found: %s", localPath)
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	fmt.Printf("[Cloudinary] Environment vars - CloudName: %s, API Key: %s...\n",
		cloudName,
		apiKey[:min(5, len(apiKey))]+"...")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		fmt.Println("[Cloudinary] Separate env vars not found, trying CLOUDINARY_URL")

		cldURL := os.Getenv("CLOUDINARY_URL")
		if cldURL == "" {
			return "", fmt.Errorf("cloudinary environment variables not set")
		}

		fmt.Printf("[Cloudinary] Using CLOUDINARY_URL: %s\n",
			maskURL(cldURL))

		cld, err := cloudinary.NewFromURL(cldURL)
		if err != nil {
			return "", fmt.Errorf("cloudinary init from URL fail: %v", err)
		}

		return uploadFile(cld, localPath)
	}

	fmt.Println("[Cloudinary] Initializing with separate params...")
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return "", fmt.Errorf("cloudinary init from params fail: %v", err)
	}

	return uploadFile(cld, localPath)
}

func uploadFile(cld *cloudinary.Cloudinary, localPath string) (string, error) {
	fmt.Printf("[Cloudinary] Uploading file: %s\n", localPath)

	resp, err := cld.Upload.Upload(context.Background(), localPath, uploader.UploadParams{
		PublicID: fmt.Sprintf("profile_%d", time.Now().UnixNano()),
		Folder:   "profiles",
	})

	fmt.Printf("[Cloudinary] Removing local file: %s\n", localPath)
	os.Remove(localPath)

	if err != nil {
		fmt.Printf("[Cloudinary] Upload error: %v\n", err)
		return "", err
	}

	if resp == nil {
		fmt.Println("[Cloudinary] Response is nil!")
		return "", fmt.Errorf("cloudinary response is nil")
	}

	fmt.Printf("[Cloudinary] Upload successful!\n")
	fmt.Printf("[Cloudinary] Secure URL: %s\n", resp.SecureURL)
	fmt.Printf("[Cloudinary] Public ID: %s\n", resp.PublicID)
	fmt.Printf("[Cloudinary] URL: %s\n", resp.URL)

	if resp.SecureURL == "" {
		fmt.Println("[Cloudinary] SecureURL is empty, trying regular URL")
		if resp.URL != "" {
			return resp.URL, nil
		}
		return "", fmt.Errorf("coth SecureURL and URL are empty")
	}

	return resp.SecureURL, nil
}

func maskURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:10] + "..." + url[len(url)-10:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func DeleteFromCloudinary(publicID string) error {
	fmt.Printf("[Cloudinary] Deleting public ID: %s\n", publicID)

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		cldURL := os.Getenv("CLOUDINARY_URL")
		if cldURL == "" {
			return fmt.Errorf("cloudinary environment variables not set")
		}

		cld, err := cloudinary.NewFromURL(cldURL)
		if err != nil {
			return fmt.Errorf("cloudinary init from URL fail: %v", err)
		}

		return deleteFile(cld, publicID)
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return fmt.Errorf("cloudinary init from params fail: %v", err)
	}

	return deleteFile(cld, publicID)
}

func deleteFile(cld *cloudinary.Cloudinary, publicID string) error {
	ctx := context.Background()

	fmt.Printf("[Cloudinary] Destroying: %s\n", publicID)

	result, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})

	if err != nil {
		fmt.Printf("[Cloudinary] Delete error: %v\n", err)
		return fmt.Errorf("cailed to delete from Cloudinary: %v", err)
	}

	fmt.Printf("[Cloudinary] Delete result: %s\n", result.Result)

	if result.Result != "ok" {
		return fmt.Errorf("cloudinary deletion failed: %s", result.Result)
	}

	fmt.Println("[Cloudinary] File deleted successfully")
	return nil
}
