# Reranker: cross-encoder и Cohere Rerank

## Scope Snapshot

- In scope: production-ready reranker реализации для `domain.Reranker`, подключаемые через `PipelineOptions.Reranker`.
- Out of scope: обучение/дообучение cross-encoder моделей, замена существующего MMR reranker'а, sub-query decomposition.

## Цель

Разработчик RAG-системы на Go получает 2–3 готовых reranker'а, которые улучшают качество retrieval на 20–40% путём переранжирования top-K по более точной модели (cross-encoder или Cohere Rerank API). Без этого — единственным механизмом ранжирования остаётся косинусная близость embedding'а, что даёт ложные положительные срабатывания.

Ключевое архитектурное решение: batch-режим через опциональный интерфейс `BatchReranker`.
Pipeline проверяет `type assert` и при наличии нескольких query (MultiQuery) вызывает `RerankBatch` одним запросом вместо N последовательных `Rerank`.
Это снижает latency и стоимость API-вызовов.

Успех: после имплементации пользователь может подключить reranker одной строкой `Reranker: reranker.NewCohere(...)` и увидеть улучшение NDCG@10 в eval harness.

## Основной сценарий

1. Пользователь создаёт pipeline с `PipelineOptions{Reranker: myReranker}`.
2. При вызове `Search().TopK(20).Retrieve(ctx)` store возвращает 20 чанков по embedding similarity.
3. `maybeRerank` передаёт все 20 чанков + исходный запрос в reranker.
4. Reranker возвращает переранжированный список (top-20, но с новым порядком).
5. Дальше pipeline применяет MMR (если включён) и dedup, затем контекст обрезается до `MaxContextChunks`.
6. Ошибка reranker'а: возвращается `fmt.Errorf("reranker: %w", err)`, pipeline не падает.

## User Stories

- **P1 (MVP)**: Cohere Rerank API — внешний HTTP-клиент, ключ через опции, модель по умолчанию `rerank-english-v3.0`.
- **P2**: LLM-based reranker — использует существующий `LLMProvider` для попарного скоринга (zero-shot, без fine-tune). Медленнее, но не требует внешнего API.
- **P3**: Локальный cross-encoder через ONNX Runtime (экспериментально). Зависит от `github.com/yalue/onnxruntime_go`.

## MVP Slice

Cohere Rerank API + LLM-based reranker. ONNX cross-encoder — исследование.

## First Deployable Outcome

`TestPipeline_Reranker_IsCalled` проходит с `reranker.NewCohere(...)` вместо `reverseReranker`. Пользователь может переранжировать результаты pgvector/Qdrant запроса через Cohere.

## Scope

- Реализация `domain.Reranker` для Cohere Rerank API
- Реализация `domain.Reranker` через `LLMProvider` (zero-shot scoring)
- Конструкторы: `NewCohereRerank`, `NewLLMReranker`
- Опциональный интерфейс `BatchReranker` с методом `RerankBatch(ctx, []string, []RetrievedChunk) ([][]RetrievedChunk, error)`
- Pipeline-level интеграция: type-assert на `BatchReranker` в multi-query режиме → один batch-запрос вместо N отдельных
- Обработка ошибок: таймауты, недоступность API, пустой ответ
- Документация и пример в `examples/reranker/`
- Интеграция с eval harness: eval-скрипт показывает улучшение NDCG/MRR при использовании reranker'а

## Контекст

- `domain.Reranker` уже определён в `internal/domain/interfaces.go:86-88`
- `PipelineOptions.Reranker` уже есть (nil по умолчанию)
- `maybeRerank` в `internal/application/retrieval.go:12-22` вызывает reranker, если не nil
- MMR и dedup выполняются после reranker'а — финальный порядок: store → reranker → MMR → dedup
- Go 1.23, net/http, без внешних HTTP-фреймворков
- Все существующие store-реализации не изменяются
- MultiQuery (HyDE + MultiQuery) создаёт N query variants → pipeline rerank'ит каждый variant отдельно
- Batch-оптимизация: если reranker implements `BatchReranker`, pipeline вызывает один `RerankBatch` вместо N × `Rerank`
- MMR, dedup, обрезка контекста выполняются после reranker'а — batch не влияет на эти стадии

## Зависимости

- Cohere Rerank: любой Go HTTP-клиент (net/http стандартный). Внешних Go-зависимостей не требуется.
- LLM-reranker: зависит от `domain.LLMProvider` — уже есть в проекте.
- ONNX cross-encoder: потребует `github.com/yalue/onnxruntime_go` (cgo). Экспериментально, не в MVP.
- `none` для меж-спековых зависимостей.

## Требования

- RQ-001 Библиотека ДОЛЖНА предоставлять `NewCohereRerank(opts CohereRerankOptions) (*CohereReranker, error)`.
- RQ-002 `CohereReranker` ДОЛЖЕН реализовывать `domain.Reranker`.
- RQ-003 `CohereRerankOptions` ДОЛЖЕН содержать `APIKey` (обязательно), `Model` (по умолчанию `rerank-english-v3.0`), `BaseURL` (по умолчанию `https://api.cohere.com/v2`), `Timeout`.
- RQ-004 Библиотека ДОЛЖНА предоставлять `NewLLMReranker(llm LLMProvider, opts LLMRerankerOptions) (*LLMReranker, error)`.
- RQ-005 `LLMReranker` ДОЛЖЕН реализовывать `domain.Reranker`, используя переданный `LLMProvider` для zero-shot scoring.
- RQ-006 Оба reranker'а ДОЛЖНЫ возвращать исходное количество чанков, только с изменённым порядком (не фильтровать).
- RQ-007 При ошибке вызова API/LLM reranker ДОЛЖЕН возвращать error; pipeline пробрасывает её вызывающему коду.
- RQ-008 При пустом списке чанков reranker ДОЛЖЕН возвращать пустой список без ошибки (no-op).
- RQ-009 API-ключ Cohere НЕ ДОЛЖЕН логироваться в открытом виде (`RedactSecrets`).
- RQ-010 Документация ДОЛЖНА содержать пример использования с Cohere Rerank и LLM-reranker, включая сравнение производительности retrieval с/без reranker'а.
- RQ-011 Библиотека ДОЛЖНА определять опциональный интерфейс `BatchReranker` с методом `RerankBatch(ctx context.Context, queries []string, chunks []RetrievedChunk) ([][]RetrievedChunk, error)`.
- RQ-012 Pipeline ДОЛЖЕН проверять (`type assertion`) реализацию `BatchReranker` при multi-query поиске и вызывать `RerankBatch` вместо N последовательных `Rerank`.
- RQ-013 При отсутствии `BatchReranker` pipeline ДОЛЖЕН вызывать `Rerank` последовательно для каждого query (backward-compatible fallback).

## Вне scope

- ONNX Runtime cross-encoder (экспериментально, P3)
- Встраивание reranker'а внутрь store (reranker — отдельный слой после retrieval)
- Cohere Embed (только Rerank API)
- Многопоточная обработка внутри reranker'а (скоринг sequential, batch — сетевой)
- Поддержка triton inference server

## Критерии приемки

### AC-001 Cohere Rerank успешно переранжирует результаты

- Почему это важно: основная интеграция с популярным managed reranker'ом
- **Given** pipeline с pgvector store, 3 документа с разной релевантностью запросу, и `CohereReranker` с валидным API-ключом
- **When** вызывается `Search(query).TopK(3).Retrieve(ctx)`
- **Then** порядок чанков отличается от порядка по embedding similarity (документально более релевантный — выше)
- Evidence: тест сравнивает порядок до и после reranker'а в eval с callback

### AC-002 Cohere Rerank с пустыми чанками

- Почему это важно: краевой случай, не должен падать
- **Given** `CohereReranker` с валидным API-ключом
- **When** вызывается `Rerank(ctx, "query", []RetrievedChunk{})`
- **Then** возвращается пустой список без ошибки
- Evidence: `len(result) == 0`, `err == nil`

### AC-003 Cohere Rerank с невалидным ключом возвращает ошибку

- Почему это важно: пользователь должен видеть понятную ошибку конфигурации
- **Given** `CohereReranker` с пустым API-ключом
- **When** конструктор `NewCohereRerank`
- **Then** возвращается `ErrInvalidAPIKey` (или аналогичный sentinel)
- Evidence: `errors.Is(err, ErrInvalidAPIKey) == true`

### AC-004 LLM-reranker переранжирует через LLM

- Почему это важно: альтернатива без внешнего API, через已有的 LLM провайдер
- **Given** pipeline с in-memory store, 2 документа, `LLMReranker` с `OpenAICompatibleLLM`
- **When** вызывается `Search(query).TopK(2).Retrieve(ctx)`
- **Then** порядок чанков изменён согласно LLM-оценке
- Evidence: тест с mock LLM, который всегда ставит чанк с "B" выше "A", проверяет порядок

### AC-005 LLM-reranker при ошибке LLM возвращает ошибку

- Почему это важно: pipeline не должен молча глотать ошибки
- **Given** `LLMReranker` с LLM, которая возвращает ошибку на Generate
- **When** вызывается `Rerank(ctx, "query", chunks)`
- **Then** возвращается не-nil ошибка
- Evidence: `err != nil`

### AC-006 Cohere Rerank с невалидным ключом в runtime

- Почему это важно: ключ может быть непустым, но недействительным — пользователь должен получить ошибку, а не панику
- **Given** `CohereReranker` с синтаксически валидным, но недействительным API-ключом
- **When** вызывается `Rerank(ctx, "query", chunks)` и Cohere возвращает 401
- **Then** возвращается error, содержащая "unauthorized" или "401"
- Evidence: `err != nil && strings.Contains(err.Error(), "401")`

### AC-007 Reranker не фильтрует чанки

- Почему это важно: пользователь ожидает то же количество результатов, только в другом порядке
- **Given** любой reranker, 10 чанков на входе
- **When** вызывается `Rerank`
- **Then** возвращается ровно 10 чанков
- Evidence: `len(out) == len(in)`

### AC-008 Batch-режим: concurrent fan-out для нескольких query

- Почему это важно: при MultiQuery (3–5 вариантов) параллельный fan-out снижает latency в N раз (все запросы выполняются одновременно)
- **Given** `CohereReranker`, 5 query variants, 10 чанков
- **When** reranker реализует `BatchReranker` и вызывается `RerankBatch(ctx, queries, chunks)`
- **Then** все 5 HTTP-запросов к Cohere API выполняются конкурентно (errgroup), возвращается 5 наборов scores
- Evidence: тест с задержкой 100ms на каждый запрос проверяет, что общее время < 150ms (а не 500ms при последовательном вызове)

### AC-009 Fallback при отсутствии BatchReranker

- Почему это важно: старые/кастомные reranker'ы не должны ломаться
- **Given** `LLMReranker`, который реализует только `Reranker` (не `BatchReranker`)
- **When** pipeline работает в multi-query режиме
- **Then** pipeline вызывает `Rerank` последовательно для каждого query
- Evidence: тест с mock, который считает количество вызовов `Rerank` — их ровно столько же, сколько query variants

### AC-010 Документация с примером

- Почему это важно: пользователь должен знать о фиче и уметь её применить
- **Given** документация в `docs/en/reranker.md` и `docs/ru/reranker.md`
- **When** новый пользователь читает getting-started
- **Then** он видит раздел "Reranking" с примерами подключения Cohere и LLM-reranker'а
- Evidence: файлы существуют, содержат код с `NewCohereRerank` и `NewLLMReranker`

## Допущения

- API Cohere Rerank v2 (`https://api.cohere.com/v2/rerank`) стабилен и соответствует документации.
- LLM Provider способен выполнять скоринг релевантности по инструкции (zero-shot).
- Пользователи, не подключающие reranker, не видят изменений производительности (nil-guard).
- API-ключи хранятся вне кода (env-переменные, secrets manager).

## Критерии успеха

- SC-001 Использование Cohere Rerank повышает NDCG@10 на ≥15% относительно baseline (embedding-only) на встроенном тестовом наборе eval harness (10 запросов, 3 релевантных документа на запрос).
- SC-002 P95 latency Cohere Rerank для 20 чанков ≤ 500ms (при условии сетевой задержки ≤ 100ms).

## Краевые случаи

- Один чанк на входе: reranker возвращает его же без изменений.
- Все чанки одинаково релевантны: порядок может не измениться (но не ошибка).
- Таймаут Cohere API: error, не паника.
- Cohere возвращает score с плавающей точкой: сортировка по score desc.
- LLM возвращает пустой/непарсируемый ответ: fallback на исходный порядок + error в лог.

## Открытые вопросы

- Оставить LLM-reranker как alpha (P2) или включить в MVP? — Пока P2, вне MVP.

## Принятые решения

- DEC-001: reranker'ы выносятся в отдельный пакет `pkg/draftrag/reranker/`. CohereReranker и BatchReranker — в `pkg/draftrag/reranker/cohere.go`. LLMReranker — в `pkg/draftrag/reranker/llm.go`. Это изолирует HTTP-клиент Cohere от основного API.
