# Observability: хуки/метрики для pipeline стадий (v1) — План

## Phase Contract

Inputs: spec и минимальный контекст репозитория.  
Outputs: plan, tasks, (при необходимости) data-model.  
Stop if: невозможно добавить hooks аддитивно без изменения поведения по умолчанию.

## Цель

Добавить опциональные hooks для стадий pipeline, чтобы пользователь мог измерять latency и ошибки по стадиям Index/Query/Answer без привязки к конкретному стеку.

## Scope

- Новые типы для hooks (без внешних зависимостей) и опция в `PipelineOptions`/`PipelineConfig`.
- Hooks покрывают стадии: chunking (если включён), embed, vector search, LLM generate.
- Hooks получают `context.Context`, operation name и ошибку (если была).
- По умолчанию (nil hooks) — no-op, поведение не меняется.

## Implementation Surfaces

- `internal/domain/hooks.go` (или `internal/domain/models.go`): интерфейс `Hooks` + enum/константы стадий.
- `internal/application/pipeline.go`: вызовы hooks вокруг embed/search/generate/chunking, измерение duration.
- `pkg/draftrag/draftrag.go`: опция `Hooks` в `PipelineOptions` (public API), прокидывание в application config.
- `internal/application/observability_hooks_test.go`: тест порядка и количества событий на Answer.

## Влияние на архитектуру

- Domain: только интерфейсы/типы, без зависимостей (только stdlib).
- Application: instrumentation вокруг существующих вызовов, без изменения бизнес-логики.
- Public API: аддитивно через `PipelineOptions`.

## Acceptance Approach

- AC-001: тестирует, что на `Answer*` вызываются hooks для embed/search/generate (и chunking при наличии chunker).
- AC-002: `NewPipeline/NewPipelineWithOptions` без hooks работает как раньше; существующие тесты остаются зелёными.

## Стратегия реализации

- DEC-001 Единый hook интерфейс + события Start/End
  Why: пользователю удобно измерять latency и ошибки без ручного тайминга.
  Tradeoff: sync вызовы; пользователь обязан держать hooks лёгкими.
  Affects: domain hooks types + application instrumentation.
  Validation: unit-тест фиксирует порядок Start/End и наличие duration.

## Риски

- Hooks могут замедлять pipeline.
  Mitigation: nil-check перед вызовом; payload минимальный; sync by design.

## Проверка

- `go test ./...`
- Unit-тесты на hooks order/count + отсутствие паник при nil hooks.

## Соответствие конституции

- Нет конфликтов: интерфейсная абстракция, чистая архитектура, контекстная безопасность, тесты.

