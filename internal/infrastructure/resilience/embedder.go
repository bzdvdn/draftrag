// @ds-task T2.2: RetryEmbedder с retry loop, backoff, CB integration (AC-001, AC-005)
// @ds-task T2.4: Интеграция с Hooks для событий retry и CB transitions (AC-006, RQ-008)
// Этот файл реализует обёртку для Embedder с retry и circuit breaker.

package resilience

import (
	"context"
	"fmt"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// RetryEmbedder — обёртка для Embedder с retry-логикой и circuit breaker.
// Реализует интерфейс domain.Embedder.
type RetryEmbedder struct {
	// embedder — базовый embedder.
	embedder domain.Embedder

	// retryConfig — настройки retry.
	retryConfig *RetryConfig

	// circuitBreaker — circuit breaker для защиты от каскадных отказов.
	circuitBreaker *CircuitBreaker

	// hooks — опциональный интерфейс для observability.
	hooks domain.Hooks
}

// NewRetryEmbedder создаёт обёртку для embedder.
// Если retryConfig == nil, используется DefaultRetryConfig().
// Если cbConfig == nil, используется DefaultCircuitBreakerConfig().
func NewRetryEmbedder(
	embedder domain.Embedder,
	retryConfig *RetryConfig,
	cbConfig *CircuitBreakerConfig,
	hooks domain.Hooks,
) *RetryEmbedder {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	return &RetryEmbedder{
		embedder:       embedder,
		retryConfig:    retryConfig,
		circuitBreaker: NewCircuitBreaker(cbConfig),
		hooks:          hooks,
	}
}

// Embed реализует domain.Embedder.
// Выполняет retry при ошибках с exponential backoff и уважает circuit breaker.
func (r *RetryEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	// Проверяем circuit breaker
	if err := r.circuitBreaker.CanExecute(); err != nil {
		r.recordEvent(ctx, "embed", 0, err, true)
		return nil, fmt.Errorf("circuit breaker: %w", err)
	}

	backoff := r.retryConfig.GetBackoff()
	maxRetries := r.retryConfig.MaxRetries

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Проверяем context cancellation перед каждой попыткой
		if err := ctx.Err(); err != nil {
			r.circuitBreaker.RecordFailure()
			r.recordEvent(ctx, "embed", attempt, err, false)
			return nil, err
		}

		// Выполняем запрос
		result, err := r.embedder.Embed(ctx, text)
		if err == nil {
			// Успех
			r.circuitBreaker.RecordSuccess()
			r.recordEvent(ctx, "embed", attempt, nil, false)
			return result, nil
		}

		lastErr = err

		// Проверяем, стоит ли повторять
		if !IsRetryable(err) {
			r.circuitBreaker.RecordFailure()
			r.recordEvent(ctx, "embed", attempt, err, false)
			return nil, err
		}

		// Фиксируем ошибку в circuit breaker и вызываем hooks (кроме последней попытки)
		if attempt < maxRetries {
			r.circuitBreaker.RecordFailure()
			r.recordEvent(ctx, "embed", attempt, err, false)
		}

		// Ждём перед следующей попыткой (если это не последняя)
		if attempt < maxRetries {
			delay := backoff.CalculateDelay(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				r.recordEvent(ctx, "embed", attempt+1, ctx.Err(), false)
				return nil, ctx.Err()
			case <-timer.C:
				// Продолжаем со следующей попыткой
			}
		}
	}

	// Все попытки исчерпаны
	r.recordEvent(ctx, "embed", maxRetries, lastErr, false)
	return nil, lastErr
}

// recordEvent фиксирует событие через hooks.
func (r *RetryEmbedder) recordEvent(ctx context.Context, operation string, attempt int, err error, rejected bool) {
	if r.hooks == nil {
		return
	}

	stage := domain.HookStage(domain.HookStageEmbed)
	ev := domain.StageStartEvent{
		Operation: fmt.Sprintf("%s:attempt=%d", operation, attempt),
		Stage:     stage,
	}

	if rejected {
		ev.Operation = fmt.Sprintf("%s:rejected", operation)
	}

	r.hooks.StageStart(ctx, ev)

	// Для завершения используем нулевую длительность (event-based hooks)
	r.hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: ev.Operation,
		Stage:     stage,
		Duration:  0,
		Err:       err,
	})
}

// CircuitBreakerState возвращает текущее состояние circuit breaker.
func (r *RetryEmbedder) CircuitBreakerState() CircuitState {
	return r.circuitBreaker.State()
}

// CircuitBreakerStats возвращает статистику circuit breaker.
func (r *RetryEmbedder) CircuitBreakerStats() Stats {
	return r.circuitBreaker.GetStats()
}
