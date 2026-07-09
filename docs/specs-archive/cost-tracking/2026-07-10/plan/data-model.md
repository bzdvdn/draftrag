# Data Model: Cost Tracking

## Status

- **change**: добавлены типы в `internal/domain/` + optional interface.

## New Types

### `TokenUsage`

Пакет: `internal/domain`

```go
type TokenUsage struct {
    PromptTokens     int64
    CompletionTokens int64
    TotalTokens      int64
}
```

- Фиксирует количество токенов, возвращённое LLM API для одного вызова `Generate`.
- Поле `TotalTokens` может не совпадать с `PromptTokens + CompletionTokens`, если API предоставляет только суммарное значение. CostTracker использует `PromptTokens` и `CompletionTokens` для расчёта стоимости; `TotalTokens` — для отчёта.
- Используется в `UsageAwareLLMProvider.GenerateWithUsage`.

### `ModelPricing`

Пакет: `internal/domain`

```go
type ModelPricing struct {
    InputCostPer1K  float64 // USD за 1K input (prompt) токенов
    OutputCostPer1K float64 // USD за 1K output (completion) токенов
}
```

- Хранится в `CostTracker` как `map[string]ModelPricing`, где ключ — имя модели.
- Расчёт: `cost = (usage.PromptTokens/1000)*pricing.InputCostPer1K + (usage.CompletionTokens/1000)*pricing.OutputCostPer1K`.

### `CostSnapshot`

Пакет: `internal/domain`

```go
type CostSnapshot struct {
    TotalTokens   int64
    PromptTokens  int64
    CompletionTokens int64
    TotalCost     float64
    CallsCount    int64
}
```

- Атомарный срез накопленной статистики.
- Возвращается методами `CostTracker.Snapshot()` и `CostTracker.Checkpoint()`.
- `Diff(prev, curr CostSnapshot) CostSnapshot` — свободная функция для расчёта дельты между двумя снапшотами.

### `UsageAwareLLMProvider`

Пакет: `internal/domain`

```go
type UsageAwareLLMProvider interface {
    LLMProvider
    GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, TokenUsage, error)
    ModelName() string
}
```

- Optional capability для LLMProvider, возвращающих token usage в API-ответе.
- `ModelName()` возвращает имя модели (например, `"gpt-4o"`, `"claude-3-haiku-20240307"`).
- Реализуется конкретными провайдерами через парсинг `usage` из JSON-ответа.

### `CostTracker`

Пакет: `internal/infrastructure/costtracker` (re-export через `pkg/draftrag/`)

```go
type CostTracker struct {
    llm      domain.LLMProvider
    pricing  map[string]domain.ModelPricing
    mu       sync.Mutex
    // накопленные счётчики
    totalTokens       int64
    promptTokens      int64
    completionTokens  int64
    totalCost         float64
    callsCount        int64
}
```

- Реализует `domain.LLMProvider` (прозрачная обёртка).
- При `Generate` проверяет, реализует ли underlying provider `UsageAwareLLMProvider`:
  - Если да — вызывает `GenerateWithUsage`, аккумулирует usage и стоимость.
  - Если нет — вызывает `Generate`, инкрементирует только `callsCount`.
- Потокобезопасность: `sync.Mutex`.

## Unchanged Types

- `domain.LLMProvider` — интерфейс **не меняется** (backward compatibility).
- `domain.StreamingLLMProvider` — интерфейс **не меняется**.
- `domain.Hooks` — **не расширяется** (cost tracking ортогонален hooks).
- `domain.PipelineOptions` — **не меняется** (CostTracker — внешняя обёртка).

## Contract Compatibility

- Новые типы — только добавление. Никакие существующие контракты не ломаются.
- `UsageAwareLLMProvider` — optional capability; type assertion не влияет на существующий код.
