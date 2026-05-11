package app

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	ServerPort string

	PaymentGRPCAddr string
	GRPCPort        string

	// ===== Redis & cache =====
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	CacheTTLSeconds int

	// ===== Rate limiter (Bonus) =====
	RateLimitEnabled bool
	RateLimitRPM     int
}

func LoadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5433"),
		DBUser:     getEnv("DB_USER", "order_user"),
		DBPassword: getEnv("DB_PASSWORD", "order_pass"),
		DBName:     getEnv("DB_NAME", "order_db"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		ServerPort: getEnv("SERVER_PORT", "8080"),

		PaymentGRPCAddr: getEnv("PAYMENT_GRPC_ADDR", "localhost:50051"),
		GRPCPort:        getEnv("GRPC_PORT", "50052"),

		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),

		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRPM:     getEnvInt("RATE_LIMIT_RPM", 10),
	}
}

func ConnectDB(cfg *Config) *sql.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("[FATAL] failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[FATAL] failed to ping database: %v", err)
	}
	log.Println("[INFO] connected to order database")
	return db
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

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}
