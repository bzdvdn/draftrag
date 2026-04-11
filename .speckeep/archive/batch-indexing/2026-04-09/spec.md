# Batch indexing

## Scope Snapshot

- In scope: метод `IndexBatch` с параллельным эмбеддингом, worker pool и rate limiting для эффективной индексации больших массивов документов
- Out of scope: streaming-индексация, транзакционная целостность batch-операций, distributed locking

## Цель

Пользователи библиотеки получают возможность эффективно индексировать большие объёмы документов с контролируемой concurrency. Успех виден по снижению времени индексации batch'ей при сохранении стабильности и предсказуемости работы с rate limiting.

## Основной сценарий

1. Пользователь формирует слайс из 1000+ документов для индексации
2. Вызывает `pipeline.IndexBatch(ctx, docs, batchSize)` с указанием размера batch'а
3. Worker pool с ограниченной concurrency обрабатывает документы параллельно
4. Rate limiter регулирует частоту вызовов Embedder API
5. Метод возвращает результат с информацией об успешно проиндексированных документах и ошибках

## Scope

- Метод `IndexBatch(ctx context.Context, docs []Document, batchSize int) (*IndexBatchResult, error)` в `Pipeline`
- Worker pool с настраиваемым количеством workers
- Rate limiting для вызовов `Embedder.Embed()`
- Обработка частичных ошибок с детализацией какие документы упали
- Интеграция с существующим `PipelineConfig` через новое поле `IndexConcurrency`

## Контекст

- Существующий `Pipeline.Index()` обрабатывает документы последовательно — bottleneck при больших объёмах
- `Embedder` интерфейс имеет метод `Embed(ctx, text string)` — вызовы могут быть дорогими и rate-limited провайдером
- `PipelineConfig` уже содержит hooks и настройки — новая concurrency-настройка должна быть опциональной с разумным default
- Chunking при batch-индексации сохраняет текущее поведение: если `Chunker` задан — каждый документ разбивается на чанки

## Требования

- RQ-001 `IndexBatch` должен обрабатывать документы параллельно с ограничением concurrency
- RQ-002 Worker pool должен иметь настраиваемый размер через `PipelineConfig.IndexConcurrency` (default: 4 workers)
- RQ-003 Rate limiter должен ограничивать количество вызовов `Embed` в единицу времени (default: 10 calls/sec)
- RQ-004 Метод должен возвращать `IndexBatchResult` со списком успешных документов и ошибок по документам
- RQ-005 При отмене контекста метод должен останавливать обработку и возвращать partial results
- RQ-006 Ошибки эмбеддинга одного документа не должны прерывать обработку batch'а

## Вне scope

- Транзакционная гарантия all-or-nothing для batch-индексации
- Retry-логика с exponential backoff для failed embeds
- Distributed/coordinated rate limiting между несколькими инстансами
- Потоковая индексация (streaming/chunked input) — только batch из []Document
- Балансировка нагрузки между несколькими embedder-эндпоинтами

## Критерии приемки

### AC-001 Параллельная обработка документов

- Почему это важно: сокращение времени индексации при больших объёмах
- **Given** массив из 100 документов и настроенный `IndexConcurrency = 5`
- **When** вызывается `IndexBatch(ctx, docs, 10)`
- **Then** документы обрабатываются параллельно с не более чем 5 одновременными goroutines
- Evidence: время индексации примерно в 4-5 раз меньше последовательной обработки (измерено через hooks или benchmarks)

### AC-002 Rate limiting вызовов Embedder

- Почему это важно: предотвращение 429 ошибок от embedder-провайдеров
- **Given** `IndexBatchRateLimit = 10` calls/sec и 20 документов
- **When** вызывается `IndexBatch(ctx, docs, 20)`
- **Then** все 20 embed-вызовов выполняются не быстрее чем за ~2 секунды с равномерным распределением
- Evidence: наблюдение через hooks StageStart/StageEnd для Embed стадии — интервалы между вызовами ≥100ms

### AC-003 Обработка частичных ошибок

- Почему это важно: пользователь должен знать какие документы не проиндексированы
- **Given** массив из 5 документов, где embedder вернёт ошибку для 2 из них
- **When** вызывается `IndexBatch(ctx, docs, 5)`
- **Then** метод возвращает `*IndexBatchResult` с 3 успешными документами и 2 ошибками с идентификацией failed документов
- Evidence: `result.Successful` содержит 3 документа, `result.Errors` содержит 2 ошибки с `DocumentID`

### AC-004 Отмена через контекст

- Почему это важно: корректное поведение при timeout/cancellation
- **Given** контекст с timeout 100ms и batch из 50 медленных документов
- **When** вызывается `IndexBatch(ctx, docs, 10)`
- **Then** метод возвращает `context.DeadlineExceeded` и `*IndexBatchResult` с частично обработанными документами
- Evidence: `result.ProcessedCount` > 0 и < 50, ошибка содержит `context.DeadlineExceeded`

### AC-005 Интеграция с Chunker

- Почему это важно: сохранение существующего поведения Pipeline
- **Given** Pipeline с настроенным `Chunker` и batch из 3 документов, каждый разбивается на 2 чанка
- **When** вызывается `IndexBatch(ctx, docs, 3)`
- **Then** все 6 чанков индексируются, concurrency применяется к уровню документов (а не чанков)
- Evidence: hooks показывают 6 StageEnd событий для HookStageChunking, 6 для HookStageEmbed, 6 для Upsert

## Допущения

- Embedder-провайдеры (OpenAI, Anthropic и т.д.) имеют rate limits; default 10 calls/sec — разумный conservative default
- Пользователи предпочитают throughput латентности при batch-индексации
- Worker pool на уровне документов (не чанков) — приемлемый баланс простоты и производительности
- Частичные ошибки — нормальный сценарий, пользователь может повторно индексировать failed документы
- Default concurrency = 4 подходит для большинства embedder-провайдеров без риска rate limit errors

## Критерии успеха

- SC-001 Batch из 1000 документов индексируется за <30 секунд при `IndexConcurrency=4` и `IndexBatchRateLimit=10` (vs >120 секунд при последовательной обработке)
- SC-002 Error rate при batch-индексации остаётся на уровне 0% при настроенном rate limit (нет 429/Too Many Requests от провайдеров)

## Краевые случаи

- Пустой слайс документов: возвращается пустой `IndexBatchResult` без ошибок
- `batchSize <= 0`: используется default batchSize (равный `IndexConcurrency`)
- Все документы падают с ошибками: возвращается `IndexBatchResult` с пустым `Successful` и заполненным `Errors`
- Гонка при одновременных Upsert'ах в VectorStore: ответственность VectorStore-реализации (наша задача — вызывать Upsert корректно)

## Открытые вопросы

- none
