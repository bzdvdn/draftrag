# Retrieval: дедупликация источников (v1) — План

## Phase Contract

Inputs: `.draftspec/specs/retrieval-deduplication/spec.md`, `.draftspec/specs/retrieval-deduplication/inspect.md`, `.draftspec/constitution.md`.
Outputs: `.draftspec/plans/retrieval-deduplication/plan.md`, `.draftspec/plans/retrieval-deduplication/data-model.md`.
Stop if: невозможно реализовать дедупликацию аддитивно без изменения поведения по умолчанию.

## Цель

Добавить опциональную дедупликацию retrieval результата (v1: по `ParentID`), чтобы уменьшить повторы в sources/prompt, сохраняя текущий default behavior без изменений.

## Scope

- In scope:
  - конфигурационный флаг/режим дедупликации (по умолчанию выключен).
  - алгоритм дедупликации по `ParentID` (оставить лучший по score).
  - unit-тесты без внешней сети.
- Out of scope:
  - дедупликация по hash контента, семантическая дедупликация, кластеризация.
  - изменения протокола цитирования внутри текста ответа.

## Implementation Surfaces

- `internal/application/pipeline.go`:
  - применить дедупликацию после retrieval (Search/SearchWithFilter) и до построения prompt.
- `pkg/draftrag/draftrag.go`:
  - добавить опцию в `PipelineOptions` для включения дедупликации.
- `internal/application/*_test.go`:
  - unit-тесты на дедупликацию и на “по умолчанию ничего не меняется”.

## Влияние на архитектуру

- Domain не меняется.
- Изменения локализованы в application+pkg конфиге; infrastructure не затрагивается.
- Поведение по умолчанию сохраняется, так как дедупликация включается только через опцию.

## Acceptance Approach

- AC-001 -> unit-тест: при включённой дедупликации и нескольких чанках с одним `ParentID` остаётся один (лучший по score).
- AC-002 -> unit-тест: при выключенной дедупликации результат полностью совпадает с исходным retrieval.

## Данные и контракты

- Вход: `domain.RetrievalResult` (`Chunks []RetrievedChunk`).
- Выход: тот же `domain.RetrievalResult`, но с опционально “очищенным” списком `Chunks`.
- Стабильность:
  - сортировка должна оставаться детерминированной (stable sort) и не ломать tie-breaker.

## Стратегия реализации

- DEC-001 “Дедупликация в application слое”
  Why: именно application orchestrates retrieval→prompt, и нам нужно менять prompt-context, не трогая domain/infrastructure.
  Tradeoff: дедупликация применяется только в pipeline use-case (не на уровне VectorStore).
  Affects: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`.
  Validation: unit-тесты AC-001/AC-002.

- DEC-002 “v1: только ParentID”
  Why: даёт быстрый выигрыш без сложных эвристик.
  Tradeoff: near-duplicate по контенту останутся.
  Affects: алгоритм дедупликации и опции.
  Validation: unit-тест AC-001.

## Incremental Delivery

### MVP (Первая ценность)

- Добавить опцию включения дедупликации.
- Реализовать дедуп по ParentID.
- Unit-тесты AC-001/AC-002.

### Итеративное расширение

- Добавить режим дедупликации по контентному hash (если потребуется).
- Добавить интеграционный тест “retrieval+answer” при включенной дедупликации.

## Порядок реализации

1. Добавить опцию в `PipelineOptions` и прокинуть в `application.PipelineConfig`.
2. Реализовать дедупликацию и применить в retrieval path перед prompt.
3. Добавить unit-тесты.
4. Прогнать `go test ./...`.

## Риски

- Риск: непредсказуемый порядок при одинаковых score.
  Mitigation: stable sort + детерминированный выбор “первый встретившийся” при равенстве.

## Rollout и compatibility

- Rollout: включается через опцию, по умолчанию выключено.
- Compatibility: аддитивное изменение `PipelineOptions`, существующий код не ломается.

## Проверка

- `go test ./...`
- Unit-тесты по AC-001/AC-002.

## Соответствие конституции

- Нет конфликтов: изменения тестируемы, контекстная безопасность сохраняется, внешних зависимостей нет.

