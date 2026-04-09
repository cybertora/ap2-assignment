package repository

import (
	"context"
	"database/sql"
	"log"

	"order-service/internal/domain"
)

type PostgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
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
	return nil
}

func (r *PostgresOrderRepository) GetByCustomerID(ctx context.Context, customerID string) ([]*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		log.Printf("[ERROR] PostgresOrderRepository.GetByCustomerID: %v", err)
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var order domain.Order
		err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&order.ItemName,
			&order.Amount,
			&order.Status,
			&order.IdempotencyKey,
			&order.CreatedAt,
		)
		if err != nil {
			log.Printf("[ERROR] PostgresOrderRepository.GetByCustomerID scan: %v", err)
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] PostgresOrderRepository.GetByCustomerID rows iteration: %v", err)
		return nil, err
	}

	return orders, nil
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
