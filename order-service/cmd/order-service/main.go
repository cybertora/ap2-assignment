package main

import (
	"fmt"
	"log"

	"order-service/internal/app"
	"order-service/internal/repository"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

func main() {
	log.Println("[INFO] starting Order Service...")

	cfg := app.LoadConfig()

	db := app.ConnectDB(cfg)
	defer db.Close()

	orderRepo := repository.NewPostgresOrderRepository(db)

	paymentClient := app.NewPaymentClient(cfg.PaymentServiceURL)

	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)

	handler := transporthttp.NewOrderHandler(orderUC)

	router := transporthttp.NewRouter(handler)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("[INFO] Order Service listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("[FATAL] failed to start server: %v", err)
	}
}
