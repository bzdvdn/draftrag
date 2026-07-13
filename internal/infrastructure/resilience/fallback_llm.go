package resilience

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task graceful-degradation#T2.1: FallbackLLMProvider — LLMProvider с цепочкой fallback (RQ-001, AC-001)
// FallbackLLMProvider — обёртка для LLMProvider с цепочкой fallback при retryable-ошибках.
type FallbackLLMProvider struct {
	providers []domain.LLMProvider
	logger    domain.Logger
	hooks     domain.Hooks
	stats     fallbackStatsInternal
}

// @sk-task graceful-degradation#T2.1: NewFallbackLLM конструктор (RQ-001, AC-001)
func NewFallbackLLM(providers []domain.LLMProvider, logger domain.Logger, hooks domain.Hooks) (*FallbackLLMProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider required")
	}
	return &FallbackLLMProvider{
		providers: providers,
		logger:    logger,
		hooks:     hooks,
	}, nil
}

// Generate returns a response from the first successful provider, falling back on retryable errors.
func (f *FallbackLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
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
				domain.LogField{Key: "component", Value: "fallback_llm"},
				domain.LogField{Key: "provider_index", Value: i},
				domain.LogField{Key: "err", Value: err},
			)
			f.recordHookEvent(ctx, i, err)
			return "", err
		}

		domain.SafeLog(ctx, f.logger, domain.LogLevelWarn, "fallback to next provider",
			domain.LogField{Key: "component", Value: "fallback_llm"},
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

// Health возвращает статус первого (primary) провайдера.
func (f *FallbackLLMProvider) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if len(f.providers) == 0 {
		return fmt.Errorf("no providers")
	}
	return f.providers[0].Health(ctx)
}

// Stats возвращает снепшот статистики fallback-цепи.
func (f *FallbackLLMProvider) Stats() FallbackStats {
	return f.stats.snapshot()
}

func (f *FallbackLLMProvider) recordHookEvent(ctx context.Context, providerIndex int, err error) {
	if f.hooks == nil {
		return
	}
	ev := domain.StageStartEvent{
		Operation: fmt.Sprintf("fallback:provider=%d", providerIndex),
		Stage:     domain.HookStageGenerate,
	}
	ctx = f.hooks.StageStart(ctx, ev)
	f.hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: ev.Operation,
		Stage:     domain.HookStageGenerate,
		Err:       err,
	})
}
