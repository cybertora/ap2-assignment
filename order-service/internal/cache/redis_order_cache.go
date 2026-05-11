package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

// RedisOrderCache — реализация порта usecase.OrderCache.
// Хранит сериализованные заказы под ключом order:<id> с настраиваемым TTL.
type RedisOrderCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisOrderCache(rdb *redis.Client, ttlSeconds int) *RedisOrderCache {
	return &RedisOrderCache{
		rdb: rdb,
		ttl: time.Duration(ttlSeconds) * time.Second,
	}
}

func key(id string) string { return "order:" + id }

// Get — cache-aside read.
// Возвращает (nil, nil) при cache miss — это сигнал use-case-у пойти в БД.
func (c *RedisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	raw, err := c.rdb.Get(ctx, key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		log.Printf("[CACHE MISS] order:%s", id)
		return nil, nil
	}
	if err != nil {
		// Любая другая ошибка — деградируем до БД, не валим запрос.
		log.Printf("[CACHE ERROR] get order:%s: %v", id, err)
		return nil, nil
	}

	var o domain.Order
	if err := json.Unmarshal(raw, &o); err != nil {
		log.Printf("[CACHE ERROR] unmarshal order:%s: %v — deleting bad entry", id, err)
		_ = c.rdb.Del(ctx, key(id)).Err()
		return nil, nil
	}
	log.Printf("[CACHE HIT] order:%s status=%s", id, o.Status)
	return &o, nil
}

func (c *RedisOrderCache) Set(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	if err := c.rdb.Set(ctx, key(order.ID), data, c.ttl).Err(); err != nil {
		log.Printf("[CACHE ERROR] set order:%s: %v", order.ID, err)
		return err
	}
	log.Printf("[CACHE SET] order:%s ttl=%s", order.ID, c.ttl)
	return nil
}

// Delete — атомарная инвалидация. Один Redis DEL — атомарная команда на стороне сервера.
func (c *RedisOrderCache) Delete(ctx context.Context, id string) error {
	n, err := c.rdb.Del(ctx, key(id)).Result()
	if err != nil {
		log.Printf("[CACHE ERROR] delete order:%s: %v", id, err)
		return err
	}
	log.Printf("[CACHE INVALIDATE] order:%s removed=%d", id, n)
	return nil
}
