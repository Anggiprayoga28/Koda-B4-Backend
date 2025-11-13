package api

import (
	"coffee-shop/middleware"
	"coffee-shop/models"
	"coffee-shop/routes"
	"log"
	"net/http"

	_ "coffee-shop/docs"

	"github.com/gin-gonic/gin"
)

var (
	app *gin.Engine
)

func init() {
	log.Println("Initializing database")
	models.InitDB()

	log.Println("Initializing Redis")
	models.InitRedis()

	gin.SetMode(gin.ReleaseMode)

	app = gin.New()
	app.Use(gin.Recovery())
	app.Use(middleware.CORSMiddleware())

	routes.SetupRoutes(app)

	log.Println("Server initialized for Vercel")
}

func Handler(w http.ResponseWriter, r *http.Request) {
	app.ServeHTTP(w, r)
}
