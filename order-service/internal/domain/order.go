package domain

import "time"

const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

type Order struct {
	ID             string
	CustomerID     string
	ItemName       string
	Amount         int64
	Status         string
	IdempotencyKey *string
	CreatedAt      time.Time
}

func (o *Order) IsPaid() bool {
	return o.Status == StatusPaid
}

func (o *Order) CanBeCancelled() bool {
	return o.Status == StatusPending
}
