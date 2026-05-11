package app

import "os"

type Config struct {
	RabbitMQURL             string
	PaymentEventsExchange   string
	PaymentEventsRoutingKey string
	PaymentDLXExchange      string
	PaymentQueue            string
	PaymentDLQ              string
}

func LoadConfig() *Config {
	return &Config{
		RabbitMQURL:             getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		PaymentEventsExchange:   getEnv("PAYMENT_EVENTS_EXCHANGE", "payments.events"),
		PaymentEventsRoutingKey: getEnv("PAYMENT_EVENTS_ROUTING_KEY", "payment.processed"),
		PaymentDLXExchange:      getEnv("PAYMENT_DLX_EXCHANGE", "payments.dlx"),
		PaymentQueue:            getEnv("PAYMENT_QUEUE", "notification.payment.processed"),
		PaymentDLQ:              getEnv("PAYMENT_DLQ", "notification.payment.processed.dlq"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
