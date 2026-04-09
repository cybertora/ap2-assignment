package domain

import "errors"

var (
	ErrPaymentNotFound = errors.New("payment not found")

	ErrInvalidAmount = errors.New("amount must be greater than zero")

	ErrAmountExceedsLimit = errors.New("amount exceeds maximum payment limit")
)
