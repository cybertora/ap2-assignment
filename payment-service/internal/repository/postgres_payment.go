package repository

import (
	"context"
	"database/sql"
	"log"

	"payment-service/internal/domain"
)

type PostgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, transaction_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		payment.ID,
		payment.OrderID,
		payment.TransactionID,
		payment.Amount,
		payment.Status,
		payment.CreatedAt,
	)
	if err != nil {
		log.Printf("[ERROR] PostgresPaymentRepository.Create: %v", err)
		return err
	}
	return nil
}

func (r *PostgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status, created_at
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, orderID)

	var payment domain.Payment
	err := row.Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Printf("[ERROR] PostgresPaymentRepository.GetByOrderID: %v", err)
		return nil, err
	}
	return &payment, nil
}

func (r *PostgresPaymentRepository) ListByStatus(ctx context.Context, status string) ([]*domain.Payment, error) {
	var rows *sql.Rows
	var err error

	if status == "" {
		query := `
			SELECT id, order_id, transaction_id, amount, status, created_at
			FROM payments
			ORDER BY created_at DESC
		`
		rows, err = r.db.QueryContext(ctx, query)
	} else {
		query := `
			SELECT id, order_id, transaction_id, amount, status, created_at
			FROM payments
			WHERE status = $1
			ORDER BY created_at DESC
		`
		rows, err = r.db.QueryContext(ctx, query, status)
	}

	if err != nil {
		log.Printf("[ERROR] PostgresPaymentRepository.ListByStatus: %v", err)
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID,
			&p.OrderID,
			&p.TransactionID,
			&p.Amount,
			&p.Status,
			&p.CreatedAt,
		); err != nil {
			log.Printf("[ERROR] PostgresPaymentRepository.ListByStatus scan: %v", err)
			return nil, err
		}
		payments = append(payments, &p)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] PostgresPaymentRepository.ListByStatus rows: %v", err)
		return nil, err
	}

	return payments, nil
}

func (r *PostgresPaymentRepository) List(ctx context.Context, customerID string) ([]*domain.Payment, error) {
	query := "SELECT id, order_id, customer_id, amount, status, created_at FROM payments"

	var rows *sql.Rows
	var err error

	if customerID != "" {
		query += " WHERE customer_id = $1"
		rows, err = r.db.QueryContext(ctx, query, customerID)
	} else {
		rows, err = r.db.QueryContext(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		p := &domain.Payment{}
		err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	return payments, nil
}
