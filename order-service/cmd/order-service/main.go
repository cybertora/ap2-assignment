package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"order-service/internal/app"
	"order-service/internal/repository"
	grpcserver "order-service/internal/transport/grpc"
	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	orderpb "github.com/cybertora/ap2-proto-generated/order"
	"google.golang.org/grpc"
)

func main() {
	log.Println("[INFO] starting Order Service...")

	cfg := app.LoadConfig()

	db := app.ConnectDB(cfg)
	defer db.Close()

	orderRepo := repository.NewPostgresOrderRepository(db)

	paymentClient, err := app.NewGRPCPaymentClient(cfg.PaymentGRPCAddr)
	if err != nil {
		log.Fatalf("[FATAL] failed to connect to Payment gRPC: %v", err)
	}
	defer paymentClient.Close()

	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	eventBus := repository.NewPgEventBus(dsn)
	if err := eventBus.Start(); err != nil {
		log.Fatalf("[FATAL] failed to start event bus: %v", err)
	}
	defer eventBus.Stop()

	orderRepo.SetEventBus(eventBus)

	grpcAddr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("[FATAL] failed to listen on %s: %v", grpcAddr, err)
	}

	grpcSrv := grpc.NewServer()
	orderGRPCHandler := grpcserver.NewOrderGRPCServer(eventBus)
	orderpb.RegisterOrderServiceServer(grpcSrv, orderGRPCHandler)

	go func() {
		log.Printf("[INFO] Order gRPC server listening on %s (streaming)", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("[FATAL] gRPC server failed: %v", err)
		}
	}()

	handler := transporthttp.NewOrderHandler(orderUC, paymentClient)
	router := transporthttp.NewRouter(handler)

	httpAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	go func() {
		log.Printf("[INFO] Order HTTP server listening on %s", httpAddr)
		if err := router.Run(httpAddr); err != nil {
			log.Fatalf("[FATAL] HTTP server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] shutting down Order Service...")
	grpcSrv.GracefulStop()
	log.Println("[INFO] Order Service stopped")
}
