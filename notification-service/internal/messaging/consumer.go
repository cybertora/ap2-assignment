package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"notification-service/internal/service"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	queueName   string
	consumerTag string
	store       *service.ProcessedPaymentsStore
	mu          sync.Mutex
}

func NewConsumer(
	amqpURL string,
	exchange string,
	routingKey string,
	dlxExchange string,
	queueName string,
	dlqName string,
	store *service.ProcessedPaymentsStore,
) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}

	// Основной exchange
	if err := ch.ExchangeDeclare(
		exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare exchange %s: %w", exchange, err)
	}

	if err := ch.ExchangeDeclare(
		dlxExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare dlx %s: %w", dlxExchange, err)
	}

	//DLQ
	if _, err := ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare dlq %s: %w", dlqName, err)
	}

	if err := ch.QueueBind(
		dlqName,
		routingKey+".dlq",
		dlxExchange,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("bind dlq %s: %w", dlqName, err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    dlxExchange,
		"x-dead-letter-routing-key": routingKey + ".dlq",
	}

	if _, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		args,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare queue %s: %w", queueName, err)
	}

	if err := ch.QueueBind(
		queueName,
		routingKey,
		exchange,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("bind queue %s: %w", queueName, err)
	}

	log.Printf("[INFO] RabbitMQ consumer connected: queue=%s dlq=%s", queueName, dlqName)

	return &Consumer{
		conn:        conn,
		channel:     ch,
		queueName:   queueName,
		consumerTag: "notification-service-consumer",
		store:       store,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		c.consumerTag,
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] notification consumer received shutdown signal")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			c.handleMessage(msg)
		}
	}
}

func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var event PaymentProcessedEvent

	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[ERROR] invalid message body, sending to DLQ: %v", err)
		c.nackToDLQ(msg)
		return
	}

	if event.PaymentID == "" {
		log.Printf("[ERROR] empty payment_id, sending to DLQ")
		c.nackToDLQ(msg)
		return
	}

	if c.store.Exists(event.PaymentID) {
		log.Printf("[INFO] duplicate message skipped: payment_id=%s", event.PaymentID)
		if err := msg.Ack(false); err != nil {
			log.Printf("[WARN] failed to ack duplicate payment_id=%s: %v", event.PaymentID, err)
		}
		return
	}

	log.Printf(
		"[NOTIFICATION] payment processed | payment_id=%s order_id=%s transaction_id=%s amount=%d status=%s processed_at=%s",
		event.PaymentID,
		event.OrderID,
		event.TransactionID,
		event.Amount,
		event.Status,
		event.ProcessedAt.Format("2006-01-02T15:04:05Z07:00"),
	)

	c.store.Save(event.PaymentID)

	if err := msg.Ack(false); err != nil {
		log.Printf("[WARN] failed to ack payment_id=%s: %v", event.PaymentID, err)
	}
}

func (c *Consumer) nackToDLQ(msg amqp.Delivery) {
	if err := msg.Nack(false, false); err != nil {
		log.Printf("[WARN] failed to nack message to DLQ: %v", err)
	}
}

func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error

	if c.channel != nil {
		if err := c.channel.Cancel(c.consumerTag, false); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := c.channel.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.channel = nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.conn = nil
	}

	if firstErr == nil {
		log.Println("[INFO] RabbitMQ consumer connection closed")
	}

	return firstErr
}
