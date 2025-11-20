package main

import (
	"coffee-shop/docs"
	"coffee-shop/middleware"
	"coffee-shop/models"
	"coffee-shop/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// @title Coffee Shop
// @version 1.0
// @description Coffee Shop Management System API Documentation
// @termsOfService http://swagger.io/terms/
// @host harlan-holden-coffee-backend.vercel.app
// @BasePath /
// @schemes https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	if os.Getenv("VERCEL") == "" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
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
	docs.SwaggerInfo.Host = "harlan-holden-coffee-backend.vercel.app"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"https"}

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
