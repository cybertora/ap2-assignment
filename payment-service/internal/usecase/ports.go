package usecase

import (
	"context"

	"payment-service/internal/domain"
	"payment-service/internal/messaging"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	List(ctx context.Context, customerID string) ([]*domain.Payment, error)
}

type PaymentEventPublisher interface {
	PublishPaymentProcessed(ctx context.Context, event messaging.PaymentProcessedEvent) error
}
