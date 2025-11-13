package models

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	redisURL := os.Getenv("REDIS_URL")

	var opt *redis.Options
	if redisURL != "" {
		parsedOpt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Println("Failed to parse Redis URL:", err)
			log.Println("Running without cache")
			return
		}
		opt = parsedOpt
	} else {
		opt = &redis.Options{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		}
	}

	RedisClient = redis.NewClient(opt)

	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Println("Redis connection failed:", err)
		log.Println("Running without cache")
		RedisClient = nil
		return
	}

	log.Println("Redis connected")
}

func CloseRedis() {
	if RedisClient != nil {
		RedisClient.Close()
	}
}
