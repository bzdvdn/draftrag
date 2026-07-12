// @sk-test prod-issues#T1.3: Streaming rate limiter tests (AC-012, AC-013)

package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// @sk-test prod-issues#T1.3: GenerateStream blocks on rate limit (AC-012)
func TestTokenBucketStreamingLLMProvider_BlocksOnGenerateStream(t *testing.T) {
	calls := 0
	mockStream := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			calls++
			ch := make(chan string, 1)
			ch <- "token"
			close(ch)
			return ch, nil
		},
	}

	p := NewTokenBucketStreamingLLMProvider(mockStream, 1, 1, nil)

	// First call: immediate
	ch, err := p.GenerateStream(context.Background(), "sys", "user")
	require.NoError(t, err)
	<-ch

	start := time.Now()
	// Second call: should block
	ch, err = p.GenerateStream(context.Background(), "sys", "user")
	require.NoError(t, err)
	<-ch
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond,
		"second streaming call should block ~1s for refill")
	assert.Equal(t, 2, calls)
}

// @sk-test prod-issues#T1.3: Generate delegates to inner Generate (AC-013)
func TestTokenBucketStreamingLLMProvider_Generate(t *testing.T) {
	genCalls := 0
	mockStream := &mockStreamingLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			genCalls++
			return "response", nil
		},
	}

	p := NewTokenBucketStreamingLLMProvider(mockStream, 0, 0, nil)

	result, err := p.Generate(context.Background(), "sys", "user")
	require.NoError(t, err)
	assert.Equal(t, "response", result)

	// passthrough: no rate limiting
	start := time.Now()
	result, err = p.Generate(context.Background(), "sys", "user")
	require.NoError(t, err)
	assert.Equal(t, "response", result)
	assert.Less(t, time.Since(start), 500*time.Millisecond)
	assert.Equal(t, 2, genCalls)
}

// @sk-test prod-issues#T1.3: context cancellation during wait
func TestTokenBucketStreamingLLMProvider_ContextCancel(t *testing.T) {
	calls := 0
	mockStream := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			calls++
			ch := make(chan string, 1)
			ch <- "token"
			close(ch)
			return ch, nil
		},
	}

	p := NewTokenBucketStreamingLLMProvider(mockStream, 1, 1, nil)

	// consume first token
	ch, err := p.GenerateStream(context.Background(), "sys", "user")
	require.NoError(t, err)
	<-ch

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	time.Sleep(10 * time.Millisecond)

	_, err = p.GenerateStream(ctx, "sys", "user")
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

// @sk-test prod-issues#T1.3: error passthrough from inner provider
func TestTokenBucketStreamingLLMProvider_ErrorPassthrough(t *testing.T) {
	expectedErr := errors.New("provider error")
	mockStream := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			return nil, expectedErr
		},
	}

	p := NewTokenBucketStreamingLLMProvider(mockStream, 10, 5, nil)
	_, err := p.GenerateStream(context.Background(), "sys", "user")
	assert.Error(t, err)
}

// @sk-test prod-issues#T1.3: Health passthrough
func TestTokenBucketStreamingLLMProvider_Health(t *testing.T) {
	healthCalled := false
	mockStream := &mockStreamingLLMProvider{
		healthFn: func(ctx context.Context) error {
			healthCalled = true
			return nil
		},
	}

	p := NewTokenBucketStreamingLLMProvider(mockStream, 0, 0, nil)
	err := p.Health(context.Background())
	assert.NoError(t, err)
	assert.True(t, healthCalled)
}

// @sk-test prod-issues#T1.3: TokensPerSecond returns 0 for passthrough
func TestTokenBucketStreamingLLMProvider_TokensPerSecond(t *testing.T) {
	mockStream := &mockStreamingLLMProvider{}
	p := NewTokenBucketStreamingLLMProvider(mockStream, 0, 0, nil)
	assert.Equal(t, float64(0), p.TokensPerSecond())

	p2 := NewTokenBucketStreamingLLMProvider(mockStream, 10, 3, nil)
	assert.InDelta(t, 10, p2.TokensPerSecond(), 0.1)
}
