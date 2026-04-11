# Ограничение контекста в Prompt для draftRAG — План

## Phase Contract

Inputs: `.draftspec/specs/prompt-context-limit/spec.md`, `.draftspec/specs/prompt-context-limit/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно задать детерминированные правила обрезания контекста без неоднозначностей.

## Цель

Добавить ограничение размера секции “Контекст:” для `Pipeline.Answer*`, управляемое через `PipelineOptions`:
- `MaxContextChunks` — лимит числа чанков,
- `MaxContextChars` — лимит числа символов контекста,
с сохранением Prompt Contract v1.

## Scope

- Public API: расширить `PipelineOptions` полями `MaxContextChars` и `MaxContextChunks`.
- Application: применить лимиты при сборке `userMessage` (в функции builder’а).
- Testing: unit-тесты на обрезание по чанкам, по символам и совместное применение.

## Implementation Surfaces

- `pkg/draftrag/draftrag.go` — добавить поля options и прокинуть лимиты в application config (T1.1, T2.1).
- `internal/application/pipeline.go` — расширить `PipelineConfig` лимитами и обновить сборку user message (T2.2).
- `pkg/draftrag/prompt_context_limit_test.go` — compile-time и sanity тесты options (T3.1).
- `internal/application/prompt_context_limit_test.go` — unit-тесты на аргументы `LLM.Generate` и длины/кол-во контекста (T3.2).

## Влияние на архитектуру

- Изменения только в пределах Pipeline config и prompt builder.
- Нет новых внешних зависимостей.

## Acceptance Approach

- AC-001 -> compile-time использование новых полей options в `pkg/draftrag/prompt_context_limit_test.go`.
- AC-002 -> unit-тест: `MaxContextChunks=1` при 3 чанках -> в prompt только 1 чанк. Surface: `internal/application/prompt_context_limit_test.go`.
- AC-003 -> unit-тест: `MaxContextChars` ограничивает длину секции контекста. Surface: `internal/application/prompt_context_limit_test.go`.
- AC-004 -> unit-тест: оба лимита включены и одновременно соблюдаются. Surface: `internal/application/prompt_context_limit_test.go`.

## Данные и контракты

- Prompt Contract v1 сохраняется:
  - “Контекст:\n...”
  - “\nВопрос:\n...”
- Лимиты применяются только к содержимому между “Контекст:\n” и “\nВопрос:\n”.

## Стратегия реализации

- DEC-001 Порядок применения лимитов: chunks → chars
  Why: ограничение по чанкам проще и уменьшает объём до применения char лимита.
  Tradeoff: при очень длинном первом чанке основным ограничителем будет char лимит.
  Affects: `internal/application/pipeline.go`
  Validation: unit-тесты AC-004.

- DEC-002 Обрезание внутри последнего чанка для соблюдения `MaxContextChars`
  Why: строгий лимит по символам должен соблюдаться даже если чанк длиннее лимита.
  Tradeoff: контекст может заканчиваться на середине чанка.
  Affects: `internal/application/pipeline.go`
  Validation: unit-тест AC-003.

- DEC-003 Валидация отрицательных лимитов как panic при создании pipeline
  Why: это ошибка конфигурации программиста; не усложняем API error-ами.
  Tradeoff: panic при неверной конфигурации.
  Affects: `pkg/draftrag/draftrag.go`
  Validation: (опционально) unit-тест на panic.

## Incremental Delivery

### MVP (Первая ценность)

- Добавить поля options + wiring в internal config.
- Обновить prompt builder и тесты AC-001..AC-004.

### Итеративное расширение

- (Out of scope) лимит по токенам.
- (Out of scope) “…” маркер обрезания.

## Порядок реализации

1. Расширить `PipelineOptions` и internal `PipelineConfig`.
2. Обновить builder user message.
3. Добавить unit-тесты.

## Риски

- Риск 1: “ломающее” изменение prompt builder.
  Mitigation: сохранить заголовки и формат v1; тесты сравнивают ожидаемые строки/инварианты.

## Rollout и compatibility

- Backward compatible: при `MaxContext* == 0` поведение не меняется.

## Проверка

- `go test ./...`
- unit-тесты по AC-001..AC-004.

## Соответствие конституции

- нет конфликтов: зависимости минимальны, тестируемость обеспечена, контекст не нарушается.

