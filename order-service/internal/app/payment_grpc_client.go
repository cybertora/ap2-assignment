package app

import (
	"context"
	"fmt"
	"log"
	"time"

	paymentpb "github.com/cybertora/ap2-proto-generated/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCPaymentClient struct {
	conn   *grpc.ClientConn
	client paymentpb.PaymentServiceClient
}

func NewGRPCPaymentClient(addr string) (*GRPCPaymentClient, error) {
	log.Printf("[INFO] connecting to Payment gRPC at %s", addr)

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to payment gRPC: %w", err)
	}

	client := paymentpb.NewPaymentServiceClient(conn)
	log.Printf("[INFO] connected to Payment gRPC at %s", addr)

	return &GRPCPaymentClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *GRPCPaymentClient) AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	log.Printf("[INFO] calling Payment gRPC: ProcessPayment(order_id=%s, amount=%d)", orderID, amount)

	resp, err := c.client.ProcessPayment(ctx, &paymentpb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return "", "", fmt.Errorf("payment gRPC call failed: %w", err)
	}

	log.Printf("[INFO] Payment gRPC response: transaction_id=%s status=%s", resp.GetTransactionId(), resp.GetStatus())

	return resp.GetTransactionId(), resp.GetStatus(), nil
}

func (c *GRPCPaymentClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
