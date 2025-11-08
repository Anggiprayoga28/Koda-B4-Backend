package main

import (
	"coffee-shop/config"
	_ "coffee-shop/docs"
	"coffee-shop/middleware"
	"coffee-shop/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	config.LoadConfig()

	if config.AppConfig.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	config.ConnectDB()
	defer config.CloseDB()

	if err := os.MkdirAll(config.AppConfig.UploadDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())
	routes.SetupRoutes(router)

	port := ":" + config.AppConfig.Port
	log.Printf("Server starting on port %s", port)
	log.Printf("Environment: %s", config.AppConfig.AppEnv)
	log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", config.AppConfig.Port)

	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
