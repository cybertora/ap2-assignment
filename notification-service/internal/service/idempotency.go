package service

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisIdempotencyStore — идемпотентность по payment_id, на Redis.
// Используем SETNX (атомарную операцию "set if not exists") как
// одновременно "проверка существования" + "захват слота" в одной команде.
// Это спасает от гонки между несколькими репликами worker-а.
type RedisIdempotencyStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisIdempotencyStore(rdb *redis.Client, ttlHours int) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{
		rdb: rdb,
		ttl: time.Duration(ttlHours) * time.Hour,
	}
}

func key(paymentID string) string { return "notif:idem:" + paymentID }

// Acquire — атомарно резервирует payment_id.
// true  -> сообщение ЕЩЁ не обрабатывалось, можно отправлять
// false -> УЖЕ обработано (или сейчас обрабатывается другим воркером) — skip
func (s *RedisIdempotencyStore) Acquire(ctx context.Context, paymentID string) (bool, error) {
	ok, err := s.rdb.SetNX(ctx, key(paymentID), "processed", s.ttl).Result()
	if err != nil {
		log.Printf("[IDEMP ERROR] setnx payment_id=%s: %v", paymentID, err)
		return false, err
	}
	if !ok {
		log.Printf("[IDEMP] duplicate detected: payment_id=%s — skipping", paymentID)
	}
	return ok, nil
}

// Release — снять "захват" в случае, если отправка окончательно провалилась
// и мы хотим разрешить повторную обработку (используется при exhausted retries
// в политике "at-least-once без потерь").
func (s *RedisIdempotencyStore) Release(ctx context.Context, paymentID string) {
	if err := s.rdb.Del(ctx, key(paymentID)).Err(); err != nil {
		log.Printf("[IDEMP WARN] release payment_id=%s: %v", paymentID, err)
	}
}
