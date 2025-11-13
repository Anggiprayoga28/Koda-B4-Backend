package models

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var DB *pgxpool.Pool

func InitDB() {
	if os.Getenv("VERCEL") == "" {
		godotenv.Load()
	}

	var dsn string

	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		dsn = databaseURL
		log.Println("Using DATABASE_URL for connection")
	} else {
		sslMode := getEnv("DB_SSLMODE", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5454"),
			getEnv("DB_USER", "anggi"),
			getEnv("DB_PASSWORD", ""),
			getEnv("DB_NAME", "coffee_shop"),
			sslMode)
		log.Println("Using individual env vars for connection")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("Failed to parse DB config:", err)
	}

	if os.Getenv("VERCEL") != "" {
		config.MaxConns = 5
		config.MinConns = 0
		config.MaxConnLifetime = time.Minute * 5
		config.MaxConnIdleTime = time.Minute * 1
		config.HealthCheckPeriod = time.Minute * 1
	} else {
		config.MaxConns = 25
		config.MinConns = 5
	}

	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = DB.Ping(ctx); err != nil {
		log.Fatal("DB ping failed:", err)
	}

	log.Println("Database connected successfully")
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
