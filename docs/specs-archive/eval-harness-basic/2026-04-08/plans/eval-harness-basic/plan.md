# Eval harness: базовая оценка качества RAG (v1) — План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи.  
Outputs: план реализации, data-model (структуры harness), tasks.  
Stop if: нельзя сделать метрики детерминированными без сетевых зависимостей в тестах.

## Цель

Добавить библиотечный eval harness, который принимает датасет кейсов и возвращает:
- агрегированные retrieval-метрики (hit@k, MRR),
- подробный отчёт по кейсам для дебага.

## Scope

- Новый публичный пакет `pkg/draftrag/eval` с моделями данных и API harness.
- Поддержка метрик hit@k и MRR в MVP.
- Детерминированные unit-тесты без сети.
- Граница: не добавляем CLI/формат импорта/экспорта датасета.

## Implementation Surfaces

- `pkg/draftrag/eval/models.go`: `Case`, `Result`, `Report`, `Metrics`.
- `pkg/draftrag/eval/harness.go`: `Harness` + `Run(ctx, pipeline, cases, opts)`.
- `pkg/draftrag/eval/metrics.go`: вычисление hit@k и MRR, per-case ranks.
- `pkg/draftrag/eval/harness_test.go`: синтетический датасет с контролируемым retrieval.

## Влияние на архитектуру

- Только новый публичный пакет (аддитивно).
- Для детерминированности в тестах используем fake pipeline через минимальный интерфейс (например, `QueryTopK`), без LLM и без сети.

## Acceptance Approach

- AC-001: тест строит retrieval выдачу с известными позициями релевантных `ParentID`, проверяет hit@k и MRR.
- AC-002: `go test ./...` остаётся зелёным; существующие пакеты и импорты не меняются.

## Данные и контракты

- Новые структуры данных в `pkg/draftrag/eval`, не меняющие существующие доменные модели.
- Контракт входа: harness читает `RetrievalResult` и извлекает ключ сопоставления (в v1: по `ParentID`).

## Стратегия реализации

- DEC-001 Вход через минимальный интерфейс retrieval
  Why: harness не должен зависеть от LLM и конкретной pipeline-реализации.
  Tradeoff: оцениваем только retrieval-часть (answer quality остаётся за пределами v1).
  Affects: `pkg/draftrag/eval/harness.go`.
  Validation: unit-тесты используют fake retrieval runner.

- DEC-002 Метрики по `ParentID` по умолчанию
  Why: стабильнее при изменении чанкинга/Chunk.ID.
  Tradeoff: может скрывать ошибки, когда важны конкретные чанки.
  Affects: `pkg/draftrag/eval/metrics.go`.
  Validation: тесты строятся на ParentID.

## Риски

- Неправильная трактовка “релевантности” (несколько правильных источников).
  Mitigation: в модели case поддержать `ExpectedParentIDs` как множество; метрики считают “первое попадание”.

## Проверка

- `go test ./...`
- Проверка корректности формул hit@k и MRR на синтетических кейсах.

## Соответствие конституции

- Нет конфликтов: аддитивный API, детерминированные тесты, без внешних зависимостей.

