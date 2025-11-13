package models

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	var opt *redis.Options

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		log.Println("Using REDIS_URL for connection")
		parsedOpt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Println("Failed to parse REDIS_URL:", err)
			log.Println("Running without cache")
			return
		}
		opt = parsedOpt
	} else {
		log.Println("Using individual env vars for Redis connection")
		opt = &redis.Options{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		}
	}

	opt.DisableIndentity = true

	RedisClient = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Println("Redis connection failed:", err)
		log.Println("Running without cache")
		RedisClient = nil
		return
	}

	log.Println("Redis connected successfully")
}

func CloseRedis() {
	if RedisClient != nil {
		RedisClient.Close()
	}
}
