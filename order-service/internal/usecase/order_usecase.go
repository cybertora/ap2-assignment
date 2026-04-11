package usecase

import (
	"context"
	"log"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
)

// OrderUseCase — бизнес-логика заказов.
// ПОЛНОСТЬЮ ИДЕНТИЧЕН Assignment 1 (Clean Architecture preserved).
// Ни одна строка бизнес-логики не была изменена при миграции на gRPC.
type OrderUseCase struct {
	repo           OrderRepository
	paymentGateway PaymentGateway
}

func NewOrderUseCase(repo OrderRepository, pg PaymentGateway) *OrderUseCase {
	return &OrderUseCase{
		repo:           repo,
		paymentGateway: pg,
	}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey *string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	if idempotencyKey != nil && *idempotencyKey != "" {
		existing, err := uc.repo.GetByIdempotencyKey(ctx, *idempotencyKey)
		if err != nil {
			log.Printf("[ERROR] idempotency check failed: %v", err)
			return nil, err
		}
		if existing != nil {
			log.Printf("[INFO] idempotent request: returning existing order %s", existing.ID)
			return existing, nil
		}
	}

	order := &domain.Order{
		ID:             uuid.New().String(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         domain.StatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		log.Printf("[ERROR] failed to create order: %v", err)
		return nil, err
	}
	log.Printf("[INFO] order created: id=%s status=%s amount=%d", order.ID, order.Status, order.Amount)

	txnID, paymentStatus, err := uc.paymentGateway.AuthorizePayment(ctx, order.ID, order.Amount)
	if err != nil {
		log.Printf("[ERROR] payment call failed for order %s: %v", order.ID, err)

		if updateErr := uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed); updateErr != nil {
			log.Printf("[ERROR] failed to update order status to Failed: %v", updateErr)
		}
		order.Status = domain.StatusFailed

		return order, domain.ErrPaymentServiceUnavailable
	}

	if paymentStatus == "Authorized" {
		if err := uc.repo.UpdateStatus(ctx, order.ID, domain.StatusPaid); err != nil {
			log.Printf("[ERROR] failed to update order status to Paid: %v", err)
			return nil, err
		}
		order.Status = domain.StatusPaid
		log.Printf("[INFO] order %s paid: transaction_id=%s", order.ID, txnID)
	} else {
		if err := uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed); err != nil {
			log.Printf("[ERROR] failed to update order status to Failed: %v", err)
			return nil, err
		}
		order.Status = domain.StatusFailed
		log.Printf("[INFO] order %s payment declined: transaction_id=%s", order.ID, txnID)
		return order, domain.ErrPaymentDeclined
	}

	return order, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrOrderNotFound
	}

	if !order.CanBeCancelled() {
		return nil, domain.ErrCancelNotAllowed
	}

	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusCancelled); err != nil {
		return nil, err
	}

	order.Status = domain.StatusCancelled
	log.Printf("[INFO] order %s cancelled", order.ID)
	return order, nil
}
