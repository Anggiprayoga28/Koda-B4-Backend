package routes

import (
	"coffee-shop/controllers"
	"coffee-shop/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(router *gin.Engine) {
	authController := controllers.NewAuthController()
	userController := controllers.NewUserController()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Coffee Shop API is running",
		})
	})

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)

			protected := auth.Group("")
			protected.Use(middleware.AuthMiddleware())
			{
				protected.GET("/profile", authController.GetProfile)
				protected.PATCH("/profile", authController.UpdateProfile)
				protected.POST("/profile/photo", authController.UpdateProfilePhoto)
				protected.POST("/change-password", authController.ChangePassword)
			}
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
		{
			admin.GET("/users", userController.GetAllUsers)
			admin.GET("/users/:id", userController.GetUserByID)
			admin.POST("/users", userController.CreateUser)
			admin.PATCH("/users/:id", userController.UpdateUser)
			admin.DELETE("/users/:id", userController.DeleteUser)
		}
	}

	router.Static("/uploads", "./uploads")
}
