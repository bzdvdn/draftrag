// @ds-task T2.3: RetryLLMProvider с retry loop, backoff, CB integration (AC-002, AC-005)
// @ds-task T2.4: Интеграция с Hooks для событий retry и CB transitions (AC-006, RQ-008)
// Этот файл реализует обёртку для LLMProvider с retry и circuit breaker.

package resilience

import (
	"context"
	"fmt"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// RetryLLMProvider — обёртка для LLMProvider с retry-логикой и circuit breaker.
// Реализует интерфейс domain.LLMProvider.
type RetryLLMProvider struct {
	// llm — базовый LLM провайдер.
	llm domain.LLMProvider

	// retryConfig — настройки retry.
	retryConfig *RetryConfig

	// circuitBreaker — circuit breaker для защиты от каскадных отказов.
	circuitBreaker *CircuitBreaker

	// hooks — опциональный интерфейс для observability.
	hooks domain.Hooks

	// logger — опциональный структурированный логгер.
	logger domain.Logger
}

// NewRetryLLMProvider создаёт обёртку для LLM провайдера.
// Если retryConfig == nil, используется DefaultRetryConfig().
// Если cbConfig == nil, используется DefaultCircuitBreakerConfig().
func NewRetryLLMProvider(
	llm domain.LLMProvider,
	retryConfig *RetryConfig,
	cbConfig *CircuitBreakerConfig,
	hooks domain.Hooks,
	logger domain.Logger,
) *RetryLLMProvider {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	return &RetryLLMProvider{
		llm:            llm,
		retryConfig:    retryConfig,
		circuitBreaker: NewCircuitBreaker(cbConfig),
		hooks:          hooks,
		logger:         logger,
	}
}

// Generate реализует domain.LLMProvider.
// Выполняет retry при ошибках с exponential backoff и уважает circuit breaker.
func (r *RetryLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	// Проверяем circuit breaker
	if err := r.circuitBreaker.CanExecute(); err != nil {
		domain.SafeLog(r.logger, ctx, domain.LogLevelWarn, "circuit breaker rejected",
			domain.LogField{Key: "component", Value: "resilience_retry"},
			domain.LogField{Key: "operation", Value: "generate"},
			domain.LogField{Key: "rejected", Value: true},
			domain.LogField{Key: "err", Value: err},
		)
		r.recordEvent(ctx, "generate", 0, err, true)
		return "", fmt.Errorf("circuit breaker: %w", err)
	}

	backoff := r.retryConfig.GetBackoff()
	maxRetries := r.retryConfig.MaxRetries

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Проверяем context cancellation перед каждой попыткой
		if err := ctx.Err(); err != nil {
			r.circuitBreaker.RecordFailure()
			r.recordEvent(ctx, "generate", attempt, err, false)
			return "", err
		}

		// Выполняем запрос
		result, err := r.llm.Generate(ctx, systemPrompt, userMessage)
		if err == nil {
			// Успех
			r.circuitBreaker.RecordSuccess()
			r.recordEvent(ctx, "generate", attempt, nil, false)
			return result, nil
		}

		lastErr = err

		// Проверяем, стоит ли повторять
		if !IsRetryable(err) {
			domain.SafeLog(r.logger, ctx, domain.LogLevelWarn, "non-retryable error",
				domain.LogField{Key: "component", Value: "resilience_retry"},
				domain.LogField{Key: "operation", Value: "generate"},
				domain.LogField{Key: "attempt", Value: attempt},
				domain.LogField{Key: "rejected", Value: false},
				domain.LogField{Key: "err", Value: err},
			)
			r.circuitBreaker.RecordFailure()
			r.recordEvent(ctx, "generate", attempt, err, false)
			return "", err
		}

		// Фиксируем ошибку в circuit breaker и вызываем hooks (кроме последней попытки)
		if attempt < maxRetries {
			domain.SafeLog(r.logger, ctx, domain.LogLevelWarn, "retry attempt failed",
				domain.LogField{Key: "component", Value: "resilience_retry"},
				domain.LogField{Key: "operation", Value: "generate"},
				domain.LogField{Key: "attempt", Value: attempt},
				domain.LogField{Key: "rejected", Value: false},
				domain.LogField{Key: "err", Value: err},
			)
			r.circuitBreaker.RecordFailure()
			r.recordEvent(ctx, "generate", attempt, err, false)
		}

		// Ждём перед следующей попыткой (если это не последняя)
		if attempt < maxRetries {
			delay := backoff.CalculateDelay(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				r.recordEvent(ctx, "generate", attempt+1, ctx.Err(), false)
				return "", ctx.Err()
			case <-timer.C:
				// Продолжаем со следующей попыткой
			}
		}
	}

	// Все попытки исчерпаны
	r.recordEvent(ctx, "generate", maxRetries, lastErr, false)
	return "", lastErr
}

// recordEvent фиксирует событие через hooks.
func (r *RetryLLMProvider) recordEvent(ctx context.Context, operation string, attempt int, err error, rejected bool) {
	if r.hooks == nil {
		return
	}

	stage := domain.HookStage(domain.HookStageGenerate)
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
func (r *RetryLLMProvider) CircuitBreakerState() CircuitState {
	return r.circuitBreaker.State()
}

// CircuitBreakerStats возвращает статистику circuit breaker.
func (r *RetryLLMProvider) CircuitBreakerStats() Stats {
	return r.circuitBreaker.GetStats()
}
