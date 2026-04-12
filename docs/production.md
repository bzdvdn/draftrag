# Production checklist + runbook

Этот документ — **стартовые практики**, а не гарантия SLO/latency. Цель: дать короткие, проверяемые шаги “что сделать до релиза” и “что делать при инциденте”.

## Checklist (перед релизом)

1. **Везде используйте `context.Context` и таймауты.** У каждого пути (`Index`, `Retrieve/Answer/Cite`, миграции, операции коллекций) должен быть `context.WithTimeout(...); defer cancel()`. См. `docs/pipeline.md`.
2. **Retry + Circuit Breaker включены на внешних API.** Для LLM/Embedder используйте `NewRetryLLMProvider` / `NewRetryEmbedder` и задайте разумные лимиты (max retries, backoff, CB threshold/timeout). См. `README.md` → “Retry + Circuit Breaker”.
3. **Кеш эмбеддингов включён.** `NewCachedEmbedder` (L1 LRU) и (опционально) Redis L2 для повторяющихся запросов. См. `README.md` → “Кэширование эмбеддингов” и “Redis L2”.
4. **Ограничения/лимиты заданы явно.** `DefaultTopK`, лимиты контекста (`MaxContextChars/Chunks`), `IndexBatch` concurrency/rate limit — зафиксированы и протестированы под ваш трафик.
5. **pgvector: миграции выполняются отдельным шагом деплоя.** Не запускайте DDL “на старте сервиса”; используйте `MigratePGVector/SetupPGVector` в deploy job/init container. См. `pkg/draftrag/pgvector.go` и `pkg/draftrag/pgvector_migrations.md`.
6. **pgvector: runtime timeouts/limits настроены.** `PGVectorRuntimeOptions` (Search/Upsert/Delete timeouts, MaxTopK, MaxParentIDs) соответствуют вашему latency budget.
7. **Qdrant/Weaviate: коллекции подготовлены.** Либо auto-create в deploy job, либо явная проверка `CollectionExists/CreateCollection` (Qdrant) / `WeaviateCollectionExists/CreateWeaviateCollection` (Weaviate). Размерность векторов совпадает с embedder.
8. **Observability включена.** Минимум: hooks по стадиям (chunking/embed/search/generate). Лучше: OTel hooks (`pkg/draftrag/otel`) и метрики/трейсы в вашу систему. См. `README.md` → “Observability hooks”.
9. **Логи безопасны.** Не логируйте сырые документы/запросы без своей политики. Убедитесь, что секреты (APIKey/bearer token) не попадают в ошибки/логи (best-effort redaction со стороны библиотеки). См. `README.md` → “Redaction и безопасность логов”.
10. **Прогон качества retrieval (eval) выполнен.** Проверьте Hit@K/MRR на ваших кейсах, зафиксируйте baseline перед релизом. См. `README.md` → “Eval harness”.
11. **Проверка регрессий.** `go test ./...` зелёный; минимальный smoke-тест индексации/поиска прогнан на staging.

## Backend notes (что важно в эксплуатации)

| Backend | Что важно | Типовые ошибки |
|---|---|---|
| PostgreSQL + pgvector | DDL миграции отдельно от сервиса; runtime таймауты; права на `CREATE EXTENSION` | permission denied, долгие DDL, dimension mismatch |
| Qdrant | коллекция существует; размерность совпадает; HTTP timeout | 404/collection missing, dimension mismatch, timeouts |
| Weaviate | class/schema существует; APIKey в headers; HTTP timeout | 401/403, schema errors, timeouts |

## Runbook (инциденты)

Ниже “быстро” = **коротко и по шагам**.

### 1) Пустая выдача / низкий recall

**Symptoms**
- `Retrieve/Answer` возвращает 0 источников или нерелевантный контекст.

**Checks**
- Индексация действительно выполнялась (количество документов/чанков).
- Размерность embedder совпадает с хранилищем (pgvector dimension, Qdrant/Weaviate dimension).
- `TopK` не слишком мал; нет ли `Filter/ParentIDs`, которые “отсекают всё”.
- Hooks/OTel: стадия `search` не возвращает ошибки и не слишком быстрая/пустая.

**Actions**
- Увеличить `TopK`, временно убрать фильтры, проверить запрос.
- Пересобрать индекс (если меняли embedder model/dimension).
- Для pgvector: убедиться, что миграции применены, индексы созданы.

### 2) Рост latency (p95/p99) или таймауты

**Symptoms**
- `context deadline exceeded`, деградация ответов, рост p95.

**Checks**
- Hooks/OTel: какая стадия выросла (`embed`, `search`, `generate`).
- Не открывается ли circuit breaker (рост `CB open` / `retry attempt failed`).
- pgvector/Qdrant/Weaviate: сетевые timeouts, очередь соединений (pool), перегруз БД/кластера.

**Actions**
- Увеличить таймауты только там, где нужно (например, `generate`), но держать общий budget.
- Снизить concurrency (`IndexBatch`, параллелизм запросов), включить/увеличить кеш эмбеддингов.
- Для pgvector: проверить план запроса/индексы, `MaxTopK`, размер контента/чанков.

### 3) Circuit breaker “open” / всплеск ретраев

**Symptoms**
- Ошибки вида “circuit breaker: open”, рост ретраев, ответы нестабильны.

**Checks**
- Тип ошибок: rate limit/5xx/timeout vs non-retryable.
- Нагрузка: одновременные запросы, burst, прогрет ли кеш.
- В логах retry/cache не должно быть секретов (redaction best-effort).

**Actions**
- Уменьшить throughput (ограничить concurrency), увеличить backoff/jitter, поднять `CBTimeout` для “охлаждения”.
- Если rate limit: добавить rate limiting на стороне сервиса, калибровать `MaxRetries`.
- Если постоянные 4xx: сделать ошибку non-retryable (на стороне провайдера/конфигурации), исправить ключ/модель/endpoint.

### 4) pgvector миграции не применяются / ошибки прав

**Symptoms**
- Ошибки DDL, отсутствуют таблицы/индексы, permission denied.

**Checks**
- Миграции запускаются отдельным шагом (deploy job/init container), а не при старте сервиса.
- Роль БД имеет права на DDL и (если нужно) `CREATE EXTENSION vector`.
- Таймаут миграций достаточный (DDL может быть долгим).

**Actions**
- Перенести миграции в отдельный шаг деплоя; разделить права runtime vs migrate.
- Включать `CreateExtension` только если вы уверены в правах/политике.

### 5) Qdrant/Weaviate: “collection/class missing” или dimension mismatch

**Symptoms**
- 404/collection missing, schema errors, dimension mismatch.

**Checks**
- Коллекция/класс создаётся до старта сервиса (или первый запрос делает create).
- `Dimension` совпадает с текущей embedding моделью.

**Actions**
- Создать коллекцию/класс через deploy job, зафиксировать dimension как константу окружения.
- При смене embedder модели: пересоздать коллекцию/переиндексировать данные.

## Security/Redaction

- draftRAG best-effort редактирует **известные библиотеке** секреты (например, `APIKey`/bearer token из options) из сообщений ошибок, которые она формирует.
- draftRAG **не делает** автоматическое обнаружение PII в произвольном тексте.
- Ответственность пользователя: не логировать сырые документы/запросы без своей политики (редакции/маскирования/retention).

