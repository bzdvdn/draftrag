package draftrag

import (
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/resilience"
)

// CircuitState — состояние circuit breaker.
type CircuitState = resilience.CircuitState

const (
	// CircuitClosed — нормальная работа.
	CircuitClosed = resilience.CircuitClosed
	// CircuitOpen — блокировка запросов.
	CircuitOpen = resilience.CircuitOpen
	// CircuitHalfOpen — пробное восстановление.
	CircuitHalfOpen = resilience.CircuitHalfOpen
)

// CircuitBreakerStats — статистика circuit breaker.
type CircuitBreakerStats = resilience.Stats

// ErrCircuitOpen возвращается, когда circuit breaker в состоянии open.
var ErrCircuitOpen = resilience.ErrCircuitOpen

// RetryOptions объединяет настройки retry и circuit breaker.
// Нулевые значения используют безопасные defaults.
type RetryOptions struct {
	// MaxRetries — максимальное количество повторных попыток (0 → 3).
	MaxRetries int

	// BaseDelay — начальная задержка перед первым retry (0 → 100ms).
	BaseDelay time.Duration

	// MaxDelay — максимальная задержка (0 → 10s).
	MaxDelay time.Duration

	// Multiplier — множитель exponential backoff (0 → 2.0).
	Multiplier float64

	// JitterFactor — доля случайной составляющей (0 → 0.25).
	JitterFactor float64

	// CBThreshold — порог ошибок для перехода circuit breaker в open (0 → 5).
	CBThreshold int

	// CBTimeout — время восстановления circuit breaker (0 → 30s).
	CBTimeout time.Duration
}

func (o RetryOptions) toInternal() (*resilience.RetryConfig, *resilience.CircuitBreakerConfig) {
	rc := resilience.DefaultRetryConfig()
	if o.MaxRetries > 0 {
		rc.MaxRetries = o.MaxRetries
	}

	backoff := resilience.DefaultBackoff()
	if o.BaseDelay > 0 {
		backoff.BaseDelay = o.BaseDelay
	}
	if o.MaxDelay > 0 {
		backoff.MaxDelay = o.MaxDelay
	}
	if o.Multiplier > 0 {
		backoff.Multiplier = o.Multiplier
	}
	if o.JitterFactor > 0 {
		backoff.JitterFactor = o.JitterFactor
	}
	rc.Backoff = backoff

	cbc := resilience.DefaultCircuitBreakerConfig()
	if o.CBThreshold > 0 {
		cbc.Threshold = o.CBThreshold
	}
	if o.CBTimeout > 0 {
		cbc.Timeout = o.CBTimeout
	}

	return rc, cbc
}

// RetryEmbedder — обёртка для Embedder с retry и circuit breaker.
// Реализует Embedder. Дополнительно предоставляет CircuitBreakerState() и CircuitBreakerStats().
type RetryEmbedder struct {
	*resilience.RetryEmbedder
}

// NewRetryEmbedder оборачивает embedder с retry и circuit breaker.
// Нулевые поля RetryOptions используют defaults (MaxRetries=3, CBThreshold=5).
//
// Для использования в Pipeline передайте как Embedder:
//
//	re := draftrag.NewRetryEmbedder(embedder, draftrag.RetryOptions{})
//	pipeline := draftrag.NewPipeline(store, llm, re)
//
// Для доступа к состоянию circuit breaker используйте type assertion:
//
//	if re, ok := embedder.(*draftrag.RetryEmbedder); ok {
//	    fmt.Println(re.CircuitBreakerState())
//	}
func NewRetryEmbedder(e Embedder, opts RetryOptions) *RetryEmbedder {
	rc, cbc := opts.toInternal()
	return &RetryEmbedder{
		RetryEmbedder: resilience.NewRetryEmbedder(e, rc, cbc, nil),
	}
}

// RetryLLMProvider — обёртка для LLMProvider с retry и circuit breaker.
// Реализует LLMProvider. Дополнительно предоставляет CircuitBreakerState() и CircuitBreakerStats().
type RetryLLMProvider struct {
	*resilience.RetryLLMProvider
}

// NewRetryLLMProvider оборачивает LLM провайдер с retry и circuit breaker.
// Нулевые поля RetryOptions используют defaults (MaxRetries=3, CBThreshold=5).
//
// Для использования в Pipeline передайте как LLMProvider:
//
//	rl := draftrag.NewRetryLLMProvider(llm, draftrag.RetryOptions{})
//	pipeline := draftrag.NewPipeline(store, rl, embedder)
func NewRetryLLMProvider(l LLMProvider, opts RetryOptions) *RetryLLMProvider {
	rc, cbc := opts.toInternal()
	return &RetryLLMProvider{
		RetryLLMProvider: resilience.NewRetryLLMProvider(l, rc, cbc, nil),
	}
}

// IsRetryable проверяет, является ли ошибка retryable.
// Context cancellation ошибки всегда non-retryable.
// Ошибки без явного флага считаются retryable (безопасный default для transient errors).
var IsRetryable = resilience.IsRetryable

// WrapRetryable помечает ошибку как retryable.
var WrapRetryable = resilience.WrapRetryable

// WrapNonRetryable помечает ошибку как non-retryable (не будет повторяться).
var WrapNonRetryable = resilience.WrapNonRetryable
