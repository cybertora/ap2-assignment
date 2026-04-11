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

// PaymentGRPCServer реализует paymentpb.PaymentServiceServer.
// Это Delivery-слой (gRPC). Бизнес-логика НЕ дублируется —
// всё делегируется в UseCase (Clean Architecture).
type PaymentGRPCServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUseCase
}

// NewPaymentGRPCServer создаёт новый gRPC-хэндлер для Payment Service.
func NewPaymentGRPCServer(uc *usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

// ProcessPayment обрабатывает gRPC-запрос на авторизацию платежа.
// Делегирует в PaymentUseCase.AuthorizePayment (тот же use case, что и в REST).
func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *paymentpb.PaymentRequest) (*paymentpb.PaymentResponse, error) {
	log.Printf("[gRPC] ProcessPayment called: order_id=%s amount=%d", req.GetOrderId(), req.GetAmount())

	// Валидация входных данных
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	// Делегируем бизнес-логику в UseCase (НЕ дублируем!)
	payment, err := s.uc.AuthorizePayment(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		log.Printf("[gRPC ERROR] ProcessPayment failed: %v", err)
		return nil, status.Errorf(codes.Internal, "payment processing failed: %v", err)
	}

	// Маппинг domain → proto response
	return &paymentpb.PaymentResponse{
		Id:            payment.ID,
		OrderId:       payment.OrderID,
		TransactionId: payment.TransactionID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     timestamppb.New(payment.CreatedAt),
	}, nil
}
