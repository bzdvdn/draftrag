package resilience

import (
	"context"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task rate-limiting-llm#T1.1: TokenBucketLLMProvider (AC-001, AC-002, AC-003, RQ-001, RQ-002)
type TokenBucketLLMProvider struct {
	inner  domain.LLMProvider
	bucket *tokenBucket
	hooks  domain.Hooks
}

// NewTokenBucketLLMProvider создаёт декоратор LLMProvider с token bucket rate limiter.
// При TokensPerSecond <= 0 возвращает passthrough-провайдер без rate limiting.
func NewTokenBucketLLMProvider(inner domain.LLMProvider, rate, burst float64, hooks domain.Hooks) *TokenBucketLLMProvider {
	if rate <= 0 {
		return &TokenBucketLLMProvider{inner: inner, hooks: hooks}
	}
	if burst < 1 {
		burst = rate
	}
	return &TokenBucketLLMProvider{
		inner:  inner,
		bucket: newTokenBucket(rate, burst),
		hooks:  hooks,
	}
}

// @sk-task rate-limiting-llm#T1.1: Generate с rate limiting (AC-001, AC-002)
// @sk-task rate-limiting-llm#T3.1: Hooks logging (AC-005, RQ-007)
func (p *TokenBucketLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if p.bucket != nil {
		waited, err := p.bucket.Take(ctx, 1)
		if waited && p.hooks != nil {
			ctx2 := p.hooks.StageStart(ctx, domain.StageStartEvent{
				Operation: "rate_limit_wait",
				Stage:     domain.HookStageRateLimit,
			})
			p.hooks.StageEnd(ctx2, domain.StageEndEvent{
				Operation: "rate_limit_wait",
				Stage:     domain.HookStageRateLimit,
				Duration:  0,
				Err:       err,
			})
		}
		if err != nil {
			return "", err
		}
	}
	return p.inner.Generate(ctx, systemPrompt, userMessage)
}

// @sk-task rate-limiting-llm#T1.1: Health passthrough (RQ-003)
func (p *TokenBucketLLMProvider) Health(ctx context.Context) error {
	return p.inner.Health(ctx)
}

// TokensPerSecond возвращает настроенную скорость, или 0 для passthrough.
func (p *TokenBucketLLMProvider) TokensPerSecond() float64 {
	if p.bucket == nil {
		return 0
	}
	return float64(time.Second) / float64(p.bucket.interval)
}
