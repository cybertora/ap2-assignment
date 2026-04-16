package http

import "time"

type CreateOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

type OrderResponse struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	ItemName   string    `json:"item_name"`
	Amount     int64     `json:"amount"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type PaymentListItem struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
