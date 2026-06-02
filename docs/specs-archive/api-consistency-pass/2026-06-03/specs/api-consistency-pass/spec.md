# api-consistency-pass: архитектурный hardening RAG-pipeline

## Scope Snapshot

- In scope: 8 замечаний из Senior Go Architect review — катастрофическая дубликация в SearchBuilder, нетипизированные ошибки в application, мисайнменджмент `mapValidationErr`, последовательный `Index`, неатомарный `UpdateDocument`, отсутствие backpressure в стриминге, fixed-rate вместо token-bucket в `IndexBatch`, отставание документации (README + capability-таблица) от кода.
- Out of scope: добавление гибридного поиска в ChromaDB/Weaviate/Milvus (отдельная фича); eval-harness метрики качества генерации (faithfulness/context relevance — отдельная фича); изменения в публичных интерфейсах embedder cache (Redis L2 контракт стабилен); производительность `wrapStreamWithHook` ниже текущего уровня (текущее поведение — best-effort zero-copy, см. RQ-007).

## Цель

Устранить системные долги, выявленные в code review draftRAG v0.1.0, до того как добавление новой стратегии retrieval (RAG-fusion, step-back, self-RAG) сделает 7×6 матрицу роутинга `SearchBuilder` неуправляемой. Бенефициары: мейнтейнеры (снижение cognitive load), пользователи библиотеки (типизированные sentinel-ошибки, документированная capability-матрица), продакшн-операторы (atomic UpdateDocument, bounded memory в стриминге). Метрика успеха: добавление новой retrieval-стратегии требует ≤ 2 точек изменения в исходном коде вместо текущих 7.

## Основной сценарий

1. **Стартовая точка:** репозиторий на ветке `feature/api-consistency-pass`, `go build ./...` и `go vet ./...` зелёные, ветка `master` зафиксирована в v0.1.0.
2. **Ход работы:** серия фокусных изменений по 8 RQ; каждое закрывается тестом из acceptance criteria; `golangci-lint run ./...` остаётся зелёным; покрытие `internal/domain` и `internal/application` не падает ниже текущих 100% / 83.3%.
3. **Результат:** `go test ./...` зелёный; новый retrieval-стратегия может быть добавлена изменением ≤ 2 файлов; `errors.Is(err, draftrag.ErrEmptyQuery)` работает в пользовательском коде; `docs/vector-stores.md` содержит capability-таблицу.
4. **Fallback-путь:** если конкретное RQ блокируется внешним ограничением (например, отсутствие транзакций в ChromaDB для RQ-005), RQ переходит в "partial" статус с явным документированием degraded-поведения, не молча.

## User Stories

- P1 Story: разработчик библиотеки добавляет новую retrieval-стратегию (например, HyDE-parent-documents fusion) за один рабочий день без копи-пасты 42 if-блоков.
- P2 Story: пользователь библиотеки в production-сервисе ловит ошибку через `errors.Is(err, draftrag.ErrEmptyQuery)` и возвращает 400 Bad Request, не парся строки.
- P3 Story: оператор читает `docs/vector-stores.md` и за 30 секунд понимает, какие stores поддерживают hybrid search и DeleteByParentID, без чтения исходного кода.
- Если grouping не помогает в brownfield-сценарии, оставлено как есть.

## MVP Slice

Минимальный независимо поставляемый срез — критические замечания (RQ-001, RQ-002, RQ-003). Без них остальные RQ строятся на шатком фундаменте: добавлять RQ-004 (Index concurrency) поверх неконсистентных ошибок = удвоение работы. MVP закрывает AC-001, AC-002, AC-003, AC-004, AC-005, AC-006.

## First Deployable Outcome

После MVP-слайса (RQ-001..RQ-003) можно сделать промежуточный коммит и ревью — изменения локализованы в `internal/application/` (3 файла) + `pkg/draftrag/search.go` + `pkg/draftrag/errors.go`. Остальные RQ могут поставляться отдельными коммитами поверх. Если хотя бы один RQ из MVP блокируется — фича не поставляется, требуется amend.

## Scope

- `pkg/draftrag/search.go` — рефакторинг роутинга (RQ-001)
- `internal/application/answer.go`, `query.go`, `pipeline.go`, `stream.go`, `batch.go` — замена `errors.New` на типизированные sentinel'ы (RQ-002)
- `pkg/draftrag/draftrag.go` — переименование/упрощение `mapValidationErr` (RQ-003)
- `internal/application/pipeline.go` — переиспользование worker pool из `batch.go` в `Index` (RQ-004)
- `internal/application/pipeline.go` + `pkg/draftrag/pgvector.go` + интерфейс `DocumentStore` — атомарность `UpdateDocument` для транзакционных хранилищ (RQ-005)
- `internal/application/stream.go` — bounded buffer для streaming-канала (RQ-006)
- `internal/application/batch.go` + `pkg/draftrag/draftrag.go` — флаг `IndexBatchRateLimitPerWorker` + документация (RQ-007)
- `README.md`, `docs/vector-stores.md`, `ROADMAP.md` — синхронизация документации (RQ-008)

## Контекст

- Репозиторий в v0.1.0, единственная активная feature-ветка — текущая (`feature/api-consistency-pass`).
- Конституция требует: Clean Architecture, capability-интерфейсы, типизированные sentinel-ошибки с `errors.Is`, контекст во всех публичных операциях, 100% покрытие domain и ≥80% application.
- Текущее покрытие (по CHANGELOG): `internal/domain` 100%, `internal/application` 83.3%, `internal/infrastructure/vectorstore` 60%.
- В коде присутствуют маркеры `@sk-task hardening-2026q2#T1.1` (R1) и `@sk-task hardening-2026q2#T3.2` (R3) — это исторические маркеры из не-завершённой ранее работы. Новые маркеры будут `@sk-task api-consistency-pass#...`.
- `HybridConfig.BMFinaKK` уже переименован в `BMFinalK` (см. `docs/specs-archive/api-resilience-fixes/`).
- `CHANGELOG.md` секция `[Unreleased]` пуста; новые изменения пойдут туда.
- AGENTS.md запрещает `git commit/push` без явной просьбы; все коммиты — ручные.
- Weaviate и Milvus реализованы в `internal/infrastructure/vectorstore/`, но не упомянуты в README — это создаёт репутационный риск и регрессию discoverability.

## Требования

### RQ-001 Рефакторинг роутинга в SearchBuilder

`pkg/draftrag/search.go` ДОЛЖЕН быть реорганизован так, чтобы публичные методы `Retrieve`, `Answer`, `Cite`, `InlineCite`, `Stream`, `StreamSources`, `StreamCite` не содержали 6 повторяющихся веток роутинга (basic, HyDE, MultiQuery, Hybrid, ParentIDs, Filter). Маршрутизация ДОЛЖНА быть вынесена в один селектор (например, `selectRetrieval(ctx) retrievalFn` и `selectGeneration(retrievalFn) genFn`), а 7 публичных методов — сведены к 2-3 строкам делегирования. Поведение ДОЛЖНО остаться идентичным; все существующие тесты в `pkg/draftrag/search_test.go` и `search_builder_test.go` ДОЛЖНЫ проходить без изменений в asserts.

### RQ-002 Типизированные sentinel-ошибки в application

В `internal/application/*.go` НЕ ДОЛЖНО быть вызовов `errors.New("question is empty")`, `errors.New("topK must be > 0")`, `errors.New("query is empty")` и подобных inline-ошибок. Все такие ошибки ДОЛЖНЫ быть заменены на `fmt.Errorf("%w: ...", domain.ErrEmptyQueryText, ...)` или `fmt.Errorf("%w: ...", domain.ErrInvalidQueryTopK, ...)`. Публичные sentinel'ы `ErrEmptyQuery` и `ErrInvalidTopK` в `pkg/draftrag/errors.go` ДОЛЖНЫ стать достижимыми через `errors.Is` для пользователя.

### RQ-003 Корректный error mapping в публичном API

`pkg/draftrag/draftrag.go` ДОЛЖЕН содержать ровно одну функцию маппинга ошибок (имя и поведение — на усмотрение реализатора, но она ДОЛЖНА отражать фактическую работу). Все sentinel-ошибки, упомянутые в публичном API (`ErrDeleteNotSupported`, `ErrStreamingNotSupported`, `ErrFiltersNotSupported`, `ErrHybridNotSupported`, `ErrEmptyQuery`, `ErrInvalidTopK`, `ErrEmptyDocument`, `ErrEmbeddingDimensionMismatch`), ДОЛЖНЫ быть достижимы через `errors.Is` при их возникновении в `application` слое. Сейчас, например, `pipeline.UpdateDocument` при отсутствии `DocumentStore` capability возвращает сырой `application.ErrDeleteNotSupported` без маппинга в публичный `draftrag.ErrDeleteNotSupported`.

### RQ-004 Конкурентный Index

`Pipeline.Index(ctx, docs)` ДОЛЖЕН использовать тот же worker pool, что и `Pipeline.IndexBatch`, с настройкой concurrency из `PipelineOptions.IndexConcurrency`. Поведение ДОЛЖНО быть эквивалентно вызову `IndexBatch(ctx, docs, IndexConcurrency)` плюс возврат `error` от первой критической ошибки (first-error semantics), либо `IndexBatch` без partial results — на усмотрение реализатора, при условии документирования контракта. Существующие тесты `Index` ДОЛЖНЫ проходить без изменения asserts.

### RQ-005 Атомарность UpdateDocument

`Pipeline.UpdateDocument(ctx, doc)` ДОЛЖЕН быть атомарен для транзакционных хранилищ (pgvector): при ошибке переиндексации старые чанки ДОЛЖНЫ быть восстановлены. Контракт `DocumentStore` ДОЛЖЕН быть расширен опциональным capability-интерфейсом (например, `TransactionalDocumentStore` с методами `Begin(ctx)`, `DeleteByParentIDInTx`, `UpsertInTx`, `Commit`/`Rollback`) — без ломки существующих реализаций. Для не-транзакционных хранилищ (Qdrant, ChromaDB, Weaviate, Milvus, in-memory) ДОЛЖНО быть документировано best-effort поведение с явным возвратом ошибки `ErrUpdateNotAtomic` (новый sentinel) при сбое после успешного delete.

### RQ-006 Bounded backpressure в streaming

`Pipeline.AnswerStream*` методы ДОЛЖНЫ использовать канал с настраиваемым буфером (опция `PipelineOptions.StreamBufferSize`, default 8) вместо текущего небуферизованного канала в `wrapStreamWithHook`. Медленный потребитель НЕ ДОЛЖЕН приводить к неограниченному росту памяти; при заполнении буфера горутина-производитель ДОЛЖНА блокироваться или отменяться по `ctx.Done()`. Поведение при `StreamBufferSize=0` ДОЛЖНО быть документировано (предлагается: 0 → unbounded warning, либо явный fallback на текущее небуферизованное поведение).

### RQ-007 Rate-limiter semantics

`PipelineOptions.IndexBatchRateLimit` ДОЛЖЕН получить второй параметр `IndexBatchRateLimitPerWorker bool` (default `false`). При `true` rate-limiter ДОЛЖЕН быть per-worker (token-bucket с refill `rateLimit` per worker); при `false` — текущее поведение (shared fixed-rate, документировано как "общий лимит на пул"). В README и `docs/production.md` ДОЛЖНО быть явно указано, что default — shared (это сознательный выбор, не баг).

### RQ-008 Синхронизация документации

`README.md` ДОЛЖЕН перечислять все 6 реализованных vector stores: in-memory, pgvector, qdrant, chromadb, weaviate, milvus. `docs/vector-stores.md` (или новый `docs/capability-matrix.md`) ДОЛЖЕН содержать capability-таблицу: строки = stores, колонки = `Basic retrieval`, `Metadata filter`, `ParentID filter`, `Hybrid search (BM25+semantic)`, `DeleteByParentID`, `Collection management`, `Streaming` (не применимо для stores). `ROADMAP.md` секция "Приоритет 2 → Additional vector stores" ДОЛЖНА быть обновлена: Weaviate и Milvus перенесены из "планируется" в "реализовано".

## Вне scope

- Добавление hybrid search в ChromaDB/Weaviate/Milvus — каждое требует отдельной feature-спецификации.
- Eval harness метрики качества генерации (faithfulness, context relevance, answer relevance) — отдельная фича.
- Изменение публичного контракта `RetryOptions`, `CacheOptions`, `PipelineOptions.Stream*` сверх объёма RQ-006 и RQ-007.
- Рефакторинг `internal/infrastructure/vectorstore/memory.go` (296 строк) — спекулятивная чистка, не блокирует v0.2.0.
- Оптимизация `buildContextTextV1` (двойной `[]rune(line)`) — микрооптимизация, не влияет на наблюдаемое поведение.
- `golangci-lint` baseline update (если появятся новые срабатывания) — отдельный housekeeping-коммит.
- Замена legacy `@sk-task hardening-2026q2#T1.1` маркеров на новые `@sk-task api-consistency-pass#...` — может быть выполнена как часть RQ-001 (sed-проход по touched-файлам), но не требует отдельного AC.

## Критерии приемки

### AC-001 SearchBuilder: ≤ 2 точек изменения для новой стратегии

- **Why:** главная метрика успеха рефакторинга.
- **Given** текущий код `pkg/draftrag/search.go` с 7×6 матрицей роутинга
- **When** добавляется фиктивная новая стратегия (например, `StepBack()`, не реализованная, только проверка компиляции)
- **Then** изменения локализованы в ≤ 2 методах: 1 в `SearchBuilder` (новый `StepBack()` builder-метод) + 1 в централизованном `selectRetrieval`/`selectGeneration`. Никаких изменений в 7 публичных методах (`Retrieve`, `Answer`, `Cite`, `InlineCite`, `Stream`, `StreamSources`, `StreamCite`).
- Evidence: `git diff main -- pkg/draftrag/search.go` показывает изменения только в `select*` helper'ах и/или `SearchBuilder`-методах; `go test ./pkg/draftrag/...` зелёный.

### AC-002 SearchBuilder: количество публичных строк падает

- **Why:** количественный индикатор устранения дубликации.
- **Given** `pkg/draftrag/search.go` до рефакторинга содержит 480 строк
- **When** рефакторинг выполнен
- **Then** файл содержит ≤ 280 строк (снижение ≥ 40%). Не публичные helper'ы (например, `selectRetrieval`) выносятся в `internal/application/` или `pkg/draftrag/search_routing.go` (отдельный файл).
- Evidence: `wc -l pkg/draftrag/search.go` показывает значение ≤ 280.

### AC-003 Application errors: `errors.Is(err, draftrag.ErrEmptyQuery)` работает

- **Why:** пользователь библиотеки в production полагается на типизированные sentinel'ы.
- **Given** `Pipeline.Answer(ctx, "   ")` (пробельный вопрос)
- **When** вызов завершается
- **Then** `errors.Is(err, draftrag.ErrEmptyQuery) == true`. Аналогично для `Query`, `Retrieve`, `Search().Answer(ctx)`, `IndexBatch` с пустым `doc.Content`.
- Evidence: новый unit-тест в `pkg/draftrag/pipeline_errors_test.go` (или существующий, расширенный) с пробельными/пустыми входами.

### AC-004 Application errors: zero inline `errors.New` для question/topK

- **Why:** структурный запрет на регрессию.
- **Given** все `.go` файлы в `internal/application/`
- **When** выполнен `grep -rn 'errors\.New("' internal/application/`
- **Then** результат содержит ≤ 1 совпадение, и это — `errors.New("question is empty")` (или эквивалент), оставленное как fallback в одном legacy-месте с `@deprecated` комментарием, ИЛИ 0 совпадений.
- Evidence: `! grep -rn 'errors\.New("question\|errors\.New("topK\|errors\.New("query' internal/application/` завершается с exit 0.

### AC-005 Error mapping: все sentinel'ы достижимы

- **Why:** пользовательский код не должен получать сырые application-ошибки.
- **Given** сценарии, в которых `application.ErrFiltersNotSupported`, `application.ErrHybridNotSupported`, `application.ErrStreamingNotSupported`, `application.ErrDeleteNotSupported` возникают
- **When** они пробрасываются через публичный API
- **Then** `errors.Is(err, draftrag.ErrFiltersNotSupported)` (и т.д.) возвращает `true`.
- Evidence: новый `pkg/draftrag/error_mapping_test.go` с таблицей test cases, проверяющей каждый mapping.

### AC-006 Error mapping: `mapValidationErr` переименован/упрощён

- **Why:** функция с misnamed-API — техдолг.
- **Given** `pkg/draftrag/draftrag.go` содержит функцию `mapValidationErr`
- **When** рефакторинг выполнен
- **Then** функция переименована (например, `mapAppError` или `wrapAppError`), реализует ВСЕ необходимые маппинги из RQ-003, и НЕ содержит dead branches.
- Evidence: `grep "mapValidationErr" pkg/draftrag/` показывает ≤ 1 совпадение (определение) или 0 (если переименована).

### AC-007 Index: использует worker pool при concurrency > 1

- **Why:** устранение архитектурного долга, симметрия с IndexBatch.
- **Given** `Pipeline{IndexConcurrency: 4}` и 10 документов по 100мс embed каждый
- **When** вызывается `pipeline.Index(ctx, docs)`
- **Then** общее время ≤ 400мс (4 параллельных воркера, 3 раунда), а не ~1000мс. При `IndexConcurrency: 1` поведение эквивалентно текущему.
- Evidence: новый benchmark или timing-тест в `internal/application/pipeline_index_concurrency_test.go` (с in-memory embedder, замеряющим `time.Now()`); чувствительность к flakiness минимизируется через достаточно крупный вход (≥ 10 документов).

### AC-008 UpdateDocument: атомарность для pgvector

- **Why:** защита от потери данных при сбое переиндексации.
- **Given** документ с ID `doc-1` уже проиндексирован в pgvector; `Pipeline.UpdateDocument(ctx, doc)` вызван с изменённым содержимым; embedder возвращает ошибку на 3-м чанке
- **When** метод возвращает ошибку
- **Then** в store остаются старые чанки `doc-1` (rollback выполнен). Количество чанков с `parent_id='doc-1'` равно pre-update количеству.
- Evidence: integration-тест в `internal/infrastructure/vectorstore/pgvector_atomic_update_test.go` (требует docker-compose, как `pgvector_test.go`) ИЛИ unit-тест с моком `TransactionalDocumentStore` в `internal/application/pipeline_test.go`.

### AC-009 UpdateDocument: degraded-path для не-транзакционных store

- **Why:** best-effort контракт должен быть явным.
- **Given** `Pipeline.UpdateDocument` с InMemoryStore (или другим не-транзакционным store)
- **When** индексация падает после успешного delete
- **Then** возвращается `draftrag.ErrUpdateNotAtomic` (новый sentinel) с wrapped underlying error.
- Evidence: unit-тест с in-memory store + failing embedder.

### AC-010 Streaming: bounded buffer + memory safety

- **Why:** предотвращение OOM при медленном consumer.
- **Given** `PipelineOptions{StreamBufferSize: 4}`; streaming-генератор производит 10000 токенов; consumer читает с задержкой 1мс/токен
- **When** streaming завершается (или отменяется по ctx)
- **Then** peak memory канала-буфера не превышает 4 * sizeof(string-header) ≈ 96 байт. Все 10000 токенов доставлены (или отменены по ctx, если ctx deadline).
- Evidence: unit-тест в `internal/application/stream_backpressure_test.go` с измерением `cap(tokenChan)`.

### AC-011 Rate-limiter: per-worker опция

- **Why:** конфигурируемость под разные сценарии нагрузки.
- **Given** `Pipeline{IndexConcurrency: 4, IndexBatchRateLimit: 10, IndexBatchRateLimitPerWorker: true}`; 100 документов с embedder, замеряющим вызовы
- **When** `IndexBatch(ctx, docs, ...)` выполнен
- **Then** общее количество embed-вызовов в секунду ≈ 40 (4 workers × 10), а не 10. При `PerWorker: false` (default) — ≈ 10.
- Evidence: unit-тест в `internal/application/batch_ratelimit_test.go` с fake-embedder'ом, считающим `time.Now()`-based rate.

### AC-012 Rate-limiter: документация default

- **Why:** устранение неоднозначности для пользователя.
- **Given** `README.md` секция "Batch-индексация больших корпусов"
- **When** ревьюер читает её
- **Then** явно указано: "По умолчанию `IndexBatchRateLimit` — общий лимит на пул воркеров (shared fixed-rate). Для лимита per-worker используйте `IndexBatchRateLimitPerWorker: true`." Аналогичная правка в `docs/production.md`.
- Evidence: `grep -A2 "IndexBatchRateLimit" README.md` содержит слово `shared` или `per-worker`.

### AC-013 README: все 6 stores перечислены

- **Why:** discoverability, репутация.
- **Given** `README.md` "Векторные хранилища" секция
- **When** просмотрена
- **Then** присутствуют: In-memory, PostgreSQL+pgvector, Qdrant, ChromaDB, Weaviate, Milvus (или эквивалентное перечисление, сгруппированное по production-readiness).
- Evidence: `grep -E "Weaviate|Milvus" README.md` возвращает ≥ 1 совпадение в секции хранилищ.

### AC-014 Capability-таблица

- **Why:** single-source-of-truth для пользователя при выборе store.
- **Given** `docs/vector-stores.md` (или `docs/capability-matrix.md`)
- **When** открыт
- **Then** содержит markdown-таблицу: | Store | Retrieval | Metadata filter | ParentID filter | Hybrid | DeleteByParentID | Collection mgmt |. Все 6 stores присутствуют как строки; ячейки заполнены `✅` / `❌` / `N/A`. Несовместимые комбинации помечены footnote.
- Evidence: `grep -c "| ✅\|❌\|N/A" docs/vector-stores.md` ≥ 30 (5 stores × 6 capabilities).

### AC-015 ROADMAP: Weaviate/Milvus отмечены реализованными

- **Why:** синхронизация roadmap с реальностью.
- **Given** `ROADMAP.md` секция "Приоритет 2 → Additional vector stores"
- **When** просмотрена
- **Then** Weaviate и Milvus перенесены в "Реализовано" с пометкой ⚠️ "hybrid search не поддерживается".
- Evidence: `grep -E "Weaviate.*✅|Milvus.*✅" ROADMAP.md` возвращает ≥ 2 совпадения.

### AC-016 Сборка и тесты зелёные

- **Why:** базовый gate.
- **Given** все изменения RQ-001..RQ-008
- **When** запущены `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`
- **Then** все команды завершаются с exit code 0. Покрытие `internal/domain` ≥ 100%, `internal/application` ≥ 83.3% (не ниже baseline).
- Evidence: вывод команд.

## Допущения

- Транзакционность pgvector через `*sql.Tx` достижима без изменения публичного API `*sql.DB` пользователя (используется `pgx`-специфичный `db.BeginTx` через тот же `*sql.DB`).
- Реализация `TransactionalDocumentStore` capability будет опциональной; существующие реализации `DocumentStore` продолжат работать без изменений.
- Текущее поведение `Index` (sequential) не имеет наблюдаемых пользователем гарантий параллелизма — рефакторинг не breaking change, если сохранён контракт "первая ошибка прерывает обработку" или явно выбран путь "partial results + error".
- Стриминг-буфер с `StreamBufferSize=0` (default) будет означать текущее поведение (unbuffered) — обратная совместимость. Альтернатива: `0` = fallback to unbounded channel с warning — должна быть зафиксирована в AC явно.
- Существующие маркеры `@sk-task hardening-2026q2#T1.1` и `#T3.2` останутся в коде как исторические; новые маркеры — `@sk-task api-consistency-pass#<RQ-id>`.
- Тесты, помеченные как integration (pgvector docker), могут потребовать `RUN_INTEGRATION_TESTS=1` env var; default — skip (как в существующих `*_test.go` файлах).
- Weaviate-тесты в `internal/infrastructure/vectorstore/weaviate_test.go` уже зелёные; Milvus — аналогично. Эти тесты служат baseline для AC-013/AC-014.

## Критерии успеха

- SC-001 Количество if-веток в `pkg/draftrag/search.go` (роутинг) уменьшается с ~42 до ≤ 14 (7 методов × 2 helper-вызова = 14, плюс HyDE/MultiQuery branch внутри helper'ов).
- SC-002 Время индексации 1000 документов с `IndexConcurrency=8` падает с ~1000с (sequential baseline) до ≤ 200с на in-memory embedder (≈ 5x speedup).
- SC-003 Добавление новой стратегии retrieval (по `git diff` от snapshot main) затрагивает ≤ 2 файла в production-коде (не считая тестов).
- SC-004 100% sentinel-ошибок в публичном API достижимы через `errors.Is` в пользовательском коде (проверяется grep + unit-тест AC-005).

## Краевые случаи

- **Пустой pipeline-config:** `IndexBatchRateLimitPerWorker=true` + `IndexConcurrency=0` → используется default concurrency (4) и per-worker rate остаётся в силе.
- **nil context:** все методы `Index`, `IndexBatch`, `UpdateDocument`, `AnswerStream*` сохраняют текущее поведение panic на nil context (это документировано).
- **Race на `processedCount` в IndexBatch:** если `Index` переиспользует worker pool, sync.Mutex уже есть в `batch.go`; рефакторинг не должен вводить новых shared mutable state.
- **Wrap в `UpdateDocument`:** если store не реализует ни `DocumentStore`, ни `TransactionalDocumentStore`, текущее поведение `ErrDeleteNotSupported` сохраняется.
- **StreamBufferSize при ctx cancel:** горутина-производитель ДОЛЖНА корректно завершиться при `ctx.Done()`, не оставляя висящих ссылок.
- **Capability-таблица при добавлении нового store:** документировано (в ADR или в `docs/contributing.md`), что добавление store обязывает обновить таблицу; AC-014 фиксирует только текущее состояние.

## Открытые вопросы

- **OQ-1:** `IndexBatchRateLimitPerWorker` vs `IndexBatchRateLimitPerWorker bool` — может ли пользователь захотеть разные rate для embed vs LLM? Скорее out of scope, но если да — может потребоваться nested options.
- **OQ-2:** `StreamBufferSize=0` → unbounded vs unbuffered — какой default безопаснее? Unbuffered = текущее поведение, backward-compatible. Unbounded = риск OOM. Предлагается unbuffered с warning в docstring. Требует подтверждения.
- **OQ-3:** `UpdateDocument` для stores, поддерживающих `DocumentStore` но НЕ `TransactionalDocumentStore` (например, ChromaDB) — должен ли rollback быть best-effort "re-insert old chunks" с возвратом `ErrUpdateNotAtomic`, или просто delete + return error без попытки восстановления? Предлагается best-effort с явным sentinel; нужно подтверждение.
- **OQ-4:** Удаление legacy `@sk-task hardening-2026q2#...` маркеров — отдельный housekeeping-коммит или часть RQ-001? Предлагается: оставить как есть, не блокирует AC.
