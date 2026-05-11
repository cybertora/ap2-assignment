package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"notification-service/internal/domain"
	"notification-service/internal/service"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	queueName   string
	consumerTag string
	worker      *service.NotificationWorker
	poolSize    int
	mu          sync.Mutex
}

func NewConsumer(
	amqpURL, exchange, routingKey, dlxExchange, queueName, dlqName string,
	worker *service.NotificationWorker,
	poolSize int,
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

	// Prefetch = poolSize — чтобы воркер-пулл всегда был "сыт", но не голоден.
	if err := ch.Qos(poolSize, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}

	// Объявления как в А3 (exchange + dlx + dlq + queue + bindings).
	if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return nil, closeAndWrap(ch, conn, "declare exchange", err)
	}
	if err := ch.ExchangeDeclare(dlxExchange, "direct", true, false, false, false, nil); err != nil {
		return nil, closeAndWrap(ch, conn, "declare dlx", err)
	}
	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return nil, closeAndWrap(ch, conn, "declare dlq", err)
	}
	if err := ch.QueueBind(dlqName, routingKey+".dlq", dlxExchange, false, nil); err != nil {
		return nil, closeAndWrap(ch, conn, "bind dlq", err)
	}
	args := amqp.Table{
		"x-dead-letter-exchange":    dlxExchange,
		"x-dead-letter-routing-key": routingKey + ".dlq",
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return nil, closeAndWrap(ch, conn, "declare queue", err)
	}
	if err := ch.QueueBind(queueName, routingKey, exchange, false, nil); err != nil {
		return nil, closeAndWrap(ch, conn, "bind queue", err)
	}

	log.Printf("[INFO] RabbitMQ consumer ready: queue=%s dlq=%s pool=%d", queueName, dlqName, poolSize)

	return &Consumer{
		conn:        conn,
		channel:     ch,
		queueName:   queueName,
		consumerTag: "notification-service-consumer",
		worker:      worker,
		poolSize:    poolSize,
	}, nil
}

func closeAndWrap(ch *amqp.Channel, conn *amqp.Connection, ctxLabel string, err error) error {
	_ = ch.Close()
	_ = conn.Close()
	return fmt.Errorf("%s: %w", ctxLabel, err)
}

// Run — запускает worker-pool: API/AMQP-поток не блокируется на отправке.
// Каждое сообщение пушится в канал jobs, воркеры разгребают параллельно.
func (c *Consumer) Run(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		c.consumerTag,
		false, // manual ack — критично для надёжности
		false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}

	jobs := make(chan amqp.Delivery, c.poolSize*2)
	var wg sync.WaitGroup

	for i := 0; i < c.poolSize; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for msg := range jobs {
				c.handle(ctx, workerID, msg)
			}
		}(i + 1)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] consumer shutdown signal — draining...")
			close(jobs)
			wg.Wait()
			return nil
		case msg, ok := <-msgs:
			if !ok {
				close(jobs)
				wg.Wait()
				return nil
			}
			jobs <- msg
		}
	}
}

// handle — обёртка над worker.Process: парсинг, ack/nack-логика.
func (c *Consumer) handle(ctx context.Context, workerID int, msg amqp.Delivery) {
	var event PaymentProcessedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[WORKER %d] invalid body -> DLQ: %v", workerID, err)
		_ = msg.Nack(false, false)
		return
	}
	if event.PaymentID == "" {
		log.Printf("[WORKER %d] empty payment_id -> DLQ", workerID)
		_ = msg.Nack(false, false)
		return
	}

	n := domain.Notification{
		PaymentID:     event.PaymentID,
		OrderID:       event.OrderID,
		TransactionID: event.TransactionID,
		Amount:        event.Amount,
		Status:        event.Status,
		To:            fmt.Sprintf("customer-%s@shop.local", event.OrderID),
		Subject:       "Your payment is " + event.Status,
		Body: fmt.Sprintf(
			"Hi! Your payment for order %s was %s.\nTransaction: %s\nAmount: %d",
			event.OrderID, event.Status, event.TransactionID, event.Amount,
		),
	}

	if err := c.worker.Process(ctx, n); err != nil {
		// Все retry-попытки уже использованы внутри worker.Process.
		// Отправляем в DLQ, чтобы не зависало в основной очереди.
		log.Printf("[WORKER %d] -> DLQ payment_id=%s: %v", workerID, event.PaymentID, err)
		_ = msg.Nack(false, false)
		return
	}

	if err := msg.Ack(false); err != nil {
		log.Printf("[WORKER %d] ack failed payment_id=%s: %v", workerID, event.PaymentID, err)
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
