package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"payment-service/internal/app"
	"payment-service/internal/repository"
	grpcserver "payment-service/internal/transport/grpc"
	transporthttp "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	paymentpb "github.com/cybertora/ap2-proto-generated/payment"
	"google.golang.org/grpc"
)

func main() {
	log.Println("[INFO] starting Payment Service...")

	cfg := app.LoadConfig()

	db := app.ConnectDB(cfg)
	defer db.Close()

	paymentRepo := repository.NewPostgresPaymentRepository(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo)

	// ─── gRPC Server (ProcessPayment) ───────────────────────────────
	grpcAddr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("[FATAL] failed to listen on %s: %v", grpcAddr, err)
	}

	// Бонус (+10%): gRPC Interceptor — логирует method name + duration
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcserver.LoggingInterceptor),
	)

	paymentGRPCHandler := grpcserver.NewPaymentGRPCServer(paymentUC)
	paymentpb.RegisterPaymentServiceServer(grpcSrv, paymentGRPCHandler)

	go func() {
		log.Printf("[INFO] Payment gRPC server listening on %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("[FATAL] gRPC server failed: %v", err)
		}
	}()

	// ─── REST Server (GET /payments/:order_id — оставляем из Assignment 1) ──
	handler := transporthttp.NewPaymentHandler(paymentUC)
	router := transporthttp.NewRouter(handler)

	httpAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	go func() {
		log.Printf("[INFO] Payment HTTP server listening on %s", httpAddr)
		if err := router.Run(httpAddr); err != nil {
			log.Fatalf("[FATAL] HTTP server failed: %v", err)
		}
	}()

	// ─── Graceful shutdown ──────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] shutting down Payment Service...")
	grpcSrv.GracefulStop()
	log.Println("[INFO] Payment Service stopped")
}
