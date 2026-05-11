package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string
	mu         sync.Mutex
}

func NewPublisher(amqpURL, exchange, routingKey string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		exchange,
		"direct",
		true,  // durable
		false, // auto-delete
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare exchange %s: %w", exchange, err)
	}

	log.Printf("[INFO] RabbitMQ publisher connected: exchange=%s routing_key=%s", exchange, routingKey)

	return &Publisher{
		conn:       conn,
		channel:    ch,
		exchange:   exchange,
		routingKey: routingKey,
	}, nil
}

func (p *Publisher) PublishPaymentProcessed(ctx context.Context, event PaymentProcessedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment processed event: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		lastErr = p.channel.PublishWithContext(
			pubCtx,
			p.exchange,
			p.routingKey,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Timestamp:    time.Now().UTC(),
				Body:         body,
			},
		)

		cancel()

		if lastErr == nil {
			log.Printf("[INFO] published PaymentProcessed event: payment_id=%s order_id=%s", event.PaymentID, event.OrderID)
			return nil
		}

		log.Printf("[WARN] publish attempt %d/3 failed for payment_id=%s: %v", attempt, event.PaymentID, lastErr)
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return fmt.Errorf("publish PaymentProcessed after retries: %w", lastErr)
}

func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error

	if p.channel != nil {
		if err := p.channel.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.channel = nil
	}

	if p.conn != nil {
		if err := p.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.conn = nil
	}

	if firstErr == nil {
		log.Println("[INFO] RabbitMQ publisher connection closed")
	}

	return firstErr
}
