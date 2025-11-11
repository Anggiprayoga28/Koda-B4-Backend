package main

import (
	"coffee-shop/middleware"
	"coffee-shop/models"
	"coffee-shop/routes"
	"log"
	"os"

	_ "coffee-shop/docs"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// @title Coffee Shop
// @version 1.0
// @description REST API for Coffee Shop Management System
// @host localhost:8082
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	godotenv.Load()

	models.InitDB()
	defer models.CloseDB()

	models.InitRedis()
	defer models.CloseRedis()

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	routes.SetupRoutes(r)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Server running on port %s", port)
	log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", port)
	r.Run(":" + port)
}
