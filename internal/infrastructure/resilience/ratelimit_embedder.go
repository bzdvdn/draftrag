package resilience

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task rate-limiting-llm#T2.1: TokenBucketEmbedder (AC-004, RQ-004)
type TokenBucketEmbedder struct {
	inner  domain.Embedder
	bucket *tokenBucket
	hooks  domain.Hooks
}

// NewTokenBucketEmbedder создаёт декоратор Embedder с token bucket rate limiter.
// При TokensPerSecond <= 0 возвращает passthrough-провайдер без rate limiting.
func NewTokenBucketEmbedder(inner domain.Embedder, rate, burst float64, hooks domain.Hooks) *TokenBucketEmbedder {
	if rate <= 0 {
		return &TokenBucketEmbedder{inner: inner, hooks: hooks}
	}
	if burst < 1 {
		burst = rate
	}
	return &TokenBucketEmbedder{
		inner:  inner,
		bucket: newTokenBucket(rate, burst),
		hooks:  hooks,
	}
}

// @sk-task rate-limiting-llm#T2.1: Embed с rate limiting (AC-004)
func (p *TokenBucketEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if p.bucket != nil {
		_, err := p.bucket.Take(ctx, 1)
		if err != nil {
			return nil, err
		}
	}
	return p.inner.Embed(ctx, text)
}

// Health делегирует проверку здоровья внутреннему эмбеддеру.
func (p *TokenBucketEmbedder) Health(ctx context.Context) error {
	return p.inner.Health(ctx)
}
