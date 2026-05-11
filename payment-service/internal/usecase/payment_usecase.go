package usecase

import (
	"context"
	"log"
	"time"

	"payment-service/internal/domain"
	"payment-service/internal/messaging"

	"github.com/google/uuid"
)

type PaymentUseCase struct {
	repo      PaymentRepository
	publisher PaymentEventPublisher
}

func NewPaymentUseCase(repo PaymentRepository, publisher PaymentEventPublisher) *PaymentUseCase {
	return &PaymentUseCase{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *PaymentUseCase) AuthorizePayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	status := domain.StatusAuthorized
	if amount > domain.MaxPaymentAmount {
		status = domain.StatusDeclined
		log.Printf("[INFO] payment declined for order %s: amount %d exceeds limit %d", orderID, amount, domain.MaxPaymentAmount)
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: uuid.New().String(),
		Amount:        amount,
		Status:        status,
		CreatedAt:     time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, payment); err != nil {
		log.Printf("[ERROR] failed to create payment: %v", err)
		return nil, err
	}

	log.Printf("[INFO] payment created: id=%s order_id=%s transaction_id=%s status=%s amount=%d",
		payment.ID, payment.OrderID, payment.TransactionID, payment.Status, payment.Amount)

	if payment.IsAuthorized() && uc.publisher != nil {
		event := messaging.PaymentProcessedEvent{
			PaymentID:     payment.ID,
			OrderID:       payment.OrderID,
			TransactionID: payment.TransactionID,
			Amount:        payment.Amount,
			Status:        payment.Status,
			ProcessedAt:   payment.CreatedAt,
		}

		if err := uc.publisher.PublishPaymentProcessed(ctx, event); err != nil {
			log.Printf("[ERROR] failed to publish PaymentProcessed event for payment_id=%s: %v", payment.ID, err)
		}
	}

	return payment, nil
}

func (uc *PaymentUseCase) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	payment, err := uc.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, domain.ErrPaymentNotFound
	}
	return payment, nil
}

// Добавь string в аргументы
func (uc *PaymentUseCase) ListPayments(ctx context.Context, customerID string) ([]*domain.Payment, error) {
	// Если твой репозиторий поддерживает фильтрацию, передай ID туда
	return uc.repo.List(ctx, customerID)
}
