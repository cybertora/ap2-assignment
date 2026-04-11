package domain

import "errors"

var (
	ErrOrderNotFound             = errors.New("order not found")
	ErrInvalidAmount             = errors.New("amount must be greater than zero")
	ErrCancelNotAllowed          = errors.New("only pending orders can be cancelled")
	ErrPaymentServiceUnavailable = errors.New("payment service unavailable")
	ErrPaymentDeclined           = errors.New("payment declined")
	ErrDuplicateIdempotencyKey   = errors.New("duplicate idempotency key")
)
