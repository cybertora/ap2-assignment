package grpc

import (
	"context"
	"log"

	"payment-service/internal/usecase"

	paymentpb "github.com/cybertora/ap2-proto-generated/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentGRPCServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUseCase
}

func NewPaymentGRPCServer(uc *usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *paymentpb.PaymentRequest) (*paymentpb.PaymentResponse, error) {
	log.Printf("[gRPC] ProcessPayment called: order_id=%s amount=%d", req.GetOrderId(), req.GetAmount())

	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	payment, err := s.uc.AuthorizePayment(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		log.Printf("[gRPC ERROR] ProcessPayment failed: %v", err)
		return nil, status.Errorf(codes.Internal, "payment processing failed: %v", err)
	}

	return &paymentpb.PaymentResponse{
		Id:            payment.ID,
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     timestamppb.New(payment.CreatedAt),
	}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *paymentpb.ListPaymentsRequest) (*paymentpb.ListPaymentsResponse, error) {
	log.Printf("[gRPC] ListPayments called: status=%q", req.GetStatus())

	payments, err := s.uc.ListPayments(ctx, req.GetStatus())
	if err != nil {
		log.Printf("[gRPC ERROR] ListPayments failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to list payments: %v", err)
	}

	var pbPayments []*paymentpb.PaymentResponse
	for _, p := range payments {
		pbPayments = append(pbPayments, &paymentpb.PaymentResponse{
			Id:            p.ID,
			OrderId:       p.OrderID,
			TransactionId: p.TransactionID,
			Amount:        p.Amount,
			Status:        p.Status,
			CreatedAt:     timestamppb.New(p.CreatedAt),
		})
	}

	log.Printf("[gRPC] ListPayments returning %d payments", len(pbPayments))

	return &paymentpb.ListPaymentsResponse{
		Payments: pbPayments,
	}, nil
}
