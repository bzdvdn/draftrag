# Batch indexing Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для этой фичи.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/models.go | T1.1 |
| internal/application/pipeline.go | T1.2, T2.1, T2.2 |
| internal/application/batch_test.go | T3.1 |

## Фаза 1: Domain types и конфигурация

Цель: добавить типы для результата batch-индексации и расширить конфигурацию Pipeline.

- [x] T1.1 Добавить `IndexBatchResult` и `IndexBatchError` в domain — структуры с полями `Successful`, `Errors`, `ProcessedCount` и `DocumentID`, `Error` соответственно. Touches: internal/domain/models.go
- [x] T1.2 Расширить `PipelineConfig` полями `IndexConcurrency` (default: 4) и `IndexBatchRateLimit` (default: 10) — конфигурация для worker pool и rate limiter. Touches: internal/application/pipeline.go

## Фаза 2: Реализация IndexBatch

Цель: реализовать метод `IndexBatch` с worker pool и rate limiting.

- [x] T2.1 Реализовать `IndexBatch` с worker pool — метод принимает `ctx`, `docs`, `batchSize`, создаёт semaphore на `IndexConcurrency` workers, обрабатывает документы параллельно, возвращает `*IndexBatchResult` с successful docs и errors. Touches: internal/application/pipeline.go
- [x] T2.2 Добавить rate limiter в `IndexBatch` — token bucket с лимитом `IndexBatchRateLimit` calls/sec для вызовов `Embed()`, интеграция с существующими hooks. Touches: internal/application/pipeline.go

## Фаза 3: Проверка

Цель: доказать корректность через тесты и обеспечить покрытие всех AC.

- [x] T3.1 Добавить тесты для `IndexBatch` — тесты на параллельность (AC-001), rate limiting (AC-002), частичные ошибки (AC-003), отмену контекста (AC-004), интеграцию с Chunker (AC-005). Touches: internal/application/batch_test.go

## Покрытие критериев приемки

- AC-001 -> T1.2, T2.1, T3.1
- AC-002 -> T1.2, T2.2, T3.1
- AC-003 -> T2.1, T3.1
- AC-004 -> T2.1, T3.1
- AC-005 -> T2.1, T3.1

## Заметки

- T2.1 и T2.2 можно объединить в одну итерацию реализации, но разделены для ясности sequential vs concurrent concerns
- Worker pool реализован через semaphore и errgroup (или аналог) для управления concurrency
- Rate limiter — простой token bucket на основе time.Ticker или канала с токенами
