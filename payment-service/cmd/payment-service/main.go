package main

import (
	"fmt"
	"log"

	"payment-service/internal/app"
	"payment-service/internal/repository"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

func main() {
	log.Println("[INFO] starting Payment Service...")

	cfg := app.LoadConfig()

	db := app.ConnectDB(cfg)
	defer db.Close()

	paymentRepo := repository.NewPostgresPaymentRepository(db)

	paymentUC := usecase.NewPaymentUseCase(paymentRepo)

	handler := transporthttp.NewPaymentHandler(paymentUC)

	router := transporthttp.NewRouter(handler)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("[INFO] Payment Service listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("[FATAL] failed to start server: %v", err)
	}
}
