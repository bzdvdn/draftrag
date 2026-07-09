// Package costtracker предоставляет обёртку LLMProvider с подсчётом токенов и стоимости.
package costtracker

import (
	"context"
	"errors"
	"sync"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var ErrStreamingNotSupported = errors.New("LLM provider does not support streaming")

// CostTracker — прозрачная обёртка над LLMProvider, накапливающая статистику
// токенов и стоимости LLM-вызовов.
//
// Потокобезопасность: sync.Mutex на запись и чтение счётчиков.
type CostTracker struct {
	llm          domain.LLMProvider
	pricing      map[string]domain.ModelPricing
	defaultModel string

	mu               sync.Mutex
	promptTokens     int64
	completionTokens int64
	totalTokens      int64
	totalCost        float64
	callsCount       int64
}

// NewCostTracker создаёт CostTracker, оборачивающий llm.
// Если pricing == nil, расчёт стоимости не производится (TotalCost всегда 0).
func NewCostTracker(llm domain.LLMProvider, pricing map[string]domain.ModelPricing) *CostTracker {
	return &CostTracker{
		llm:     llm,
		pricing: pricing,
	}
}

// @sk-task cost-tracking: Generate — прозрачная обёртка с подсчётом (AC-001, AC-006, RQ-001, RQ-005)
// Generate генерирует ответ, подсчитывая токены и стоимость.
//
// Если underlying provider реализует UsageAwareLLMProvider, используется
// GenerateWithUsage для извлечения token usage.
// Иначе — только инкремент callsCount.
func (ct *CostTracker) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	ua, ok := ct.llm.(domain.UsageAwareLLMProvider)
	if ok {
		answer, usage, err := ua.GenerateWithUsage(ctx, systemPrompt, userMessage)
		if err != nil {
			return "", err
		}
		modelName := ua.ModelName()
		ct.accumulate(usage, modelName)
		return answer, nil
	}

	answer, err := ct.llm.Generate(ctx, systemPrompt, userMessage)
	if err != nil {
		return "", err
	}
	ct.accumulate(domain.TokenUsage{}, "")
	return answer, nil
}

// @sk-task cost-tracking: GenerateStream — обёртка для streaming (AC-005, RQ-006, T3.4)
// GenerateStream делегирует вызов underlying StreamingLLMProvider и
// извлекает token usage из финального chunk, если провайдер реализует
// UsageAwareStreamingLLMProvider. Иначе — только инкремент callsCount.
func (ct *CostTracker) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sp, ok := ct.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	ch, err := sp.GenerateStream(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, err
	}

	out := make(chan string, cap(ch))
	go func() {
		defer close(out)
		for token := range ch {
			out <- token
		}

		if usa, ok := ct.llm.(domain.UsageAwareStreamingLLMProvider); ok {
			if usage, ok := usa.StreamUsage(); ok {
				modelName := ""
				if ua, ok := ct.llm.(domain.UsageAwareLLMProvider); ok {
					modelName = ua.ModelName()
				}
				ct.accumulate(usage, modelName)
				return
			}
		}
		ct.accumulateCall()
	}()

	return out, nil
}

// @sk-task cost-tracking: Health — делегирует underlying provider (RQ-001)
func (ct *CostTracker) Health(ctx context.Context) error {
	return ct.llm.Health(ctx)
}

// @sk-task cost-tracking: Snapshot — атомарный срез статистики (AC-003, RQ-003)
// Snapshot возвращает текущую накопленную статистику с момента создания
// CostTracker или последнего Reset.
func (ct *CostTracker) Snapshot() domain.CostSnapshot {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return domain.CostSnapshot{
		PromptTokens:     ct.promptTokens,
		CompletionTokens: ct.completionTokens,
		TotalTokens:      ct.totalTokens,
		TotalCost:        ct.totalCost,
		CallsCount:       ct.callsCount,
	}
}

// @sk-task cost-tracking: Checkpoint — алиас Snapshot (AC-007, RQ-007)
// Checkpoint фиксирует текущий absolute срез для последующего расчёта дельты
// через domain.Diff.
func (ct *CostTracker) Checkpoint() domain.CostSnapshot {
	return ct.Snapshot()
}

// @sk-task cost-tracking: Reset — обнуление статистики (AC-004, RQ-004)
// Reset сбрасывает все счётчики в ноль. Конкурентно завершённые вызовы
// между вызовом Reset и Snapshot могут быть учтены (snapshot до/после).
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.promptTokens = 0
	ct.completionTokens = 0
	ct.totalTokens = 0
	ct.totalCost = 0
	ct.callsCount = 0
}

// accumulate потокобезопасно обновляет счётчики.
func (ct *CostTracker) accumulate(usage domain.TokenUsage, modelName string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.promptTokens += usage.PromptTokens
	ct.completionTokens += usage.CompletionTokens
	ct.totalTokens += usage.TotalTokens
	ct.callsCount++

	if ct.pricing != nil && usage.TotalTokens > 0 {
		p, ok := ct.pricing[modelName]
		if !ok {
			p, ok = ct.pricing[ct.defaultModel]
		}
		if ok {
			cost := (float64(usage.PromptTokens) / 1000.0) * p.InputCostPer1K
			cost += (float64(usage.CompletionTokens) / 1000.0) * p.OutputCostPer1K
			ct.totalCost += cost
		}
	}
}

// SetDefaultModel устанавливает имя модели по умолчанию для расчёта стоимости,
// используется если UsageAwareLLMProvider.ModelName() не найден в pricing.
func (ct *CostTracker) SetDefaultModel(model string) {
	ct.defaultModel = model
}

// accumulateCall инкрементирует только счётчик вызовов (без токенов и стоимости).
// Используется для streaming-вызовов, где token usage недоступен.
func (ct *CostTracker) accumulateCall() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.callsCount++
}
