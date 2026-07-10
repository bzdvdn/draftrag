package draftrag

import (
	"fmt"

	"github.com/bzdvdn/draftrag/internal/infrastructure/resilience"
)

// TokenBucketOptions содержит настройки token bucket rate limiter.
// Нулевые значения (TokensPerSecond=0) отключают rate limiting.
type TokenBucketOptions struct {
	// TokensPerSecond — максимальное количество запросов в секунду.
	// 0 отключает rate limiting.
	TokensPerSecond float64

	// BurstSize — максимальный burst (пиковый размер очереди токенов).
	// Если 0, используется значение TokensPerSecond.
	BurstSize float64
}

// @sk-task rate-limiting-llm#T1.2: NewTokenBucketLLMProvider (AC-003, RQ-006)
func NewTokenBucketLLMProvider(llm LLMProvider, opts TokenBucketOptions) (LLMProvider, error) {
	if opts.TokensPerSecond < 0 || opts.BurstSize < 0 {
		return nil, fmt.Errorf("token bucket: rate and burst must be non-negative, got rate=%v burst=%v",
			opts.TokensPerSecond, opts.BurstSize)
	}
	return resilience.NewTokenBucketLLMProvider(llm, opts.TokensPerSecond, opts.BurstSize, nil), nil
}

// @sk-task rate-limiting-llm#T2.2: NewTokenBucketEmbedder (AC-004, RQ-004)
func NewTokenBucketEmbedder(emb Embedder, opts TokenBucketOptions) (Embedder, error) {
	if opts.TokensPerSecond < 0 || opts.BurstSize < 0 {
		return nil, fmt.Errorf("token bucket: rate and burst must be non-negative, got rate=%v burst=%v",
			opts.TokensPerSecond, opts.BurstSize)
	}
	return resilience.NewTokenBucketEmbedder(emb, opts.TokensPerSecond, opts.BurstSize, nil), nil
}
