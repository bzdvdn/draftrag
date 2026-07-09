package resilience

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task graceful-degradation#T3.1: FallbackStreamingLLMProvider (RQ-009, AC-006)
// FallbackStreamingLLMProvider — обёртка для StreamingLLMProvider с цепочкой fallback.
type FallbackStreamingLLMProvider struct {
	providers []domain.StreamingLLMProvider
	logger    domain.Logger
	hooks     domain.Hooks
	stats     fallbackStatsInternal
}

// @sk-task graceful-degradation#T3.1: NewFallbackStreamingLLM конструктор (RQ-009, AC-006)
func NewFallbackStreamingLLM(providers []domain.StreamingLLMProvider, logger domain.Logger, hooks domain.Hooks) (*FallbackStreamingLLMProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider required")
	}
	return &FallbackStreamingLLMProvider{
		providers: providers,
		logger:    logger,
		hooks:     hooks,
	}, nil
}

func (f *FallbackStreamingLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var lastErr error
	primaryFailed := false
	for i, provider := range f.providers {
		if i > 0 {
			f.stats.setLastError(lastErr)
		}

		if err := ctx.Err(); err != nil {
			f.stats.recordCall(true)
			return "", err
		}

		result, err := provider.Generate(ctx, systemPrompt, userMessage)
		if err == nil {
			f.stats.recordCall(primaryFailed)
			return result, nil
		}

		if i == 0 {
			primaryFailed = true
		}
		lastErr = err

		if !IsRetryable(err) {
			f.stats.recordCall(primaryFailed)
			f.stats.setLastError(err)
			domain.SafeLog(ctx, f.logger, domain.LogLevelWarn, "non-retryable error, no fallback",
				domain.LogField{Key: "component", Value: "fallback_streaming_llm"},
				domain.LogField{Key: "provider_index", Value: i},
				domain.LogField{Key: "err", Value: err},
			)
			f.recordHookEvent(ctx, i, err)
			return "", err
		}

		domain.SafeLog(ctx, f.logger, domain.LogLevelWarn, "fallback to next provider",
			domain.LogField{Key: "component", Value: "fallback_streaming_llm"},
			domain.LogField{Key: "from_provider", Value: i},
			domain.LogField{Key: "to_provider", Value: i + 1},
			domain.LogField{Key: "err", Value: err},
		)
		f.stats.recordFallback()
		f.recordHookEvent(ctx, i, err)
	}

	f.stats.recordCall(primaryFailed)
	aggregate := &aggregateError{lastErr: lastErr}
	f.stats.setLastError(aggregate)
	return "", aggregate
}

func (f *FallbackStreamingLLMProvider) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if len(f.providers) == 0 {
		return fmt.Errorf("no providers")
	}
	return f.providers[0].Health(ctx)
}

func (f *FallbackStreamingLLMProvider) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out := make(chan string)

	go func() {
		defer close(out)

		primaryFailed := false
		for i, provider := range f.providers {
			if i > 0 {
				f.stats.setLastError(nil)
			}

			if err := ctx.Err(); err != nil {
				f.stats.recordCall(true)
				return
			}

			stream, err := provider.GenerateStream(ctx, systemPrompt, userMessage)
			if err != nil {
				if i == 0 {
					primaryFailed = true
				}
				if !IsRetryable(err) {
					f.stats.recordCall(primaryFailed)
					return
				}
				if i < len(f.providers)-1 {
					f.stats.recordFallback()
				}
				continue
			}

			select {
			case token, ok := <-stream:
				if !ok {
					if i == 0 {
						primaryFailed = true
					}
					if i < len(f.providers)-1 {
						f.stats.recordFallback()
					}
					continue
				}
				f.stats.recordCall(primaryFailed)
				out <- token
				for t := range stream {
					out <- t
				}
				return

			case <-ctx.Done():
				f.stats.recordCall(true)
				return
			}
		}
	}()

	return out, nil
}

func (f *FallbackStreamingLLMProvider) Stats() FallbackStats {
	return f.stats.snapshot()
}

func (f *FallbackStreamingLLMProvider) recordHookEvent(ctx context.Context, providerIndex int, err error) {
	if f.hooks == nil {
		return
	}
	ev := domain.StageStartEvent{
		Operation: fmt.Sprintf("fallback:streaming_provider=%d", providerIndex),
		Stage:     domain.HookStageGenerate,
	}
	ctx = f.hooks.StageStart(ctx, ev)
	f.hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: ev.Operation,
		Stage:     domain.HookStageGenerate,
		Err:       err,
	})
}
