package usecase

import (
	"context"

	"payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	ListByStatus(ctx context.Context, status string) ([]*domain.Payment, error)
}
