// streaming-test-client — тестовый клиент для демонстрации
// Server-side Streaming RPC (SubscribeToOrderUpdates).
//
// Использование:
//
//	go run streaming_client.go <order_id>
//
// Клиент подключается к Order Service gRPC и слушает обновления
// статуса заказа в реальном времени.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	orderpb "github.com/cybertora/ap2-proto-generated/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run streaming_client.go <order_id>")
		fmt.Println("Example: go run streaming_client.go 550e8400-e29b-41d4-a716-446655440000")
		os.Exit(1)
	}

	orderID := os.Args[1]
	addr := getEnv("ORDER_GRPC_ADDR", "localhost:50052")

	log.Printf("Connecting to Order gRPC at %s...", addr)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := orderpb.NewOrderServiceClient(conn)

	log.Printf("Subscribing to updates for order %s...", orderID)
	log.Println("Waiting for status changes (create/cancel an order in another terminal)...")
	log.Println("Press Ctrl+C to stop.")
	fmt.Println("─────────────────────────────────────────")

	stream, err := client.SubscribeToOrderUpdates(context.Background(), &orderpb.OrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream ended (server closed).")
			break
		}
		if err != nil {
			log.Fatalf("Error receiving update: %v", err)
		}

		fmt.Printf("📦 UPDATE: order_id=%s | status=%s | time=%s\n",
			update.GetOrderId(),
			update.GetStatus(),
			update.GetUpdatedAt().AsTime().Format("2006-01-02 15:04:05 UTC"),
		)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
