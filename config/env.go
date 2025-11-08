package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv        string
	Port          string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	JWTSecret     string
	JWTExpiry     string
	UploadDir     string
	MaxUploadSize int64
}

var AppConfig *Config

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	maxUploadSize, _ := strconv.ParseInt(os.Getenv("MAX_UPLOAD_SIZE"), 10, 64)
	if maxUploadSize == 0 {
		maxUploadSize = 5242880
	}

	AppConfig = &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		Port:          getEnv("APP_PORT", getEnv("PORT", "8082")),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5454"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "coffee_shop"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		JWTSecret:     getEnv("JWT_SECRET", "secret"),
		JWTExpiry:     getEnv("JWT_EXPIRY", "24h"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadSize: maxUploadSize,
	}

	log.Println("Configuration loaded successfully")
	log.Printf("Environment: %s", AppConfig.AppEnv)
	log.Printf("Server will run on port: %s", AppConfig.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
