// @sk-task rate-limiting-llm#T0.3: Token bucket tests (AC-001, AC-002, AC-004)

package resilience

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// @sk-test rate-limiting-llm#T0.3: TestTokenBucket_Take_Blocks (AC-001)
func TestTokenBucket_Take_Blocks(t *testing.T) {
	tb := newTokenBucket(1, 1)

	ctx := context.Background()
	start := time.Now()

	_, err := tb.Take(ctx, 1)
	assert.NoError(t, err)

	_, err = tb.Take(ctx, 1)
	assert.NoError(t, err)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond,
		"second take should block ~1s for refill")
	tb.Stop()
}

// @sk-test rate-limiting-llm#T0.3: TestTokenBucket_Take_ContextCancel (AC-002)
func TestTokenBucket_Take_ContextCancel(t *testing.T) {
	tb := newTokenBucket(1, 1)

	ctx, cancel := context.WithCancel(context.Background())

	_, err := tb.Take(ctx, 1)
	assert.NoError(t, err)

	cancel()

	// Allow a small window for cancellation to propagate
	time.Sleep(10 * time.Millisecond)

	_, err = tb.Take(ctx, 1)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	tb.Stop()
}

// @sk-test rate-limiting-llm#T0.3: TestTokenBucket_Take_RefillRate (AC-001, AC-004)
func TestTokenBucket_Take_RefillRate(t *testing.T) {
	tb := newTokenBucket(10, 5)

	ctx := context.Background()
	start := time.Now()

	// Consume all burst tokens
	for range 5 {
		_, err := tb.Take(ctx, 1)
		assert.NoError(t, err)
	}

	// 6th call must wait at least ~100ms for refill
	_, err := tb.Take(ctx, 1)
	assert.NoError(t, err)
	elapsed := time.Since(start)

	// 5 burst = instant, 6th = wait ~100ms, total ≥ 6.1s? No.
	// At rate=10, burst=5: first 5 take from burst (~0), 6th waits ~100ms
	assert.GreaterOrEqual(t, elapsed, 90*time.Millisecond)
	tb.Stop()
}
