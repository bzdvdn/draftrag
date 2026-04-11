// @sk-task T3.4: Unit-tests для RetryLLMProvider (AC-002, AC-005)
// @sk-task T3.5: Unit-tests для hooks интеграции (AC-006)
// Проверка retry исчерпания, context cancellation.

package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMProvider — mock реализация domain.LLMProvider.
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	args := m.Called(ctx, systemPrompt, userMessage)
	return args.String(0), args.Error(1)
}

func TestRetryLLMProvider_Success(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("response", nil).Once()

	retryLLM := NewRetryLLMProvider(mockLLM, nil, nil, nil)

	result, err := retryLLM.Generate(context.Background(), "system", "user")

	assert.NoError(t, err)
	assert.Equal(t, "response", result)
	mockLLM.AssertExpectations(t)
}

func TestRetryLLMProvider_RetryExhausted(t *testing.T) {
	// AC-002: RetryLLMProvider исчерпание попыток
	mockLLM := new(MockLLMProvider)

	// Все попытки возвращают ошибку
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("", errors.New("API error")).Times(4) // initial + 3 retries

	config := &RetryConfig{
		MaxRetries: 3,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
	}
	retryLLM := NewRetryLLMProvider(mockLLM, config, nil, nil)

	result, err := retryLLM.Generate(context.Background(), "system", "user")

	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "API error")
	// Ровно 4 вызова (1 + 3 retry)
	mockLLM.AssertNumberOfCalls(t, "Generate", 4)
}

func TestRetryLLMProvider_ContextCancellation(t *testing.T) {
	// AC-005: Context cancellation прерывает retry
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("", errors.New("transient error")).Once()

	ctx, cancel := context.WithCancel(context.Background())

	config := &RetryConfig{
		MaxRetries: 5,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Second, // Большая задержка
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
	}
	retryLLM := NewRetryLLMProvider(mockLLM, config, nil, nil)

	// Отменяем контекст через 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := retryLLM.Generate(ctx, "system", "user")
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, "", result)
	// Должно завершиться быстро
	assert.Less(t, elapsed, 500*time.Millisecond)
	// Только 1 вызов
	mockLLM.AssertNumberOfCalls(t, "Generate", 1)
}

func TestRetryLLMProvider_NonRetryableError(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	// Возвращаем ошибку, помеченную как non-retryable
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("", WrapNonRetryable(errors.New("invalid request"))).Once()

	config := &RetryConfig{
		MaxRetries: 3,
	}
	retryLLM := NewRetryLLMProvider(mockLLM, config, nil, nil)

	result, err := retryLLM.Generate(context.Background(), "system", "user")

	assert.Error(t, err)
	assert.Equal(t, "", result)
	// Не должно быть retry
	mockLLM.AssertNumberOfCalls(t, "Generate", 1)
}

func TestRetryLLMProvider_CircuitBreakerBlocks(t *testing.T) {
	mockLLM := new(MockLLMProvider)

	cbConfig := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   10 * time.Second,
	}
	config := &RetryConfig{
		MaxRetries: 0,
	}
	retryLLM := NewRetryLLMProvider(mockLLM, config, cbConfig, nil)

	// Переводим circuit breaker в open
	retryLLM.circuitBreaker.RecordFailure()

	result, err := retryLLM.Generate(context.Background(), "system", "user")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")
	assert.Equal(t, "", result)
	// Generate не должен быть вызван
	mockLLM.AssertNumberOfCalls(t, "Generate", 0)
}

func TestRetryLLMProvider_HooksCalled(t *testing.T) {
	// AC-006: Hooks получают события о retry attempts
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("", errors.New("transient error")).Once()
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("response", nil).Once()

	mockHooks := new(MockHooks)
	mockHooks.On("StageStart", mock.Anything, mock.Anything).Return()
	mockHooks.On("StageEnd", mock.Anything, mock.Anything).Return()

	config := &RetryConfig{
		MaxRetries: 3,
		Backoff: &Backoff{
			BaseDelay:    10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
	}
	retryLLM := NewRetryLLMProvider(mockLLM, config, nil, mockHooks)

	retryLLM.Generate(context.Background(), "system", "user")

	// По 2 вызова на каждую попытку
	mockHooks.AssertNumberOfCalls(t, "StageStart", 2)
	mockHooks.AssertNumberOfCalls(t, "StageEnd", 2)
}

func TestRetryLLMProvider_HooksCalledOnRejection(t *testing.T) {
	// AC-006: Hooks получают событие при отклонении circuit breaker
	mockLLM := new(MockLLMProvider)
	mockHooks := new(MockHooks)
	mockHooks.On("StageStart", mock.Anything, mock.Anything).Return()
	mockHooks.On("StageEnd", mock.Anything, mock.Anything).Return()

	cbConfig := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   10 * time.Second,
	}
	retryLLM := NewRetryLLMProvider(mockLLM, nil, cbConfig, mockHooks)

	// Переводим в open
	retryLLM.circuitBreaker.RecordFailure()

	retryLLM.Generate(context.Background(), "system", "user")

	// Должен быть вызов hooks для rejected запроса
	mockHooks.AssertNumberOfCalls(t, "StageStart", 1)
	mockHooks.AssertNumberOfCalls(t, "StageEnd", 1)
}

func TestRetryLLMProvider_CircuitBreakerState(t *testing.T) {
	mockLLM := new(MockLLMProvider)
	retryLLM := NewRetryLLMProvider(mockLLM, nil, nil, nil)

	assert.Equal(t, CircuitClosed, retryLLM.CircuitBreakerState())

	stats := retryLLM.CircuitBreakerStats()
	assert.Equal(t, CircuitClosed, stats.State)
	assert.Equal(t, 0, stats.FailureCount)
}

func TestRetryLLMProvider_RetryThenSuccess(t *testing.T) {
	mockLLM := new(MockLLMProvider)

	// Первая попытка — ошибка, вторая — успех
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("", errors.New("transient")).Once()
	mockLLM.On("Generate", mock.Anything, "system", "user").
		Return("success", nil).Once()

	config := &RetryConfig{
		MaxRetries: 3,
		Backoff: &Backoff{
			BaseDelay:    10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
	}
	retryLLM := NewRetryLLMProvider(mockLLM, config, nil, nil)

	result, err := retryLLM.Generate(context.Background(), "system", "user")

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	// 2 вызова (1 ошибка + 1 успех)
	mockLLM.AssertNumberOfCalls(t, "Generate", 2)
}
