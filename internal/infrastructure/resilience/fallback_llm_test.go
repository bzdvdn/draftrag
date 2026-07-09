package resilience

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLLMProvider struct {
	generateFn func(ctx context.Context, systemPrompt, userMessage string) (string, error)
	healthFn   func(ctx context.Context) error
}

func (m *mockLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return m.generateFn(ctx, systemPrompt, userMessage)
}

func (m *mockLLMProvider) Health(ctx context.Context) error {
	if m.healthFn != nil {
		return m.healthFn(ctx)
	}
	return nil
}

type mockHooks struct {
	onStart []domain.StageStartEvent
	onEnd   []domain.StageEndEvent
}

func (m *mockHooks) StageStart(ctx context.Context, ev domain.StageStartEvent) context.Context {
	m.onStart = append(m.onStart, ev)
	return ctx
}

func (m *mockHooks) StageEnd(ctx context.Context, ev domain.StageEndEvent) {
	m.onEnd = append(m.onEnd, ev)
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_RetryableFailover (AC-001)
func TestFallbackLLM_RetryableFailover(t *testing.T) {
	primary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "", WrapRetryable(errors.New("primary down"))
		},
	}
	secondary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "secondary response", nil
		},
	}

	fb, err := NewFallbackLLM([]domain.LLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	result, err := fb.Generate(context.Background(), "system", "user")
	require.NoError(t, err)
	assert.Equal(t, "secondary response", result)

	stats := fb.Stats()
	assert.Equal(t, int64(1), stats.TotalCalls)
	assert.Equal(t, int64(1), stats.PrimaryFailures)
	assert.Equal(t, int64(1), stats.FallbackCount)
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_NonRetryableError (AC-002)
func TestFallbackLLM_NonRetryableError(t *testing.T) {
	expectedErr := errors.New("invalid auth")
	primary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "", WrapNonRetryable(expectedErr)
		},
	}
	secondary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "should not reach", nil
		},
	}

	fb, err := NewFallbackLLM([]domain.LLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	result, err := fb.Generate(context.Background(), "system", "user")
	require.Error(t, err)
	assert.Empty(t, result)
	assert.True(t, errors.Is(err, expectedErr) || err.Error() == expectedErr.Error())

	stats := fb.Stats()
	assert.Equal(t, int64(0), stats.FallbackCount)
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_AllProvidersFailed (AC-003)
func TestFallbackLLM_AllProvidersFailed(t *testing.T) {
	primary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "", WrapRetryable(errors.New("primary down"))
		},
	}
	secondary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "", WrapRetryable(errors.New("secondary down"))
		},
	}

	fb, err := NewFallbackLLM([]domain.LLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	_, err = fb.Generate(context.Background(), "system", "user")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAllProvidersFailed), "expected ErrAllProvidersFailed, got %v", err)

	stats := fb.Stats()
	assert.Equal(t, int64(2), stats.FallbackCount)
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_HealthNoFallback (AC-004)
func TestFallbackLLM_HealthNoFallback(t *testing.T) {
	healthErr := errors.New("circuit breaker is open")
	primary := &mockLLMProvider{
		healthFn: func(ctx context.Context) error {
			return healthErr
		},
	}
	secondary := &mockLLMProvider{
		healthFn: func(ctx context.Context) error {
			return nil
		},
	}

	fb, err := NewFallbackLLM([]domain.LLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	err = fb.Health(context.Background())
	require.Error(t, err)
	assert.Equal(t, healthErr.Error(), err.Error())
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_HooksOnError (AC-005)
func TestFallbackLLM_HooksOnError(t *testing.T) {
	hooks := &mockHooks{}
	primary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "", WrapRetryable(errors.New("primary down"))
		},
	}
	secondary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "ok", nil
		},
	}

	fb, err := NewFallbackLLM([]domain.LLMProvider{primary, secondary}, nil, hooks)
	require.NoError(t, err)

	result, err := fb.Generate(context.Background(), "system", "user")
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Len(t, hooks.onEnd, 1)
	assert.Error(t, hooks.onEnd[0].Err)
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_EmptyChain (AC-009)
func TestFallbackLLM_EmptyChain(t *testing.T) {
	_, err := NewFallbackLLM([]domain.LLMProvider{}, nil, nil)
	require.Error(t, err)
}

// @sk-test graceful-degradation#T2.2: TestFallbackLLM_StatsCounters (AC-008)
func TestFallbackLLM_StatsCounters(t *testing.T) {
	callCount := 0
	primary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			callCount++
			if callCount <= 2 {
				return "", WrapRetryable(errors.New("primary down"))
			}
			return "ok", nil
		},
	}
	secondary := &mockLLMProvider{
		generateFn: func(ctx context.Context, _, _ string) (string, error) {
			return "secondary response", nil
		},
	}

	fb, err := NewFallbackLLM([]domain.LLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	// Call 1: primary fails, secondary succeeds
	_, err = fb.Generate(context.Background(), "system", "user")
	require.NoError(t, err)

	// Call 2: primary fails, secondary succeeds
	_, err = fb.Generate(context.Background(), "system", "user")
	require.NoError(t, err)

	// Call 3: primary succeeds
	callCount = 0 // reset for third call pattern
	primary.generateFn = func(ctx context.Context, _, _ string) (string, error) {
		return "primary ok", nil
	}
	_, err = fb.Generate(context.Background(), "system", "user")
	require.NoError(t, err)

	stats := fb.Stats()
	assert.Equal(t, int64(3), stats.TotalCalls)
	assert.Equal(t, int64(2), stats.FallbackCount)
}

type mockStreamingLLMProvider struct {
	generateFn       func(ctx context.Context, systemPrompt, userMessage string) (string, error)
	healthFn         func(ctx context.Context) error
	generateStreamFn func(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error)
}

func (m *mockStreamingLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return m.generateFn(ctx, systemPrompt, userMessage)
}

func (m *mockStreamingLLMProvider) Health(ctx context.Context) error {
	if m.healthFn != nil {
		return m.healthFn(ctx)
	}
	return nil
}

func (m *mockStreamingLLMProvider) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	return m.generateStreamFn(ctx, systemPrompt, userMessage)
}

// @sk-test graceful-degradation#T3.3: TestFallbackStreamingLLM_RetryableFailover (AC-006)
func TestFallbackStreamingLLM_RetryableFailover(t *testing.T) {
	primary := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			ch := make(chan string)
			close(ch)
			return ch, nil
		},
	}
	secondary := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			ch := make(chan string, 1)
			ch <- "secondary token"
			close(ch)
			return ch, nil
		},
	}

	fb, err := NewFallbackStreamingLLM([]domain.StreamingLLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	stream, err := fb.GenerateStream(context.Background(), "system", "user")
	require.NoError(t, err)

	var tokens []string
	for t := range stream {
		tokens = append(tokens, t)
	}
	assert.Equal(t, []string{"secondary token"}, tokens)

	stats := fb.Stats()
	assert.Equal(t, int64(1), stats.FallbackCount)
}

// @sk-test graceful-degradation#T3.4: TestFallbackUsageAwareLLM_RetryableFailover (AC-007)
func TestFallbackUsageAwareLLM_RetryableFailover(t *testing.T) {
	primary := &mockUsageAwareLLMProvider{
		generateWithUsageFn: func(ctx context.Context, _, _ string) (string, domain.TokenUsage, error) {
			return "", domain.TokenUsage{}, WrapRetryable(errors.New("primary down"))
		},
	}
	secondary := &mockUsageAwareLLMProvider{
		generateWithUsageFn: func(ctx context.Context, _, _ string) (string, domain.TokenUsage, error) {
			return "secondary response", domain.TokenUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30}, nil
		},
	}

	fb, err := NewFallbackUsageAwareLLM([]domain.UsageAwareLLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	result, usage, err := fb.GenerateWithUsage(context.Background(), "system", "user")
	require.NoError(t, err)
	assert.Equal(t, "secondary response", result)
	assert.Equal(t, int64(30), usage.TotalTokens)
	assert.Equal(t, int64(10), usage.PromptTokens)
	assert.Equal(t, int64(20), usage.CompletionTokens)

	stats := fb.Stats()
	assert.Equal(t, int64(1), stats.FallbackCount)
}

type mockUsageAwareLLMProvider struct {
	generateFn          func(ctx context.Context, systemPrompt, userMessage string) (string, error)
	healthFn            func(ctx context.Context) error
	generateWithUsageFn func(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error)
	modelNameFn         func() string
}

func (m *mockUsageAwareLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return m.generateFn(ctx, systemPrompt, userMessage)
}

func (m *mockUsageAwareLLMProvider) Health(ctx context.Context) error {
	if m.healthFn != nil {
		return m.healthFn(ctx)
	}
	return nil
}

func (m *mockUsageAwareLLMProvider) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	return m.generateWithUsageFn(ctx, systemPrompt, userMessage)
}

func (m *mockUsageAwareLLMProvider) ModelName() string {
	if m.modelNameFn != nil {
		return m.modelNameFn()
	}
	return "mock-model"
}

// @sk-test graceful-degradation#T3.5: TestFallbackStreamingLLM_Stats (AC-008)
func TestFallbackStreamingLLM_Stats(t *testing.T) {
	callCount := 0
	primary := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			callCount++
			if callCount <= 2 {
				ch := make(chan string)
				close(ch)
				return ch, nil
			}
			ch := make(chan string, 1)
			ch <- "primary token"
			close(ch)
			return ch, nil
		},
	}
	secondary := &mockStreamingLLMProvider{
		generateStreamFn: func(ctx context.Context, _, _ string) (<-chan string, error) {
			ch := make(chan string, 1)
			ch <- "secondary token"
			close(ch)
			return ch, nil
		},
	}

	fb, err := NewFallbackStreamingLLM([]domain.StreamingLLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	drain := func() {
		stream, err := fb.GenerateStream(context.Background(), "system", "user")
		require.NoError(t, err)
		//nolint:revive // intentional drain of stream channel
		for range stream {
		}
	}

	drain()
	drain()
	drain()

	stats := fb.Stats()
	assert.Equal(t, int64(3), stats.TotalCalls)
	assert.Equal(t, int64(2), stats.FallbackCount)
}

// @sk-test graceful-degradation#T3.5: TestFallbackUsageAwareLLM_Stats (AC-008)
func TestFallbackUsageAwareLLM_Stats(t *testing.T) {
	primary := &mockUsageAwareLLMProvider{
		generateWithUsageFn: func(ctx context.Context, _, _ string) (string, domain.TokenUsage, error) {
			return "", domain.TokenUsage{}, WrapRetryable(errors.New("primary down"))
		},
		modelNameFn: func() string { return "primary" },
	}
	secondary := &mockUsageAwareLLMProvider{
		generateWithUsageFn: func(ctx context.Context, _, _ string) (string, domain.TokenUsage, error) {
			return "ok", domain.TokenUsage{TotalTokens: 5}, nil
		},
		modelNameFn: func() string { return "secondary" },
	}

	fb, err := NewFallbackUsageAwareLLM([]domain.UsageAwareLLMProvider{primary, secondary}, nil, nil)
	require.NoError(t, err)

	_, _, err = fb.GenerateWithUsage(context.Background(), "system", "user")
	require.NoError(t, err)

	stats := fb.Stats()
	assert.Equal(t, int64(1), stats.TotalCalls)
	assert.Equal(t, int64(1), stats.PrimaryFailures)
	assert.Equal(t, int64(1), stats.FallbackCount)
	assert.Equal(t, "secondary", fb.ModelName())
}
