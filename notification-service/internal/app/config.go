package app

import (
	"os"
	"strconv"
)

type Config struct {
	RabbitMQURL             string
	PaymentEventsExchange   string
	PaymentEventsRoutingKey string
	PaymentDLXExchange      string
	PaymentQueue            string
	PaymentDLQ              string

	// Redis
	RedisAddr           string
	RedisPassword       string
	RedisDB             int
	IdempotencyTTLHours int

	// Adapter Pattern
	ProviderMode string // REAL | SIMULATED
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Retry policy
	RetryMaxAttempts int
	RetryBaseDelayMS int
	WorkerPoolSize   int
}

func LoadConfig() *Config {
	return &Config{
		RabbitMQURL:             getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		PaymentEventsExchange:   getEnv("PAYMENT_EVENTS_EXCHANGE", "payments.events"),
		PaymentEventsRoutingKey: getEnv("PAYMENT_EVENTS_ROUTING_KEY", "payment.processed"),
		PaymentDLXExchange:      getEnv("PAYMENT_DLX_EXCHANGE", "payments.dlx"),
		PaymentQueue:            getEnv("PAYMENT_QUEUE", "notification.payment.processed"),
		PaymentDLQ:              getEnv("PAYMENT_DLQ", "notification.payment.processed.dlq"),

		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnvInt("REDIS_DB", 1),
		IdempotencyTTLHours: getEnvInt("IDEMPOTENCY_TTL_HOURS", 24),

		ProviderMode: getEnv("PROVIDER_MODE", "SIMULATED"),
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "no-reply@shop.local"),

		RetryMaxAttempts: getEnvInt("RETRY_MAX_ATTEMPTS", 5),
		RetryBaseDelayMS: getEnvInt("RETRY_BASE_DELAY_MS", 2000),
		WorkerPoolSize:   getEnvInt("WORKER_POOL_SIZE", 4),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
