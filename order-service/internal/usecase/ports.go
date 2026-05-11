package usecase

import (
	"context"

	"order-service/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
}

type PaymentGateway interface {
	AuthorizePayment(ctx context.Context, orderID string, amount int64) (transactionID string, status string, err error)
}

// OrderCache — порт кэша заказов.
// Чистая архитектура: use-case НЕ знает про Redis, только про этот интерфейс.
type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error) // (nil, nil) => cache miss
	Set(ctx context.Context, order *domain.Order) error
	Delete(ctx context.Context, id string) error
}
