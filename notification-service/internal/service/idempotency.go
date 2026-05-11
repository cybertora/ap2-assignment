package service

import "sync"

type ProcessedPaymentsStore struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

func NewProcessedPaymentsStore() *ProcessedPaymentsStore {
	return &ProcessedPaymentsStore{
		seen: make(map[string]struct{}),
	}
}

func (s *ProcessedPaymentsStore) Exists(paymentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.seen[paymentID]
	return ok
}

func (s *ProcessedPaymentsStore) Save(paymentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seen[paymentID] = struct{}{}
}
