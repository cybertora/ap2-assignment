package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/app"
	"notification-service/internal/messaging"
	"notification-service/internal/provider"
	"notification-service/internal/retry"
	"notification-service/internal/service"

	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("[INFO] starting Notification Service...")
	cfg := app.LoadConfig()

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		cancel()
		log.Fatalf("[FATAL] redis ping failed: %v", err)
	}
	cancel()
	defer rdb.Close()
	log.Printf("[INFO] connected to Redis at %s (db=%d)", cfg.RedisAddr, cfg.RedisDB)

	// --- Adapter Pattern: выбор провайдера по PROVIDER_MODE ---
	sender := provider.New(cfg)

	// --- Идемпотентность и retry ---
	idemp := service.NewRedisIdempotencyStore(rdb, cfg.IdempotencyTTLHours)
	retryPolicy := retry.New(cfg.RetryMaxAttempts, cfg.RetryBaseDelayMS)

	// --- Use-case worker (чистая архитектура — без знаний о Redis/AMQP) ---
	worker := service.NewNotificationWorker(sender, idemp, retryPolicy)

	// --- AMQP consumer + worker-pool ---
	consumer, err := messaging.NewConsumer(
		cfg.RabbitMQURL,
		cfg.PaymentEventsExchange,
		cfg.PaymentEventsRoutingKey,
		cfg.PaymentDLXExchange,
		cfg.PaymentQueue,
		cfg.PaymentDLQ,
		worker,
		cfg.WorkerPoolSize,
	)
	if err != nil {
		log.Fatalf("[FATAL] failed to initialize notification consumer: %v", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("[WARN] failed to close consumer cleanly: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("[FATAL] notification consumer stopped with error: %v", err)
	}

	log.Println("[INFO] Notification Service stopped")
}
