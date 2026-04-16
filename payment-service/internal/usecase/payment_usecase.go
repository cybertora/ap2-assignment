package usecase

import (
	"context"
	"log"
	"time"

	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUseCase struct {
	repo PaymentRepository
}

func NewPaymentUseCase(repo PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
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

func (uc *PaymentUseCase) ListPayments(ctx context.Context, status string) ([]*domain.Payment, error) {
	payments, err := uc.repo.ListByStatus(ctx, status)
	if err != nil {
		log.Printf("[ERROR] failed to list payments: %v", err)
		return nil, err
	}

	log.Printf("[INFO] listed %d payments (filter: status=%q)", len(payments), status)
	return payments, nil
}
