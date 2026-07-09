# Batch indexing План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи.
Outputs: plan, data model.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Добавить метод `IndexBatch` в application.Pipeline с worker pool для параллельной индексации документов, rate limiting для embedder-вызовов и структурированным результатом с частичными ошибками.

## Scope

- Расширение `internal/application/pipeline.go`: новый метод `IndexBatch` и вспомогательные типы
- Расширение `PipelineConfig`: поля `IndexConcurrency` и `IndexBatchRateLimit`
- Новый тип `IndexBatchResult` в `internal/domain/models.go` для возврата результатов batch-операции
- Внутренняя реализация worker pool и rate limiter в `internal/application/` (не публичный API)

## Implementation Surfaces

- `internal/application/pipeline.go` — существующий файл, добавляется метод `IndexBatch` и интеграция с hooks
- `internal/application/` — новый файл `batch.go` (или встроенно в pipeline.go) с реализацией worker pool и rate limiter
- `internal/domain/models.go` — существующий файл, добавляется тип `IndexBatchResult` и `IndexBatchError`
- `internal/application/config.go` (если существует) или расширение `PipelineConfig` в `pipeline.go` — новые поля конфигурации

## Влияние на архитектуру

- Локальное изменение: добавление метода в application layer без изменения domain interfaces
- Новая capability Pipeline, не ломающая существующий API — метод `Index` остаётся без изменений
- Worker pool живёт внутри вызова `IndexBatch` — нет shared state между вызовами
- Rate limiter не влияет на другие методы Pipeline (Query, Answer и т.д.)

## Acceptance Approach

- AC-001 (параллельная обработка) -> реализация worker pool с semaphore на `IndexConcurrency` workers; observable через hooks — количество одновременных Embed-вызовов не превышает `IndexConcurrency`
- AC-002 (rate limiting) -> token bucket rate limiter с лимитом `IndexBatchRateLimit`; observable через hooks — интервалы между StageStart событиями для Embed ≥ 1/RateLimit
- AC-003 (частичные ошибки) -> `IndexBatchResult` со слайсами `Successful []Document` и `Errors []IndexBatchError`; каждый `IndexBatchError` содержит `DocumentID` и `Error`
- AC-004 (отмена контекста) -> проверка `ctx.Err()` в worker loop, возврат partial result с уже обработанными документами
- AC-005 (интеграция с Chunker) -> сохранение существующей логики chunking внутри worker'а; observable через hooks — StageStart/StageEnd для Chunking вызываются для каждого документа

## Данные и контракты

- AC-003 требует новой сущности `IndexBatchResult` в domain слое для возврата результатов
- API boundaries не меняются — публичный API пакета остаётся стабильным
- Event contracts не меняются — hooks уже существуют и используются
- Data model: добавляется `IndexBatchResult` и `IndexBatchError` в `internal/domain/models.go`
- Контракты не требуются — это внутренняя capability Pipeline

## Стратегия реализации

- DEC-001 Worker pool на уровне документов (не чанков)
  Why: chunking может породить разное количество чанков на документ, что делает балансировку на уровне чанков сложной; на уровне документов проще гарантировать `IndexConcurrency` и отслеживать partial results
  Tradeoff: если документ порождает много чанков, один worker будет занят долго; приемлемо для типичных сценариев
  Affects: `internal/application/pipeline.go`, `internal/application/batch.go`
  Validation: тест на параллельность — 10 документов с задержкой 100ms каждый, concurrency=5, общее время ~200ms (не 1000ms)

- DEC-002 Token bucket rate limiter внутри `IndexBatch`
  Why: embedder-провайдеры (OpenAI, Anthropic) имеют rate limits; token bucket даёт равномерное распределение вызовов
  Tradeoff: добавляет латентность при высоком rate limit, но предотвращает 429 ошибки
  Affects: `internal/application/batch.go`
  Validation: тест на rate limiting — 20 embed-вызовов с лимитом 10/sec должны занять ≥2 секунды

- DEC-003 Сохранение существующего `Index` без изменений
  Why: backward compatibility; пользователи могут выбирать между простым `Index` (последовательный) и `IndexBatch` (параллельный с rate limiting)
  Tradeoff: дублирование логики chunking+embedding между методами
  Affects: `internal/application/pipeline.go`
  Validation: существующие тесты `Index` продолжают проходить

- DEC-004 Результат ошибки содержит DocumentID
  Why: пользователь должен знать какие документы упали для retry
  Tradeoff: требует передачи DocumentID через worker channel
  Affects: `internal/domain/models.go` — новый тип `IndexBatchError`
  Validation: тест на partial errors — результат содержит ошибки с идентификацией документов

## Incremental Delivery

### MVP (Первая ценность)

- Расширение `PipelineConfig` с `IndexConcurrency`
- Реализация `IndexBatch` с worker pool (без rate limiting)
- Тип `IndexBatchResult` с фиксацией успешных документов
- Тесты покрывающие AC-001, AC-003 (частично), AC-005

Критерий готовности MVP: `IndexBatch` работает параллельно, возвращает результат, интеграция с Chunker сохранена.

### Итеративное расширение

- Добавить `IndexBatchRateLimit` в `PipelineConfig`
- Реализовать token bucket rate limiter
- Добавить `Errors []IndexBatchError` в `IndexBatchResult` для полного покрытия AC-003
- Тесты на AC-002, AC-004

## Порядок реализации

1. Добавить `IndexBatchResult` и `IndexBatchError` в `internal/domain/models.go`
2. Расширить `PipelineConfig` полями `IndexConcurrency` и `IndexBatchRateLimit`
3. Реализовать `IndexBatch` в `internal/application/pipeline.go` (worker pool + базовая логика)
4. Добавить rate limiter внутрь `IndexBatch`
5. Добавить тесты для всех AC

Пункты 1-2 можно делать параллельно. Пункт 3 зависит от 1-2. Пункт 4 зависит от 3.

## Риски

- Риск: worker pool может породить слишком много goroutines при большом batch
  Mitigation: semaphore на `IndexConcurrency` ограничивает одновременные workers; каждый worker обрабатывает один документ

- Риск: rate limiting может быть сложным для тестирования с внешними embedder
  Mitigation: rate limiter — внутренняя деталь, тестируется через mock embedder с задержками

- Риск: partial results при отмене контекста могут быть неполными
  Mitigation: сохранение результата в защищённой структуре с mutex, проверка `ctx.Err()` перед каждым embed-вызовом

## Rollout и compatibility

- Новый метод `IndexBatch` — добавление, не breaking change
- Существующий `Index` остаётся без изменений — полная backward compatibility
- Новые поля в `PipelineConfig` — optional с defaults (IndexConcurrency=4, IndexBatchRateLimit=10)
- Не требуется migration, backfill или feature flag

## Проверка

- Тест: `IndexBatch` с 10 документами, concurrency=5, embedder с задержкой — проверка времени < 3*delay (AC-001)
- Тест: `IndexBatch` с rate limit 10/sec, 20 документов — проверка duration ≥ 2 секунд (AC-002)
- Тест: `IndexBatch` с 5 документами, 2 падают с ошибкой — проверка `result.Successful`=3, `result.Errors`=2 с DocumentID (AC-003)
- Тест: `IndexBatch` с timeout context, 50 документов — проверка `context.DeadlineExceeded` и `result.ProcessedCount` < 50 (AC-004)
- Тест: `IndexBatch` с настроенным Chunker — проверка StageStart/StageEnd hooks для каждого чанка (AC-005)
- `go vet`, `go fmt`, `golangci-lint` без ошибок

## Соответствие конституции

- [Интерфейсная абстракция] — `IndexBatch` использует существующие `Embedder`, `VectorStore`, `Chunker` интерфейсы; новые типы (`IndexBatchResult`) — domain модели
- [Чистая архитектура] — worker pool и rate limiter живут в application слое; domain не знает о concurrency
- [Минимальная конфигурация] — `PipelineConfig` получает optional поля с разумными defaults
- [Контекстная безопасность] — `context.Context` передаётся в `IndexBatch` и проверяется в worker'ах
- [Тестируемость] — все компоненты тестируются через существующие интерфейсы и моки

Нет конфликтов с конституцией.
