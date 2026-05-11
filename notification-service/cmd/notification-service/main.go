package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"notification-service/internal/app"
	"notification-service/internal/messaging"
	"notification-service/internal/service"
)

func main() {
	log.Println("[INFO] starting Notification Service...")

	cfg := app.LoadConfig()
	store := service.NewProcessedPaymentsStore()

	consumer, err := messaging.NewConsumer(
		cfg.RabbitMQURL,
		cfg.PaymentEventsExchange,
		cfg.PaymentEventsRoutingKey,
		cfg.PaymentDLXExchange,
		cfg.PaymentQueue,
		cfg.PaymentDLQ,
		store,
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
