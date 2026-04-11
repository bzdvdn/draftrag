# PipelineOptions / NewPipelineWithOptions для draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/pipeline-config/spec.md`, `.draftspec/specs/pipeline-config/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно сохранить backward compatibility, не усложняя публичный API.

## Цель

Добавить единый публичный entrypoint `NewPipelineWithOptions(...)` и `PipelineOptions`, чтобы управлять:
- `DefaultTopK` (для `Query`/`Answer`),
- `SystemPrompt` (override для `Answer`),
- `Chunker` (включение chunker пути для `Index`),
при сохранении существующих фабрик и дефолтов.

## Scope

- Public API: `PipelineOptions` + `NewPipelineWithOptions` в `pkg/draftrag`.
- Backward compatibility: `NewPipeline` и `NewPipelineWithChunker` остаются рабочими.
- Application: прокинуть `SystemPrompt` и optional `Chunker` в use-case слой.
- Testing: unit-тесты на defaultTopK, system prompt override, chunker via options.

## Implementation Surfaces

- `pkg/draftrag/draftrag.go` — добавить `PipelineOptions` и `NewPipelineWithOptions` (T1.1, T1.2).
- `internal/application/pipeline.go` — расширить pipeline конфигурацией system prompt (и/или options struct) (T2.1).
- `pkg/draftrag/pipeline_options_test.go` — тесты публичного API (compile-time, DefaultTopK, SystemPrompt, Chunker) (T3.1).
- `internal/application/pipeline_options_test.go` — тесты use-case: Generate получает переопределённый system prompt, Index использует chunker если задан (T3.2).

## Влияние на архитектуру

- Публичный API расширяется аддитивно.
- Application слой получает минимальную конфигурацию (system prompt), оставаясь независимым от infrastructure.
- Избегаем `(*Pipeline, error)` в конструкторах, чтобы не ломать стиль библиотеки (как в других фабриках).

## Acceptance Approach

- AC-001 -> compile-time тест на наличие `PipelineOptions` и `NewPipelineWithOptions` в `pkg/draftrag/pipeline_options_test.go`.
- AC-002 -> unit-тест проверяет, что `DefaultTopK` используется в `Query`/`Answer` (делегирование на TopK методы). Surface: `pkg/draftrag/pipeline_options_test.go`.
- AC-003 -> unit-тест проверяет, что `LLMProvider.Generate` получает `systemPrompt` из options. Surface: `internal/application/pipeline_options_test.go` (или pkg тест, если проще).
- AC-004 -> unit-тест проверяет, что `Index` использует chunker путь, если задан `opts.Chunker`. Surface: `internal/application/pipeline_options_test.go`.

## Данные и контракты

- Data model: `PipelineOptions` (в `pkg`) и минимальная “конфигурация pipeline” в application.
- Контракты:
  - `DefaultTopK` применяется только в `Query`/`Answer` wrappers.
  - `SystemPrompt` применяется в `Answer` use-case (если не пустой).
  - `Chunker` включает chunker путь индексации.

## Стратегия реализации

- DEC-001 Options struct только в pkg, а в application — минимальные поля конфигурации
  Why: публичная модель опций не должна протекать в internal; application получает только то, что реально нужно.
  Tradeoff: потребуется маппинг options → internal config.
  Affects: `pkg/draftrag/draftrag.go`, `internal/application/pipeline.go`
  Validation: unit-тесты AC-002..AC-004.

- DEC-002 Валидация `DefaultTopK <= 0` как panic на конструкторе
  Why: ошибка конфигурации программиста; не усложняем API возвратом error.
  Tradeoff: panic при неправильной конфигурации.
  Affects: `pkg/draftrag/draftrag.go`
  Validation: (опционально) unit-тест на panic, либо оставить без теста как простое правило.

- DEC-003 Сохранить существующие фабрики и дефолты
  Why: backward compatibility.
  Tradeoff: временно остаются несколько entrypoints.
  Affects: `pkg/draftrag/draftrag.go`
  Validation: существующие тесты не ломаются.

## Incremental Delivery

### MVP (Первая ценность)

- `PipelineOptions` + `NewPipelineWithOptions`
- DefaultTopK + SystemPrompt override + Chunker wiring
- unit-тесты AC-001..AC-004

### Итеративное расширение

- (Out of scope) MaxContextChars / MaxContextChunks для prompt.

## Порядок реализации

1. Добавить `PipelineOptions` и `NewPipelineWithOptions` в pkg.
2. Прокинуть конфигурацию в application pipeline.
3. Добавить тесты для AC.

## Риски

- Риск 1: “расползание” конфигурации между pkg и application.
  Mitigation: держать internal config минимальным и целевым.
- Риск 2: случайное изменение дефолтов старых фабрик.
  Mitigation: отдельные тесты на дефолты и сохранение существующих тестов.

## Rollout и compatibility

- Rollout не требуется.
- Compatibility: аддитивные изменения, существующие entrypoints остаются.

## Проверка

- `go test ./...`
- unit-тесты по AC-001..AC-004.

## Соответствие конституции

- нет конфликтов: интерфейсы сохраняются, зависимости минимальны, ctx safety не меняется.

