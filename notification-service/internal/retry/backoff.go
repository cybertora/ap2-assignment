package retry

import (
	"context"
	"log"
	"math/rand"
	"time"
)

// Policy — политика повторов с экспоненциальной задержкой + jitter.
//
//	попытка 1: base
//	попытка 2: base * 2
//	попытка 3: base * 4
//	попытка n: base * 2^(n-1)
type Policy struct {
	MaxAttempts int
	BaseDelay   time.Duration
}

func New(maxAttempts, baseDelayMS int) Policy {
	return Policy{
		MaxAttempts: maxAttempts,
		BaseDelay:   time.Duration(baseDelayMS) * time.Millisecond,
	}
}

// Do — выполняет fn с экспоненциальным backoff.
// Если fn вернула nil — возвращает nil.
// Если все попытки исчерпаны — возвращает последнюю ошибку.
func (p Policy) Do(ctx context.Context, label string, fn func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				log.Printf("[RETRY %s] succeeded on attempt %d/%d", label, attempt, p.MaxAttempts)
			}
			return nil
		}
		lastErr = err

		if attempt == p.MaxAttempts {
			break
		}

		// 2^(attempt-1) * base + jitter (±20%)
		delay := p.BaseDelay * (1 << (attempt - 1))
		jitter := time.Duration(rand.Int63n(int64(delay) / 5))
		sleep := delay + jitter

		log.Printf("[RETRY %s] attempt %d/%d failed: %v — sleeping %s",
			label, attempt, p.MaxAttempts, err, sleep)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}
	}
	log.Printf("[RETRY %s] exhausted after %d attempts: %v", label, p.MaxAttempts, lastErr)
	return lastErr
}
