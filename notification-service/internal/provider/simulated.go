package provider

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"notification-service/internal/domain"
)

// SimulatedSender — мок-адаптер: имитация сети + случайные ошибки.
// Полезен в разработке и при демонстрации retry / backoff.
type SimulatedSender struct {
	failureRate float64 // 0.0..1.0; 0.3 = 30% сбоев
	minLatency  time.Duration
	maxLatency  time.Duration
	rng         *rand.Rand
}

func NewSimulatedSender() *SimulatedSender {
	return &SimulatedSender{
		failureRate: 0.3,
		minLatency:  200 * time.Millisecond,
		maxLatency:  1200 * time.Millisecond,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *SimulatedSender) Name() string { return "SIMULATED" }

func (s *SimulatedSender) Send(ctx context.Context, n domain.Notification) error {
	// Случайная задержка имитирует сетевые расходы.
	d := s.minLatency + time.Duration(s.rng.Int63n(int64(s.maxLatency-s.minLatency)))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
	}

	if s.rng.Float64() < s.failureRate {
		log.Printf("[PROVIDER SIMULATED] transient failure payment_id=%s", n.PaymentID)
		return errors.New("simulated transient provider error")
	}

	log.Printf("[PROVIDER SIMULATED] ✉️  sent to=%s subject=%q payment_id=%s",
		n.To, n.Subject, n.PaymentID)
	return nil
}
