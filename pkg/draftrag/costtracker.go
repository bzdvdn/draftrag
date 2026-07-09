package draftrag

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/infrastructure/costtracker"
)

// CostTracker — прозрачная обёртка над LLMProvider, накапливающая статистику
// токенов и стоимости LLM-вызовов.
//
// @sk-task cost-tracking: публичный CostTracker (AC-001..004, 006, 007, RQ-001..005, 007)
type CostTracker struct {
	inner *costtracker.CostTracker
}

// NewCostTracker создаёт CostTracker, оборачивающий llm.
// Если pricing == nil, расчёт стоимости не производится.
//
// @sk-task cost-tracking: конструктор CostTracker (AC-001, RQ-001)
func NewCostTracker(llm LLMProvider, pricing map[string]ModelPricing) *CostTracker {
	return &CostTracker{
		inner: costtracker.NewCostTracker(llm, pricing),
	}
}

// Generate генерирует ответ, подсчитывая токены и стоимость.
//
// @sk-task cost-tracking: Generate (AC-001, RQ-001)
func (ct *CostTracker) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return ct.inner.Generate(ctx, systemPrompt, userMessage)
}

// Health проверяет доступность underlying LLM провайдера.
func (ct *CostTracker) Health(ctx context.Context) error {
	return ct.inner.Health(ctx)
}

// Snapshot возвращает текущую накопленную статистику.
//
// @sk-task cost-tracking: Snapshot (AC-003, RQ-003)
func (ct *CostTracker) Snapshot() CostSnapshot {
	return ct.inner.Snapshot()
}

// Checkpoint фиксирует текущий absolute срез для последующего расчёта дельты.
//
// @sk-task cost-tracking: Checkpoint (AC-007, RQ-007)
func (ct *CostTracker) Checkpoint() CostSnapshot {
	return ct.inner.Checkpoint()
}

// Reset сбрасывает все счётчики в ноль.
//
// @sk-task cost-tracking: Reset (AC-004, RQ-004)
func (ct *CostTracker) Reset() {
	ct.inner.Reset()
}

// SetDefaultModel устанавливает имя модели по умолчанию для расчёта стоимости.
func (ct *CostTracker) SetDefaultModel(model string) {
	ct.inner.SetDefaultModel(model)
}

// GenerateStream делегирует вызов underlying StreamingLLMProvider.
// Подсчитывает вызов (callsCount++), token usage из streaming не извлекается.
//
// @sk-task cost-tracking: GenerateStream (AC-005, T3.4)
func (ct *CostTracker) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	return ct.inner.GenerateStream(ctx, systemPrompt, userMessage)
}
