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

	GetByCustomerID(ctx context.Context, customerID string) ([]*domain.Order, error) //все заказы юзера
}

type PaymentGateway interface {
	AuthorizePayment(ctx context.Context, orderID string, amount int64) (transactionID string, status string, err error)
}
