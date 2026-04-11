// @ds-task T1.2: Exponential backoff с jitter (DEC-003)
// Этот файл определяет стратегию задержек между retry-попытками.

package resilience

import (
	"math"
	"math/rand"
	"time"
)

// Backoff определяет стратегию задержек между retry-попытками.
// Использует exponential backoff с jitter для предотвращения thundering herd.
type Backoff struct {
	// BaseDelay — начальная задержка перед первым retry.
	BaseDelay time.Duration

	// MaxDelay — максимальная задержка (предел exponential роста).
	MaxDelay time.Duration

	// Multiplier — множитель для exponential backoff (обычно 2).
	Multiplier float64

	// JitterFactor — доля jitter от задержки (0.25 = 25%).
	JitterFactor float64
}

// DefaultBackoff возвращает Backoff с разумными значениями по умолчанию:
// BaseDelay: 100ms, MaxDelay: 10s, Multiplier: 2, JitterFactor: 0.25.
func DefaultBackoff() *Backoff {
	return &Backoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.25,
	}
}

// CalculateDelay вычисляет задержку для указанной попытки.
// attempt номеруется с 0 (0 = первая retry после первоначальной ошибки).
// Формула: delay = min(baseDelay * multiplier^attempt, maxDelay) * (1 + jitter)
// где jitter = random(0, jitterFactor).
func (b *Backoff) CalculateDelay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Расчёт exponential задержки
	multiplier := math.Pow(b.Multiplier, float64(attempt))
	delay := float64(b.BaseDelay) * multiplier

	// Ограничиваем максимальной задержкой
	if delay > float64(b.MaxDelay) {
		delay = float64(b.MaxDelay)
	}

	// Добавляем jitter (случайную составляющую до JitterFactor)
	if b.JitterFactor > 0 {
		jitter := 1.0 + rand.Float64()*b.JitterFactor
		delay *= jitter
	}

	return time.Duration(delay)
}

// RetryConfig содержит настройки для retry-логики.
type RetryConfig struct {
	// MaxRetries — максимальное количество retry-попыток.
	// 0 означает "без retry" (только одна попытка).
	MaxRetries int

	// Backoff — стратегия задержек между попытками.
	// Если nil, используется DefaultBackoff().
	Backoff *Backoff
}

// DefaultRetryConfig возвращает RetryConfig с разумными значениями по умолчанию:
// MaxRetries: 3, Backoff: DefaultBackoff().
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		Backoff:    DefaultBackoff(),
	}
}

// GetBackoff возвращает backoff из конфигурации или дефолтный.
func (c *RetryConfig) GetBackoff() *Backoff {
	if c.Backoff == nil {
		return DefaultBackoff()
	}
	return c.Backoff
}
