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
	categoryCtrl := &controllers.CategoryController{}
	productDetailCtrl := &controllers.ProductDetailController{}
	transactionCtrl := &controllers.TransactionController{}
	historyCtrl := &controllers.HistoryController{}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	router.POST("/auth/register", authCtrl.Register)
	router.POST("/auth/login", authCtrl.Login)
	router.POST("/auth/change-password", authCtrl.ChangePassword)

	router.GET("/categories", categoryCtrl.GetCategories)
	router.GET("/categories/:id", categoryCtrl.GetCategoryByID)

	router.GET("/products", productCtrl.GetAllProducts)
	router.GET("/products/filter", productCtrl.FilterProducts)
	router.GET("/products/favorite", productCtrl.GetFavoriteProducts)
	router.GET("/products/:id", productCtrl.GetProductByID)
	router.GET("/products/:id/detail", productDetailCtrl.GetProductDetail)

	authProtected := router.Group("/auth")
	authProtected.Use(middleware.AuthMiddleware())
	{
		authProtected.GET("/profile", authCtrl.GetProfile)
		authProtected.PATCH("/profile", authCtrl.UpdateProfile)
	}

	cartRoutes := router.Group("/cart")
	cartRoutes.Use(middleware.AuthMiddleware())
	{
		cartRoutes.POST("", productDetailCtrl.AddToCart)
		cartRoutes.GET("", productDetailCtrl.GetCart)
	}

	orderRoutes := router.Group("/orders")
	orderRoutes.Use(middleware.AuthMiddleware())
	{
		orderRoutes.POST("", orderCtrl.CreateOrder)
	}

	transactionRoutes := router.Group("/transactions")
	transactionRoutes.Use(middleware.AuthMiddleware())
	{
		transactionRoutes.POST("/checkout", transactionCtrl.Checkout)
	}

	historyRoutes := router.Group("/history")
	historyRoutes.Use(middleware.AuthMiddleware())
	{
		historyRoutes.GET("", historyCtrl.GetHistory)
		historyRoutes.GET("/:id", historyCtrl.GetOrderDetail)
	}

	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		admin.GET("/dashboard", orderCtrl.GetDashboard)

		admin.GET("/profiles", userCtrl.GetAllUsers)
		admin.GET("/profiles/:id", userCtrl.GetUserByID)

		admin.GET("/users", userCtrl.GetAllUsers)
		admin.GET("/users/:id", userCtrl.GetUserByID)
		admin.POST("/users", userCtrl.CreateUser)
		admin.PATCH("/users/:id", userCtrl.UpdateUser)
		admin.DELETE("/users/:id", userCtrl.DeleteUser)

		admin.POST("/categories", categoryCtrl.CreateCategory)
		admin.PATCH("/categories/:id", categoryCtrl.UpdateCategory)
		admin.DELETE("/categories/:id", categoryCtrl.DeleteCategory)

		admin.POST("/products", productCtrl.CreateProduct)
		admin.PATCH("/products/:id", productCtrl.UpdateProduct)
		admin.DELETE("/products/:id", productCtrl.DeleteProduct)

		admin.GET("/orders", orderCtrl.GetAllOrders)
		admin.GET("/orders/:id", orderCtrl.GetOrderByID)
		admin.PATCH("/orders/:id/status", orderCtrl.UpdateOrderStatus)
	}

	router.Static("/uploads", "./uploads")
}
