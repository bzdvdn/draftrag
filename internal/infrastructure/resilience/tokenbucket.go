// @sk-task rate-limiting-llm#T0.2: Token bucket (AC-001, AC-002, AC-004)

package resilience

import (
	"context"
	"sync"
	"time"
)

type tokenBucket struct {
	mu        sync.Mutex
	cond      *sync.Cond
	tokens    float64
	maxTokens float64
	interval  time.Duration
	ticker    *time.Ticker
	closeCh   chan struct{}
}

func newTokenBucket(rate, burst float64) *tokenBucket {
	interval := time.Duration(float64(time.Second) / rate)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	tb := &tokenBucket{
		tokens:    burst,
		maxTokens: burst,
		interval:  interval,
		ticker:    time.NewTicker(interval),
		closeCh:   make(chan struct{}),
	}
	tb.cond = sync.NewCond(&tb.mu)

	go func() {
		for {
			select {
			case <-tb.ticker.C:
				tb.mu.Lock()
				if tb.tokens < tb.maxTokens {
					tb.tokens++
					if tb.tokens > tb.maxTokens {
						tb.tokens = tb.maxTokens
					}
				}
				tb.cond.Broadcast()
				tb.mu.Unlock()
			case <-tb.closeCh:
				tb.ticker.Stop()
				return
			}
		}
	}()

	return tb
}

func (tb *tokenBucket) Take(ctx context.Context, n int64) (waited bool, err error) {
	tb.mu.Lock()

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		tb.mu.Unlock()
		return false, nil
	}

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			tb.cond.Broadcast()
		case <-done:
		}
	}()

	for tb.tokens < float64(n) {
		select {
		case <-ctx.Done():
			tb.mu.Unlock()
			return true, ctx.Err()
		default:
		}
		tb.cond.Wait()
	}

	tb.tokens -= float64(n)
	tb.mu.Unlock()
	return true, nil
}

func (tb *tokenBucket) Stop() {
	select {
	case <-tb.closeCh:
	default:
		close(tb.closeCh)
	}
}
