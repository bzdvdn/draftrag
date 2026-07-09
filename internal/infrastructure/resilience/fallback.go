package resilience

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// @sk-task graceful-degradation#T1.1: ErrAllProvidersFailed sentinel (RQ-012, AC-003)
var ErrAllProvidersFailed = errors.New("all providers failed")

type aggregateError struct {
	lastErr error
}

func (e *aggregateError) Error() string {
	return fmt.Sprintf("all providers failed: %v", e.lastErr)
}

func (e *aggregateError) Unwrap() error {
	return e.lastErr
}

func (e *aggregateError) Is(target error) bool {
	return target == ErrAllProvidersFailed
}

type fallbackStatsInternal struct {
	totalCalls      atomic.Int64
	primaryFailures atomic.Int64
	fallbackCount   atomic.Int64
	mu              sync.Mutex
	lastError       error
}

func (s *fallbackStatsInternal) recordCall(primaryFailed bool) {
	s.totalCalls.Add(1)
	if primaryFailed {
		s.primaryFailures.Add(1)
	}
}

func (s *fallbackStatsInternal) recordFallback() {
	s.fallbackCount.Add(1)
}

func (s *fallbackStatsInternal) setLastError(err error) {
	s.mu.Lock()
	s.lastError = err
	s.mu.Unlock()
}

func (s *fallbackStatsInternal) snapshot() FallbackStats {
	s.mu.Lock()
	le := s.lastError
	s.mu.Unlock()
	return FallbackStats{
		TotalCalls:      s.totalCalls.Load(),
		PrimaryFailures: s.primaryFailures.Load(),
		FallbackCount:   s.fallbackCount.Load(),
		LastError:       le,
	}
}

// @sk-task graceful-degradation#T1.1: FallbackStats — thread-safe статистика цепи (RQ-011, AC-008)
type FallbackStats struct {
	TotalCalls      int64
	PrimaryFailures int64
	FallbackCount   int64
	LastError       error
}
