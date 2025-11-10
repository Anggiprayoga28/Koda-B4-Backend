package routes

import (
	"coffee-shop/controllers"
	"coffee-shop/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(router *gin.Engine) {
	authCtrl := &controllers.AuthController{}
	userCtrl := &controllers.UserController{}
	productCtrl := &controllers.ProductController{}
	orderCtrl := &controllers.OrderController{}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	router.POST("/auth/register", authCtrl.Register)
	router.POST("/auth/login", authCtrl.Login)
	router.GET("/categories", productCtrl.GetAllCategories)
	router.GET("/products", productCtrl.GetAllProducts)
	router.GET("/products/:id", productCtrl.GetProductByID)

	auth := router.Group("/")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.GET("/auth/profile", authCtrl.GetProfile)
		auth.PATCH("/auth/profile", authCtrl.UpdateProfile)
		auth.POST("/auth/profile/photo", authCtrl.UpdateProfilePhoto)
		auth.POST("/auth/change-password", authCtrl.ChangePassword)
		auth.POST("/orders", orderCtrl.CreateOrder)
	}

	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		admin.GET("/dashboard", orderCtrl.GetDashboard)

		admin.GET("/users", userCtrl.GetAllUsers)
		admin.GET("/users/:id", userCtrl.GetUserByID)
		admin.POST("/users", userCtrl.CreateUser)
		admin.PATCH("/users/:id", userCtrl.UpdateUser)
		admin.DELETE("/users/:id", userCtrl.DeleteUser)

		admin.POST("/products", productCtrl.CreateProduct)
		admin.PATCH("/products/:id", productCtrl.UpdateProduct)
		admin.DELETE("/products/:id", productCtrl.DeleteProduct)

		admin.GET("/orders", orderCtrl.GetAllOrders)
		admin.GET("/orders/:id", orderCtrl.GetOrderByID)
		admin.PATCH("/orders/:id/status", orderCtrl.UpdateOrderStatus)
	}

	router.Static("/uploads", "./uploads")
}
