package domain

import "time"

const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"
)

const MaxPaymentAmount int64 = 100000

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64 // Amount in cents. NEVER use float64 for money.
	Status        string
	CreatedAt     time.Time
}

func (p *Payment) IsAuthorized() bool {
	return p.Status == StatusAuthorized
}
