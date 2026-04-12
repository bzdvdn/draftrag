// @sk-task T3.3: Unit-tests для RetryEmbedder (AC-001, AC-005)
// @sk-task T3.5: Unit-tests для hooks интеграции (AC-006)
// Проверка retry успеха, исчерпания попыток, context cancellation.

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

// MockEmbedder — mock реализация domain.Embedder.
type MockEmbedder struct {
	mock.Mock
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float64), args.Error(1)
}

// MockHooks — mock реализация domain.Hooks.
type MockHooks struct {
	mock.Mock
}

func (m *MockHooks) StageStart(ctx context.Context, ev domain.StageStartEvent) {
	m.Called(ctx, ev)
}

func (m *MockHooks) StageEnd(ctx context.Context, ev domain.StageEndEvent) {
	m.Called(ctx, ev)
}

func TestRetryEmbedder_Success(t *testing.T) {
	mockEmb := new(MockEmbedder)
	mockEmb.On("Embed", mock.Anything, "test text").Return([]float64{0.1, 0.2}, nil).Once()

	retryEmb := NewRetryEmbedder(mockEmb, nil, nil, nil, nil)

	result, err := retryEmb.Embed(context.Background(), "test text")

	assert.NoError(t, err)
	assert.Equal(t, []float64{0.1, 0.2}, result)
	mockEmb.AssertExpectations(t)
}

func TestRetryEmbedder_RetrySuccess(t *testing.T) {
	// AC-001: RetryEmbedder успешный retry
	mockEmb := new(MockEmbedder)

	// Первая попытка — ошибка, вторая — успех
	mockEmb.On("Embed", mock.Anything, "test text").
		Return(nil, errors.New("transient error")).Once()
	mockEmb.On("Embed", mock.Anything, "test text").
		Return([]float64{0.1, 0.2}, nil).Once()

	config := &RetryConfig{
		MaxRetries: 3,
		Backoff: &Backoff{
			BaseDelay:    10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
	}
	retryEmb := NewRetryEmbedder(mockEmb, config, nil, nil, nil)

	result, err := retryEmb.Embed(context.Background(), "test text")

	assert.NoError(t, err)
	assert.Equal(t, []float64{0.1, 0.2}, result)
	// Должно быть 2 вызова базового embedder
	mockEmb.AssertNumberOfCalls(t, "Embed", 2)
}

func TestRetryEmbedder_MaxRetriesExceeded(t *testing.T) {
	// AC-001 частично: после исчерпания попыток возвращается ошибка
	mockEmb := new(MockEmbedder)

	// Все попытки возвращают ошибку
	mockEmb.On("Embed", mock.Anything, "test text").
		Return(nil, errors.New("persistent error")).Times(4) // initial + 3 retries

	config := &RetryConfig{
		MaxRetries: 3,
		Backoff: &Backoff{
			BaseDelay:    1 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			Multiplier:   2.0,
			JitterFactor: 0,
		},
	}
	retryEmb := NewRetryEmbedder(mockEmb, config, nil, nil, nil)

	result, err := retryEmb.Embed(context.Background(), "test text")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "persistent error")
	mockEmb.AssertNumberOfCalls(t, "Embed", 4)
}

func TestRetryEmbedder_ContextCancellation(t *testing.T) {
	// AC-005: Context cancellation прерывает retry
	mockEmb := new(MockEmbedder)
	mockEmb.On("Embed", mock.Anything, "test text").
		Return(nil, errors.New("transient error")).Once()

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
	retryEmb := NewRetryEmbedder(mockEmb, config, nil, nil, nil)

	// Отменяем контекст через 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := retryEmb.Embed(ctx, "test text")
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Nil(t, result)
	// Должно завершиться быстро (менее backoff delay)
	assert.Less(t, elapsed, 500*time.Millisecond)
	// Должно быть только 1 вызов (первая попытка)
	mockEmb.AssertNumberOfCalls(t, "Embed", 1)
}

func TestRetryEmbedder_NonRetryableError(t *testing.T) {
	mockEmb := new(MockEmbedder)
	// Возвращаем ошибку, помеченную как non-retryable
	mockEmb.On("Embed", mock.Anything, "test text").
		Return(nil, WrapNonRetryable(errors.New("permanent error"))).Once()

	config := &RetryConfig{
		MaxRetries: 3,
	}
	retryEmb := NewRetryEmbedder(mockEmb, config, nil, nil, nil)

	result, err := retryEmb.Embed(context.Background(), "test text")

	assert.Error(t, err)
	assert.Nil(t, result)
	// Не должно быть retry — сразу возвращаем ошибку
	mockEmb.AssertNumberOfCalls(t, "Embed", 1)
}

func TestRetryEmbedder_CircuitBreakerBlocks(t *testing.T) {
	// Проверяем, что circuit breaker блокирует запросы
	mockEmb := new(MockEmbedder)
	// Embed никогда не вызовется из-за circuit breaker

	cbConfig := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   10 * time.Second, // Долгий timeout
	}
	config := &RetryConfig{
		MaxRetries: 0, // Без retry
	}
	retryEmb := NewRetryEmbedder(mockEmb, config, cbConfig, nil, nil)

	// Переводим circuit breaker в open
	retryEmb.circuitBreaker.RecordFailure()

	result, err := retryEmb.Embed(context.Background(), "test text")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")
	assert.Nil(t, result)
	// Embed не должен быть вызван
	mockEmb.AssertNumberOfCalls(t, "Embed", 0)
}

func TestRetryEmbedder_HooksCalled(t *testing.T) {
	// AC-006: Hooks получают события о retry attempts
	mockEmb := new(MockEmbedder)
	mockEmb.On("Embed", mock.Anything, "test text").
		Return(nil, errors.New("transient error")).Once()
	mockEmb.On("Embed", mock.Anything, "test text").
		Return([]float64{0.1, 0.2}, nil).Once()

	mockHooks := new(MockHooks)
	// Ожидаем вызовы hooks: start и end для каждой попытки
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
	retryEmb := NewRetryEmbedder(mockEmb, config, nil, mockHooks, nil)

	retryEmb.Embed(context.Background(), "test text")

	// Должно быть по 2 вызова StageStart и StageEnd (по одному на попытку)
	mockHooks.AssertNumberOfCalls(t, "StageStart", 2)
	mockHooks.AssertNumberOfCalls(t, "StageEnd", 2)
}

func TestRetryEmbedder_HooksCalledOnRejection(t *testing.T) {
	// AC-006: Hooks получают событие при отклонении circuit breaker
	mockEmb := new(MockEmbedder)
	mockHooks := new(MockHooks)
	mockHooks.On("StageStart", mock.Anything, mock.Anything).Return()
	mockHooks.On("StageEnd", mock.Anything, mock.Anything).Return()

	cbConfig := &CircuitBreakerConfig{
		Threshold: 1,
		Timeout:   10 * time.Second,
	}
	retryEmb := NewRetryEmbedder(mockEmb, nil, cbConfig, mockHooks, nil)

	// Переводим в open
	retryEmb.circuitBreaker.RecordFailure()

	retryEmb.Embed(context.Background(), "test text")

	// Должен быть вызов hooks для rejected запроса
	mockHooks.AssertNumberOfCalls(t, "StageStart", 1)
	mockHooks.AssertNumberOfCalls(t, "StageEnd", 1)
}

func TestRetryEmbedder_CircuitBreakerState(t *testing.T) {
	mockEmb := new(MockEmbedder)
	retryEmb := NewRetryEmbedder(mockEmb, nil, nil, nil, nil)

	assert.Equal(t, CircuitClosed, retryEmb.CircuitBreakerState())

	stats := retryEmb.CircuitBreakerStats()
	assert.Equal(t, CircuitClosed, stats.State)
	assert.Equal(t, 0, stats.FailureCount)
}
