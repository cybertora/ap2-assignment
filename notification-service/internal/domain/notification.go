package domain

import "context"

// Notification — доменная сущность уведомления (декаплинг от транспортного DTO).
type Notification struct {
	PaymentID     string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string
	To            string // email получателя — для демо генерируется из CustomerID
	Subject       string
	Body          string
}

// EmailSender / NotificationProvider — порт для отправителя уведомлений.
// Use-case (worker) НЕ зависит от SMTP/Mailjet/Mock — он работает только с этим интерфейсом.
type EmailSender interface {
	Send(ctx context.Context, n Notification) error
	Name() string
}
