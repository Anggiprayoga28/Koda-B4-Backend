package handler

import (
	"coffee-shop/middleware"
	"coffee-shop/models"
	"coffee-shop/routes"
	"log"
	"net/http"
	"sync"

	_ "coffee-shop/docs"

	"github.com/gin-gonic/gin"
)

var (
	app  *gin.Engine
	once sync.Once
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		log.Println("Initializing database")
		models.InitDB()

		log.Println("Initializing Redis")
		models.InitRedis()

		gin.SetMode(gin.ReleaseMode)

		app = gin.New()
		app.Use(gin.Recovery())
		app.Use(middleware.CORSMiddleware())

		routes.SetupRoutes(app)

		log.Println("Server initialized")
	})

	app.ServeHTTP(w, r)
}
