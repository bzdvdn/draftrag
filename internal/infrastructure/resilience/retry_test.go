// @sk-task T3.1: Unit-tests для Backoff (DEC-003)
// Проверка exponential backoff с jitter.

package resilience

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultBackoff(t *testing.T) {
	b := DefaultBackoff()

	assert.Equal(t, 100*time.Millisecond, b.BaseDelay)
	assert.Equal(t, 10*time.Second, b.MaxDelay)
	assert.Equal(t, 2.0, b.Multiplier)
	assert.Equal(t, 0.25, b.JitterFactor)
}

func TestBackoff_CalculateDelay_Exponential(t *testing.T) {
	b := &Backoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0, // без jitter для точной проверки
	}

	// attempt 0: 100ms * 2^0 = 100ms
	delay := b.CalculateDelay(0)
	assert.Equal(t, 100*time.Millisecond, delay)

	// attempt 1: 100ms * 2^1 = 200ms
	delay = b.CalculateDelay(1)
	assert.Equal(t, 200*time.Millisecond, delay)

	// attempt 2: 100ms * 2^2 = 400ms
	delay = b.CalculateDelay(2)
	assert.Equal(t, 400*time.Millisecond, delay)

	// attempt 3: 100ms * 2^3 = 800ms
	delay = b.CalculateDelay(3)
	assert.Equal(t, 800*time.Millisecond, delay)
}

func TestBackoff_CalculateDelay_MaxDelay(t *testing.T) {
	b := &Backoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	// attempt 0: 100ms (не достигли max)
	delay := b.CalculateDelay(0)
	assert.Equal(t, 100*time.Millisecond, delay)

	// attempt 1: 200ms (не достигли max)
	delay = b.CalculateDelay(1)
	assert.Equal(t, 200*time.Millisecond, delay)

	// attempt 2: 400ms (не достигли max)
	delay = b.CalculateDelay(2)
	assert.Equal(t, 400*time.Millisecond, delay)

	// attempt 3: 800ms, но max = 500ms
	delay = b.CalculateDelay(3)
	assert.Equal(t, 500*time.Millisecond, delay)

	// attempt 4: всё ещё max = 500ms
	delay = b.CalculateDelay(4)
	assert.Equal(t, 500*time.Millisecond, delay)
}

func TestBackoff_CalculateDelay_Jitter(t *testing.T) {
	b := &Backoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.25,
	}

	// Выполняем множество измерений для проверки диапазона
	for i := 0; i < 100; i++ {
		delay := b.CalculateDelay(0)
		// Базовая задержка 100ms, jitter до 25%
		// delay должен быть в диапазоне [100ms, 125ms]
		minDelay := 100 * time.Millisecond
		maxDelay := 125 * time.Millisecond

		assert.GreaterOrEqual(t, delay, minDelay, "delay should be >= base delay")
		assert.LessOrEqual(t, delay, maxDelay, "delay should be <= base delay * (1 + jitter)")
	}
}

func TestBackoff_CalculateDelay_JitterRange(t *testing.T) {
	b := &Backoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.25,
	}

	// Проверяем разные attempt-ы
	for attempt := 0; attempt < 5; attempt++ {
		baseDelay := float64(b.BaseDelay) * math.Pow(b.Multiplier, float64(attempt))
		if baseDelay > float64(b.MaxDelay) {
			baseDelay = float64(b.MaxDelay)
		}

		minExpected := time.Duration(baseDelay)
		maxExpected := time.Duration(baseDelay * 1.25)

		// Множественные измерения для статистической значимости
		for i := 0; i < 50; i++ {
			delay := b.CalculateDelay(attempt)
			assert.GreaterOrEqual(t, delay, minExpected,
				"attempt %d: delay %v should be >= %v", attempt, delay, minExpected)
			assert.LessOrEqual(t, delay, maxExpected,
				"attempt %d: delay %v should be <= %v", attempt, delay, maxExpected)
		}
	}
}

func TestBackoff_CalculateDelay_NegativeAttempt(t *testing.T) {
	b := DefaultBackoff()

	// Негативный attempt должен быть приведён к 0
	delay := b.CalculateDelay(-1)
	assert.GreaterOrEqual(t, delay, b.BaseDelay)
}

func TestDefaultRetryConfig(t *testing.T) {
	c := DefaultRetryConfig()

	assert.Equal(t, 3, c.MaxRetries)
	assert.NotNil(t, c.Backoff)
}

func TestRetryConfig_GetBackoff(t *testing.T) {
	// Когда Backoff не nil — возвращаем его
	customBackoff := &Backoff{BaseDelay: 1 * time.Second}
	c := &RetryConfig{Backoff: customBackoff}
	assert.Equal(t, customBackoff, c.GetBackoff())

	// Когда Backoff nil — возвращаем дефолтный
	c2 := &RetryConfig{Backoff: nil}
	defaultB := DefaultBackoff()
	assert.Equal(t, defaultB.BaseDelay, c2.GetBackoff().BaseDelay)
}
