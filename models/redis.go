package models

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Println("Redis connection failed:", err)
		log.Println("Running without cache")
		return
	}

	log.Println("Redis connected")
}

func CloseRedis() {
	if RedisClient != nil {
		RedisClient.Close()
	}
}
