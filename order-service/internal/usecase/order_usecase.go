package usecase

import (
	"context"
	"log"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
)

type OrderUseCase struct {
	repo           OrderRepository
	paymentGateway PaymentGateway
	cache          OrderCache // НОВОЕ: порт кэша
}

func NewOrderUseCase(repo OrderRepository, pg PaymentGateway, cache OrderCache) *OrderUseCase {
	return &OrderUseCase{
		repo:           repo,
		paymentGateway: pg,
		cache:          cache,
	}
}

// ---------------- helpers ----------------

// invalidateCache — единая точка инвалидации; вызывается ПОСЛЕ любой успешной мутации статуса.
// Использует Redis DEL — атомарная команда на стороне сервера, что гарантирует
// что после UpdateStatus в БД ни один читатель не получит протухшую запись.
func (uc *OrderUseCase) invalidateCache(ctx context.Context, orderID string) {
	if uc.cache == nil {
		return
	}
	if err := uc.cache.Delete(ctx, orderID); err != nil {
		// Не валим бизнес-операцию из-за кэша, но логируем.
		log.Printf("[WARN] cache invalidation failed for order %s: %v", orderID, err)
	}
}

// ---------------- use cases ----------------

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
		if upErr := uc.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed); upErr != nil {
			log.Printf("[ERROR] failed to update order status to Failed: %v", upErr)
		}
		order.Status = domain.StatusFailed
		uc.invalidateCache(ctx, order.ID) // <-- atomic invalidation
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
		uc.invalidateCache(ctx, order.ID) // <-- atomic invalidation
		return order, domain.ErrPaymentDeclined
	}

	// Заказ только что создан и успешно оплачен — прогреем кэш свежим значением.
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, order)
	}
	return order, nil
}

// GetOrder — классический Cache-Aside Read.
//  1. cache.Get
//  2. cache miss -> repo.GetByID
//  3. cache.Set (заполняем кэш на TTL)
func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	if uc.cache != nil {
		if cached, _ := uc.cache.Get(ctx, id); cached != nil {
			return cached, nil
		}
	}

	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrOrderNotFound
	}

	// Прогрев кэша.
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, order)
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

	uc.invalidateCache(ctx, id) // <-- atomic invalidation сразу после смены статуса
	return order, nil
}

// MarkPaid — вызывается ИЗ ВНЕ (например, по событию payment.processed),
// чтобы инвалидировать кэш после оплаты. Оставлен как явная точка интеграции.
func (uc *OrderUseCase) MarkPaid(ctx context.Context, id string) error {
	if err := uc.repo.UpdateStatus(ctx, id, domain.StatusPaid); err != nil {
		return err
	}
	uc.invalidateCache(ctx, id)
	return nil
}
