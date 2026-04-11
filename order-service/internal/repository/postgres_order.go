package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"order-service/internal/domain"
)

type PostgresOrderRepository struct {
	db       *sql.DB
	eventBus *PgEventBus
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) SetEventBus(eb *PgEventBus) {
	r.eventBus = eb
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, item_name, amount, status, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		order.ID,
		order.CustomerID,
		order.ItemName,
		order.Amount,
		order.Status,
		order.IdempotencyKey,
		order.CreatedAt,
	)
	if err != nil {
		log.Printf("[ERROR] PostgresOrderRepository.Create: %v", err)
		return err
	}
	return nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at
		FROM orders
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var order domain.Order
	err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.IdempotencyKey,
		&order.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Printf("[ERROR] PostgresOrderRepository.GetByID: %v", err)
		return nil, err
	}
	return &order, nil
}

func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		log.Printf("[ERROR] PostgresOrderRepository.UpdateStatus: %v", err)
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrOrderNotFound
	}

	if r.eventBus != nil {
		payload := fmt.Sprintf("%s:%s", id, status)
		notifyQuery := fmt.Sprintf("NOTIFY order_updates, '%s'", payload)
		if _, err := r.db.ExecContext(ctx, notifyQuery); err != nil {
			log.Printf("[WARN] failed to send NOTIFY: %v", err)
			// Не возвращаем ошибку — основная операция уже выполнена
		} else {
			log.Printf("[INFO] NOTIFY sent: order_updates -> %s", payload)
		}
	}

	return nil
}

func (r *PostgresOrderRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at
		FROM orders
		WHERE idempotency_key = $1
	`
	row := r.db.QueryRowContext(ctx, query, key)

	var order domain.Order
	err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.IdempotencyKey,
		&order.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Printf("[ERROR] PostgresOrderRepository.GetByIdempotencyKey: %v", err)
		return nil, err
	}
	return &order, nil
}
