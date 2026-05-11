package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// NewRateLimiter — fixed-window rate limiter на Redis.
// Алгоритм:
//
//	key = "rl:<identity>:<minute_bucket>"
//	INCR key  (атомарно)
//	если значение == 1 — EXPIRE key 60s
//	если значение > limit — 429
//
// Identity берётся из заголовка X-User-Id, либо из IP (fallback).
func NewRateLimiter(rdb *redis.Client, requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := c.GetHeader("X-User-Id")
		if identity == "" {
			identity = c.ClientIP()
		}

		bucket := time.Now().UTC().Unix() / 60 // одно окно = 1 минута
		key := fmt.Sprintf("rl:%s:%d", identity, bucket)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Если Redis недоступен — НЕ блокируем трафик (fail-open).
			log.Printf("[RATE-LIMIT WARN] redis incr failed: %v — fail-open", err)
			c.Next()
			return
		}
		if count == 1 {
			_ = rdb.Expire(ctx, key, time.Minute).Err()
		}

		remaining := int64(requestsPerMinute) - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if count > int64(requestsPerMinute) {
			retryAfter := 60 - (time.Now().UTC().Unix() % 60)
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "too many requests",
				"limit":       requestsPerMinute,
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}
