package main

import (
	"coffee-shop/docs"
	"coffee-shop/middleware"
	"coffee-shop/models"
	"coffee-shop/routes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	cloudName := "dhufmfcps"
	apiKey := "743921273345177"
	apiSecret := "5VYVYcSsDx2Gk1RkKrnJbNQxp4w"

	fmt.Println("=== Verifikasi Cloudinary Credentials ===")
	fmt.Printf("Cloud Name: %s\n", cloudName)
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("API Secret: %s\n", apiSecret)

	// Cloud name harus lowercase dan tanpa spasi
	cloudName = strings.ToLower(strings.TrimSpace(cloudName))
	fmt.Printf("\nCloud Name (lowercase): %s\n", cloudName)

	// Format yang benar untuk Cloudinary URL
	cloudinaryURL := fmt.Sprintf("cloudinary://%s:%s@%s", apiKey, apiSecret, cloudName)
	fmt.Printf("\nCloudinary URL Format:\n%s\n", cloudinaryURL)

	// Cek panjang credential
	fmt.Printf("\nPanjang API Key: %d (harus 16 digit)\n", len(apiKey))
	fmt.Printf("Panjang API Secret: %d\n", len(apiSecret))
}

// @title Coffee Shop
// @version 1.0
// @description Coffee Shop Management System API Documentation
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	if os.Getenv("VERCEL") == "" {
		_ = godotenv.Load()
	}

	models.InitDB()
	defer models.CloseDB()

	models.InitRedis()
	defer models.CloseRedis()

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())

	docs.SwaggerInfo.Title = "Coffee Shop API"
	docs.SwaggerInfo.Description = "Coffee Shop Management System API"
	docs.SwaggerInfo.Version = "1.0"

	swaggerHost := os.Getenv("SWAGGER_HOST")
	if swaggerHost == "" {
		swaggerHost = "localhost:8083"
	}
	docs.SwaggerInfo.Host = swaggerHost
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	routes.SetupRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Swagger documentation available at: http://localhost:%s/swagger/index.html", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
