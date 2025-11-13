package api

import (
	"coffee-shop/middleware"
	"coffee-shop/models"
	"coffee-shop/routes"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	router *gin.Engine
	once   sync.Once
)

func initApp() {
	once.Do(func() {
		gin.SetMode(gin.ReleaseMode)

		models.InitDB()
		models.InitRedis()

		router = gin.New()
		router.Use(gin.Recovery())
		router.Use(middleware.CORSMiddleware())

		routes.SetupRoutes(router)
	})
}

func Handler(w http.ResponseWriter, r *http.Request) {
	initApp()
	router.ServeHTTP(w, r)
}
