package models

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

var DB *pgxpool.Pool

func InitDB() {
	if os.Getenv("VERCEL") == "" {
		_ = godotenv.Load()
	}

	dsn := buildDSN()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("Failed to parse DB config:", err)
	}

	if os.Getenv("VERCEL") != "" {
		config.MaxConns = 5
		config.MinConns = 0
		config.MaxConnLifetime = 5 * time.Minute
		config.MaxConnIdleTime = 1 * time.Minute
		config.HealthCheckPeriod = 1 * time.Minute
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

	if err := runMigrations(dsn); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
}

func buildDSN() string {
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		log.Println("Using DATABASE_URL for connection")
		return databaseURL
	}

	sslMode := getEnv("DB_SSLMODE", "disable")
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5454")
	user := getEnv("DB_USER", "anggi")
	password := getEnv("DB_PASSWORD", "")
	name := getEnv("DB_NAME", "coffee_shop")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, name, sslMode,
	)

	log.Println("Using individual env vars for connection")
	return dsn
}

func runMigrations(dsn string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open DB for migrations: %w", err)
	}
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationPath, err := filepath.Abs("database/migration")
	if err != nil {
		return fmt.Errorf("failed to resolve migration path: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Database migrations applied (or already up to date)")
	return nil
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
