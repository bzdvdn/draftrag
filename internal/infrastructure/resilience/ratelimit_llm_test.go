package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// @sk-test rate-limiting-llm#T1.3: TestTokenBucketLLMProvider_Blocks (AC-001)
func TestTokenBucketLLMProvider_Blocks(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "sys", "user").
		Return("ok", nil).Times(2)

	p := NewTokenBucketLLMProvider(mockLLM, 1, 1, nil)

	start := time.Now()
	_, err := p.Generate(context.Background(), "sys", "user")
	assert.NoError(t, err)

	_, err = p.Generate(context.Background(), "sys", "user")
	assert.NoError(t, err)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond,
		"second call should block ~1s for refill")
	mockLLM.AssertExpectations(t)
}

// @sk-test rate-limiting-llm#T1.3: TestTokenBucketLLMProvider_ContextCancel (AC-002)
func TestTokenBucketLLMProvider_ContextCancel(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "sys", "user").
		Return("ok", nil).Once()

	p := NewTokenBucketLLMProvider(mockLLM, 1, 1, nil)

	ctx, cancel := context.WithCancel(context.Background())

	_, err := p.Generate(ctx, "sys", "user")
	assert.NoError(t, err)

	cancel()
	time.Sleep(10 * time.Millisecond)

	_, err = p.Generate(ctx, "sys", "user")
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	mockLLM.AssertExpectations(t)
}

// @sk-test rate-limiting-llm#T1.3: TestTokenBucketLLMProvider_Passthrough (AC-003)
func TestTokenBucketLLMProvider_Passthrough(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "sys", "user").
		Return("ok", nil).Times(5)

	p := NewTokenBucketLLMProvider(mockLLM, 0, 0, nil)

	start := time.Now()
	for range 5 {
		_, err := p.Generate(context.Background(), "sys", "user")
		assert.NoError(t, err)
	}
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 500*time.Millisecond,
		"passthrough should not block")
	mockLLM.AssertExpectations(t)
}

// @sk-test rate-limiting-llm#T3.2: TestTokenBucketLLMProvider_Hooks (AC-005)
func TestTokenBucketLLMProvider_Hooks(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "sys", "user").
		Return("ok", nil).Times(2)

	mockHooks := new(MockHooks)
	mockHooks.On("StageStart", mock.Anything, mock.MatchedBy(func(ev domain.StageStartEvent) bool {
		return ev.Stage == domain.HookStageRateLimit && ev.Operation == "rate_limit_wait"
	})).Return(nil).Once()
	mockHooks.On("StageEnd", mock.Anything, mock.MatchedBy(func(ev domain.StageEndEvent) bool {
		return ev.Stage == domain.HookStageRateLimit && ev.Operation == "rate_limit_wait"
	})).Return().Once()

	p := NewTokenBucketLLMProvider(mockLLM, 1, 1, mockHooks)

	_, err := p.Generate(context.Background(), "sys", "user")
	assert.NoError(t, err)

	_, err = p.Generate(context.Background(), "sys", "user")
	assert.NoError(t, err)

	mockHooks.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
}

// @sk-test rate-limiting-llm#T4.1: TestTokenBucketLLMProvider_WithRetry (AC-006)
func TestTokenBucketLLMProvider_WithRetry(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "sys", "user").
		Return("", errors.New("429 rate limit")).Once()
	mockLLM.On("Generate", mock.Anything, "sys", "user").
		Return("ok", nil).Once()

	tbProvider := NewTokenBucketLLMProvider(mockLLM, 100, 10, nil)
	retryProvider := NewRetryLLMProvider(tbProvider,
		&RetryConfig{MaxRetries: 1, Backoff: &Backoff{BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Multiplier: 1, JitterFactor: 0}},
		nil, nil, nil)

	result, err := retryProvider.Generate(context.Background(), "sys", "user")
	assert.NoError(t, err)
	assert.Equal(t, "ok", result)
	mockLLM.AssertExpectations(t)
}
