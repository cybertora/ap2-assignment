package usecase

import (
	"context"

	"order-service/internal/domain"
)

// OrderRepository — порт для работы с хранилищем заказов.
// НЕ ИЗМЕНЁН из Assignment 1 (Clean Architecture).
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error

	GetByID(ctx context.Context, id string) (*domain.Order, error)

	UpdateStatus(ctx context.Context, id string, status string) error

	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
}

// PaymentGateway — порт для вызова Payment Service.
// НЕ ИЗМЕНЁН из Assignment 1 (Clean Architecture).
// В Assignment 1 реализация была REST, теперь — gRPC.
// Use Case не знает о деталях транспорта (Dependency Inversion).
type PaymentGateway interface {
	AuthorizePayment(ctx context.Context, orderID string, amount int64) (transactionID string, status string, err error)
}
