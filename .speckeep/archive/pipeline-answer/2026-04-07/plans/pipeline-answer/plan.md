# Pipeline.Answer для draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/pipeline-answer/spec.md`, `.draftspec/specs/pipeline-answer/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно зафиксировать минимальный prompt contract и traceability AC→tasks без расплывчатости.

## Цель

Добавить публичные методы `Pipeline.Answer` и `Pipeline.AnswerTopK`, которые выполняют полный RAG-цикл поверх существующих зависимостей `Embedder`, `VectorStore` и `LLMProvider`:
1) embed вопроса,
2) поиск контекста (`Search`),
3) сборка prompt (system+user),
4) вызов `Generate`,
5) возврат строкового ответа.

## Scope

- Public API: методы `Answer`/`AnswerTopK` на `pkg/draftrag.Pipeline` + русские godoc.
- Application: новый метод use-case на `internal/application.Pipeline` (чтобы логика оставалась внутри application слоя).
- Prompt: фиксированный system prompt и детерминированный формат user message (v1).
- Testing: unit-тесты (без внешней сети), проверяющие порядок вызовов и prompt contract.

## Implementation Surfaces

- `pkg/draftrag/draftrag.go` — добавить публичные методы `(*Pipeline) Answer` и `(*Pipeline) AnswerTopK`, валидация входных данных и маппинг ошибок в публичные sentinel (T1.1, T2.1).
- `internal/application/pipeline.go` — добавить use-case метод `Answer(ctx, question, topK)` (или аналог), реализующий retrieval→prompt→generate (T2.2).
- `pkg/draftrag/pipeline_answer_test.go` — unit-тесты публичного API (compile-time, валидация, ctx, prompt) (T3.1).
- `internal/application/pipeline_answer_test.go` — unit-тесты use-case: порядок вызовов `Embed`→`Search`→`Generate`, формат prompt contract (T3.2).
- `domain.LLMProvider`, `domain.VectorStore`, `domain.Embedder` — зависимости use-case; используем заглушки в тестах (T3.2).

## Влияние на архитектуру

- Clean Architecture сохраняется: orchestration логика размещается в application слое, а pkg слой остаётся тонким wrapper’ом с валидацией/маппингом публичных ошибок.
- Зависимости не увеличиваются: только стандартная библиотека.
- Совместимость: аддитивное API — не ломает существующий `Index`/`Query`.

## Acceptance Approach

- AC-001 -> методы `Answer`/`AnswerTopK` добавлены на `pkg/draftrag.Pipeline`; compile-time проверка в `pkg/draftrag/pipeline_answer_test.go`.
- AC-002 -> unit-тесты use-case подтверждают, что выполняются `Embed(question)` → `Search(embedding, topK)` → `Generate(systemPrompt, userMessage)` и результат `Generate` возвращается. Surfaces: `internal/application/pipeline_answer_test.go`.
- AC-003 -> unit-тесты проверяют, что `systemPrompt` и `userMessage` соответствуют Prompt Contract v1. Surfaces: `internal/application/pipeline_answer_test.go`.
- AC-004 -> валидация `question/topK` маппится в `ErrEmptyQuery` / `ErrInvalidTopK`. Surfaces: `pkg/draftrag/draftrag.go`, `pkg/draftrag/pipeline_answer_test.go`.
- AC-005 -> отменённый/просроченный ctx возвращает `context.Canceled`/`context.DeadlineExceeded` (не позже 100мс в тестовом сценарии) и не выполняет лишнюю работу. Surfaces: `pkg/draftrag/pipeline_answer_test.go`, `internal/application/pipeline_answer_test.go`.

## Данные и контракты

- Data model: persisted state не добавляется; метод вычисляет prompt и возвращает string.
- Prompt contract фиксируется в коде (константы/шаблон в application), чтобы обеспечить детерминированность тестов.
- Новые публичные ошибки не требуются: используем существующие `ErrEmptyQuery` и `ErrInvalidTopK`.

## Стратегия реализации

- DEC-001 Логика RAG-ответа в application слое
  Why: application слой — место оркестрации use-case; pkg слой остаётся тонким и стабильным.
  Tradeoff: два уровня методов (pkg wrapper + application use-case).
  Affects: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
  Validation: unit-тесты use-case + публичные unit-тесты.

- DEC-002 Fixed prompt contract v1 (без пользовательской кастомизации)
  Why: минимальный и тестируемый контракт, без расширения API до появления реальной потребности.
  Tradeoff: меньше гибкости для пользователей.
  Affects: `internal/application/pipeline.go`
  Validation: unit-тест AC-003.

- DEC-003 Использовать существующий retrieval-поток (`Embed` + `Search`) вместо вызова `QueryTopK` из pkg слоя
  Why: application уже владеет зависимостями и может собрать полный цикл без лишних round-trips и маппинга ошибок.
  Tradeoff: небольшое дублирование частей логики `Query` (embedding+search) внутри нового метода.
  Affects: `internal/application/pipeline.go`
  Validation: unit-тесты AC-002.

## Incremental Delivery

### MVP (Первая ценность)

- Добавить `AnswerTopK` (application + pkg wrapper) + тесты AC-002..AC-005.
- Добавить `Answer` как thin wrapper над `AnswerTopK` с defaultTop=5 + тест AC-001.

### Итеративное расширение

- (Out of scope) Опции кастомизации system prompt.
- (Out of scope) Ограничение контекста по токенам.

## Порядок реализации

1. Application use-case метод `Answer(ctx, question, topK)`.
2. Публичные методы `Pipeline.Answer*` и валидация/маппинг ошибок.
3. Unit-тесты application и pkg.

## Риски

- Риск 1: prompt contract будет трудно менять без breaking changes.
  Mitigation: фиксируем минимальный v1 формат и оставляем расширения на будущие фичи.
- Риск 2: недостаточная контекстная информация (нулевой результат retrieval).
  Mitigation: контракт допускает пустой “Контекст:” и system prompt явно описывает поведение при недостатке информации.

## Rollout и compatibility

- Rollout не требуется: аддитивные методы в библиотеке.
- Compatibility: не изменяем поведение `Index`/`Query`; добавляем новые entrypoints.

## Проверка

- Automated:
  - `go test ./...`
  - unit-тесты по AC-001..AC-005
- Manual:
  - `go doc` на `Pipeline.Answer`/`Pipeline.AnswerTopK` (godoc на русском).

## Соответствие конституции

- нет конфликтов: `context.Context` первым параметром, минимальные зависимости, тестируемость через заглушки, Clean Architecture соблюдена.

