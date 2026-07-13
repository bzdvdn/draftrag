package resilience

import (
	"context"
	"fmt"
	"sync"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task graceful-degradation#T3.2: FallbackUsageAwareLLMProvider (RQ-010, AC-007)
// FallbackUsageAwareLLMProvider — обёртка для UsageAwareLLMProvider с цепочкой fallback.
type FallbackUsageAwareLLMProvider struct {
	providers     []domain.UsageAwareLLMProvider
	logger        domain.Logger
	hooks         domain.Hooks
	stats         fallbackStatsInternal
	activeIndexMu sync.Mutex
	activeIndex   int
}

// @sk-task graceful-degradation#T3.2: NewFallbackUsageAwareLLM конструктор (RQ-010, AC-007)
func NewFallbackUsageAwareLLM(providers []domain.UsageAwareLLMProvider, logger domain.Logger, hooks domain.Hooks) (*FallbackUsageAwareLLMProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider required")
	}
	return &FallbackUsageAwareLLMProvider{
		providers:   providers,
		logger:      logger,
		hooks:       hooks,
		activeIndex: 0,
	}, nil
}

// Generate делегирует вызов провайдерам в порядке приоритета с fallback.
func (f *FallbackUsageAwareLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
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
			f.setActiveIndex(i)
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
				domain.LogField{Key: "component", Value: "fallback_usage_aware"},
				domain.LogField{Key: "provider_index", Value: i},
				domain.LogField{Key: "err", Value: err},
			)
			f.recordHookEvent(ctx, i, err)
			return "", err
		}

		domain.SafeLog(ctx, f.logger, domain.LogLevelWarn, "fallback to next provider",
			domain.LogField{Key: "component", Value: "fallback_usage_aware"},
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
func (f *FallbackUsageAwareLLMProvider) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if len(f.providers) == 0 {
		return fmt.Errorf("no providers")
	}
	return f.providers[0].Health(ctx)
}

// GenerateWithUsage делегирует вызов с трекингом токенов и fallback.
func (f *FallbackUsageAwareLLMProvider) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	if err := ctx.Err(); err != nil {
		return "", domain.TokenUsage{}, err
	}

	var lastErr error
	primaryFailed := false
	for i, provider := range f.providers {
		if i > 0 {
			f.stats.setLastError(lastErr)
		}

		if err := ctx.Err(); err != nil {
			f.stats.recordCall(true)
			return "", domain.TokenUsage{}, err
		}

		result, usage, err := provider.GenerateWithUsage(ctx, systemPrompt, userMessage)
		if err == nil {
			f.setActiveIndex(i)
			f.stats.recordCall(primaryFailed)
			return result, usage, nil
		}

		if i == 0 {
			primaryFailed = true
		}
		lastErr = err

		if !IsRetryable(err) {
			f.stats.recordCall(primaryFailed)
			f.stats.setLastError(err)
			domain.SafeLog(ctx, f.logger, domain.LogLevelWarn, "non-retryable error, no fallback",
				domain.LogField{Key: "component", Value: "fallback_usage_aware"},
				domain.LogField{Key: "provider_index", Value: i},
				domain.LogField{Key: "err", Value: err},
			)
			f.recordHookEvent(ctx, i, err)
			return "", domain.TokenUsage{}, err
		}

		domain.SafeLog(ctx, f.logger, domain.LogLevelWarn, "fallback to next provider",
			domain.LogField{Key: "component", Value: "fallback_usage_aware"},
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
	return "", domain.TokenUsage{}, aggregate
}

// ModelName возвращает имя модели первого провайдера.
func (f *FallbackUsageAwareLLMProvider) ModelName() string {
	f.activeIndexMu.Lock()
	idx := f.activeIndex
	f.activeIndexMu.Unlock()
	if idx < 0 || idx >= len(f.providers) {
		return ""
	}
	return f.providers[idx].ModelName()
}

func (f *FallbackUsageAwareLLMProvider) setActiveIndex(idx int) {
	f.activeIndexMu.Lock()
	f.activeIndex = idx
	f.activeIndexMu.Unlock()
}

// Stats возвращает снепшот статистики fallback-цепи.
func (f *FallbackUsageAwareLLMProvider) Stats() FallbackStats {
	return f.stats.snapshot()
}

func (f *FallbackUsageAwareLLMProvider) recordHookEvent(ctx context.Context, providerIndex int, err error) {
	if f.hooks == nil {
		return
	}
	ev := domain.StageStartEvent{
		Operation: fmt.Sprintf("fallback:usage_provider=%d", providerIndex),
		Stage:     domain.HookStageGenerate,
	}
	ctx = f.hooks.StageStart(ctx, ev)
	f.hooks.StageEnd(ctx, domain.StageEndEvent{
		Operation: ev.Operation,
		Stage:     domain.HookStageGenerate,
		Err:       err,
	})
}
