// @sk-task prod-issues#T1.3: TokenBucketStreamingLLMProvider (AC-012, AC-013, RQ-029)

package resilience

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// TokenBucketStreamingLLMProvider — декоратор StreamingLLMProvider с token bucket rate limiter.
// Применяет rate limiting к Generate и GenerateStream.
type TokenBucketStreamingLLMProvider struct {
	inner  domain.StreamingLLMProvider
	bucket *tokenBucket
	hooks  domain.Hooks
}

// NewTokenBucketStreamingLLMProvider создаёт rate-limited обёртку для StreamingLLMProvider.
func NewTokenBucketStreamingLLMProvider(inner domain.StreamingLLMProvider, rate, burst float64, hooks domain.Hooks) *TokenBucketStreamingLLMProvider {
	if rate <= 0 {
		return &TokenBucketStreamingLLMProvider{inner: inner, hooks: hooks}
	}
	if burst < 1 {
		burst = rate
	}
	return &TokenBucketStreamingLLMProvider{
		inner:  inner,
		bucket: newTokenBucket(rate, burst),
		hooks:  hooks,
	}
}

// Generate применяет rate limit и делегирует внутреннему LLM.
func (p *TokenBucketStreamingLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if p.bucket != nil {
		waited, err := p.bucket.Take(ctx, 1)
		if waited && p.hooks != nil {
			p.hooks.StageEnd(p.hooks.StageStart(ctx, domain.StageStartEvent{
				Operation: "rate_limit_wait",
				Stage:     domain.HookStageRateLimit,
			}), domain.StageEndEvent{
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

// GenerateStream применяет rate limit и делегирует streaming внутреннему LLM.
func (p *TokenBucketStreamingLLMProvider) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if p.bucket != nil {
		waited, err := p.bucket.Take(ctx, 1)
		if waited && p.hooks != nil {
			p.hooks.StageEnd(p.hooks.StageStart(ctx, domain.StageStartEvent{
				Operation: "rate_limit_wait",
				Stage:     domain.HookStageRateLimit,
			}), domain.StageEndEvent{
				Operation: "rate_limit_wait",
				Stage:     domain.HookStageRateLimit,
				Duration:  0,
				Err:       err,
			})
		}
		if err != nil {
			return nil, err
		}
	}
	return p.inner.GenerateStream(ctx, systemPrompt, userMessage)
}

// Health делегирует проверку здоровья внутреннему LLM.
func (p *TokenBucketStreamingLLMProvider) Health(ctx context.Context) error {
	return p.inner.Health(ctx)
}

// TokensPerSecond возвращает настроенную скорость rate limiter'а.
func (p *TokenBucketStreamingLLMProvider) TokensPerSecond() float64 {
	if p.bucket == nil {
		return 0
	}
	return float64(1) / p.bucket.interval.Seconds()
}
