package service

import (
	"context"
	"fmt"
	"log"

	"notification-service/internal/domain"
	"notification-service/internal/retry"
)

// NotificationWorker — use-case уровень: содержит чистую бизнес-логику
// "получили payment.processed -> идемпотентно отправили email с retry".
// Зависит только от портов (EmailSender, IdempotencyStore) — не знает ни про Redis, ни про AMQP.
type NotificationWorker struct {
	sender domain.EmailSender
	idemp  IdempotencyStore
	retry  retry.Policy
}

// IdempotencyStore — порт. Удобно для тестов (можно подменить in-memory мок).
type IdempotencyStore interface {
	Acquire(ctx context.Context, paymentID string) (bool, error)
	Release(ctx context.Context, paymentID string)
}

func NewNotificationWorker(sender domain.EmailSender, idemp IdempotencyStore, rp retry.Policy) *NotificationWorker {
	return &NotificationWorker{sender: sender, idemp: idemp, retry: rp}
}

// Process — единая точка обработки одного события.
func (w *NotificationWorker) Process(ctx context.Context, n domain.Notification) error {
	// 1) Идемпотентность.
	acquired, err := w.idemp.Acquire(ctx, n.PaymentID)
	if err != nil {
		return fmt.Errorf("idempotency acquire: %w", err)
	}
	if !acquired {
		return nil // дубликат — корректно пропускаем
	}

	// 2) Отправка с retry.
	err = w.retry.Do(ctx, "send:"+n.PaymentID, func(ctx context.Context) error {
		return w.sender.Send(ctx, n)
	})
	if err != nil {
		// Все попытки провалились — отпускаем замок, чтобы DLQ-логика/перепосылка позже сработала.
		log.Printf("[WORKER] giving up payment_id=%s after retries: %v", n.PaymentID, err)
		w.idemp.Release(ctx, n.PaymentID)
		return err
	}

	log.Printf("[WORKER] delivered payment_id=%s via %s", n.PaymentID, w.sender.Name())
	return nil
}
