# Cost Tracking — Задачи

## Phase Contract

Inputs: plan, data-model, spec.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/domain/models.go` | T1.1 |
| `internal/infrastructure/costtracker/costtracker.go` | T1.2 |
| `internal/infrastructure/llm/openai_compatible_responses.go` | T2.1 |
| `internal/infrastructure/llm/anthropic.go` | T3.1 |
| `internal/infrastructure/llm/openai_chat.go` | T3.2 |
| `internal/infrastructure/llm/mistral.go` | T3.3 |
| `internal/infrastructure/llm/deepseek.go` | T3.3 |
| `pkg/draftrag/draftrag.go` | T1.3 |
| `pkg/draftrag/costtracker.go` | T1.3 |
| `pkg/draftrag/errors.go` | T1.3 |
| `internal/infrastructure/costtracker/costtracker_test.go` | T4.1 |
| `examples/cost-tracking/main.go` | T2.2 |

## Implementation Context

- **Цель MVP**: прозрачная обёртка CostTracker + один реальный провайдер с usage + пример.
- **Границы приемки**: AC-001, 002, 003, 004, 006, 007 — MVP; AC-005 — deferred.
- **Ключевые решения**:
  - DEC-001: `UsageAwareLLMProvider` — optional capability; `LLMProvider` не меняется.
  - DEC-002: CostTracker — внешний wrapper, не встроен в PipelineOptions.
  - DEC-003: Thread-safety через `sync.Mutex` (три связанных счётчика).
  - DEC-004: `Snapshot()` / `Checkpoint()` — value copy под блокировкой; `Diff()` — свободная функция.
  - DEC-005: `ModelName()` на `UsageAwareLLMProvider`; fallback — имя из конструктора CostTracker.
- **Типы данных** (см. `data-model.md`): `TokenUsage`, `ModelPricing`, `CostSnapshot`, `UsageAwareLLMProvider`.
- **Расчёт стоимости**: `cost = (pt/1000)*inputCost + (ct/1000)*outputCost`.
- **Graceful degradation**: если провайдер не реализует `UsageAwareLLMProvider` — только `callsCount++`.
- **Streaming**: deferred (AC-005); финальный chunk может содержать usage, но в MVP не обрабатывается.
- **Вне scope**: embedder-трекинг, персистентность, бюджетные лимиты.
- **Proof signals**: unit-тесты с mock; `go test -race` чист; пример компилируется и выводит Snapshot.

## Фаза 1: Основа — domain типы + CostTracker ядро

Цель: типы данных, optional interface и ядро CostTracker без реальных провайдеров.

- [x] T1.1 Добавить типы TokenUsage, ModelPricing, CostSnapshot и интерфейс UsageAwareLLMProvider в domain. Touches: `internal/domain/interfaces.go`, `internal/domain/models.go`

- [x] T1.2 Реализовать CostTracker — обёртку LLMProvider с накоплением статистики. Touches: `internal/infrastructure/costtracker/costtracker.go`

- [x] T1.3 Re-export новых типов и CostTracker в публичный API pkg/draftrag. Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/costtracker.go`, `pkg/draftrag/errors.go`

## Фаза 2: MVP Slice — первый real provider + демо

Цель: первый работающий провайдер + пример, доказывающий ценность.

- [x] T2.1 Реализовать UsageAwareLLMProvider для OpenAICompatibleResponsesLLM (парсинг usage из API-ответа). Touches: `internal/infrastructure/llm/openai_compatible_responses.go`

- [x] T2.2 Добавить пример использования CostTracker + OpenAI provider. Touches: `examples/cost-tracking/main.go`

## Фаза 3: Основная реализация — остальные провайдеры

Цель: покрыть все коммерческие провайдеры поддержкой usage.

- [x] T3.1 Добавить UsageAwareLLMProvider для Anthropic ClaudeLLM. Touches: `internal/infrastructure/llm/anthropic.go`

- [x] T3.2 Добавить UsageAwareLLMProvider для OpenAI Chat (openai_chat.go). Touches: `internal/infrastructure/llm/openai_chat.go`

- [x] T3.3 Добавить UsageAwareLLMProvider для Mistral и DeepSeek. Touches: `internal/infrastructure/llm/mistral.go`, `internal/infrastructure/llm/deepseek.go`, `pkg/draftrag/mistral_llm.go`, `pkg/draftrag/deepseek_llm.go`

- [x] T3.4 Добавить поддержку streaming usage в CostTracker (перехват финального chunk). Touches: `internal/infrastructure/costtracker/costtracker.go`

## Фаза 4: Проверка

Цель: полное тестовое покрытие, race-проверка, verify.

- [x] T4.1 Добавить unit-тесты для CostTracker и каждого провайдера с usage + race test. Touches: `internal/infrastructure/costtracker/costtracker_test.go`, `internal/infrastructure/llm/anthropic_test.go`, `internal/infrastructure/llm/openai_chat_test.go`, `internal/infrastructure/llm/openai_compatible_responses_test.go`

- [x] T4.2 Verify — сборка, линтер, пример. Touches: `Makefile`

## Покрытие критериев приемки

- AC-001 -> T1.2, T2.1, T4.1
- AC-002 -> T1.2, T2.1, T4.1
- AC-003 -> T1.2, T4.1
- AC-004 -> T1.2, T4.1
- AC-005 -> T3.4, T4.1
- AC-006 -> T1.2, T4.1
- AC-007 -> T1.2, T4.1

## Заметки

- T1.1 и T1.2 последовательно (T1.1 -> T1.2).
- T3.1, T3.2, T3.3 независимы — можно параллелить.
- T3.4 (streaming) зависит от T1.2 и может быть отложен.
- T4.1 и T4.2 завершающие.
- Trace-маркеры `@sk-task cost-tracking` на всех owning function/method/test/type declarations.
