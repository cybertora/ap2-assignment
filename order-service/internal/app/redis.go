package app

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient — единая точка подключения к Redis для Order Service.
// Используется и кэшем заказов, и rate-limiter-ом.
func NewRedisClient(cfg *Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("[FATAL] redis ping failed (%s): %v", cfg.RedisAddr, err)
	}
	log.Printf("[INFO] connected to Redis at %s (db=%d)", cfg.RedisAddr, cfg.RedisDB)
	return client
}
