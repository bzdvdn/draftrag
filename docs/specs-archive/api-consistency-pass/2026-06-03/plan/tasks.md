# api-consistency-pass Задачи

## Phase Contract

Inputs: plan.md (7 commits, 8 DEC), spec.md (16 AC), data-model.md (no-change stub).
Outputs: tasks.md с 4 фазами, Surface Map, AC coverage.
Stop if: AC не удаётся привязать — не наш случай (16 AC → 11 задач, 1:1+).

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/domain/models.go` | T1.1 |
| `internal/application/worker_pool.go` (новый) | T1.2 |
| `internal/application/atomic_update.go` (новый) | T1.1, T3.2 |
| `internal/application/pipeline.go` | T1.2, T3.1, T3.2 |
| `internal/application/batch.go` | T1.2, T3.1, T3.4 |
| `internal/application/query.go` | T2.1 |
| `internal/application/answer.go` | T2.1 |
| `internal/application/stream.go` | T2.1, T3.3 |
| `internal/application/retrieval.go` | T2.1 |
| `internal/application/pipeline_test.go` | T1.2, T2.1, T3.1, T3.2 |
| `internal/application/batch_test.go` | T1.2, T3.4 |
| `internal/application/stream_backpressure_test.go` (новый) | T3.3 |
| `internal/application/batch_ratelimit_test.go` (новый) | T3.4 |
| `internal/application/t4_1_coverage_test.go` (новый) | T4.1 |
| `internal/application/mmr.go` | T4.1 |
| `internal/infrastructure/vectorstore/pgvector.go` | T3.2, T4.1 |
| `internal/infrastructure/vectorstore/t4_1_coverage_test.go` (новый) | T4.1 |
| `internal/infrastructure/vectorstore/pgvector_atomic_update_test.go` (новый) | T3.2 |
| `internal/infrastructure/vectorstore/memory.go` | T3.2 |
| `pkg/draftrag/draftrag.go` | T2.2, T3.1, T3.3, T3.4 |
| `pkg/draftrag/errors.go` | T2.2 |
| `pkg/draftrag/search.go` | T2.3 |
| `pkg/draftrag/search_routing.go` (новый) | T2.3 |
| `pkg/draftrag/error_mapping_test.go` (новый) | T2.2 |
| `pkg/draftrag/pipeline_errors_test.go` (новый) | T2.1 |
| `pkg/draftrag/search_test.go` | T2.3 |
| `pkg/draftrag/search_builder_test.go` | T2.3 |
| `README.md` | T3.5 |
| `docs/vector-stores.md` | T3.5 |
| `docs/production.md` | T3.4 |
| `ROADMAP.md` | T3.5 |

## Implementation Context

- **Цель MVP:** errors cleanup (RQ-002+003) + SearchBuilder routing refactor (RQ-001) → AC-001..AC-006 зелёные.
- **Границы приёмки MVP:** AC-001, AC-002, AC-003, AC-004, AC-005, AC-006. Остальные AC закрываются в Фазе 3.
- **Ключевые правила (конституция):** Clean Architecture (infrastructure → application → domain, без обратных импортов); capability-интерфейсы optional (type-assertion pattern); context.Context первым аргументом; sentinel+`errors.Is` норма; комментарии на русском.
- **Инварианты:** backward-compat через zero-value defaults (новые `PipelineOptions` поля — zero-safe); `TransactionalDocumentStore` — новый optional capability, существующие store продолжают работать без изменений; streaming/rate-limit defaults сохраняют текущее поведение.
- **Ошибки/коды:** все `errors.New("question is empty")` → `fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)`; аналогично для `topK must be > 0`; новый `ErrUpdateNotAtomic` для degraded-path; `mapAppError` — единственная точка sentinel-маппинга.
- **Контракты/протокол:** public API additions только: (1) два новых поля `PipelineOptions` (`StreamBufferSize int`, `IndexBatchRateLimitPerWorker bool`); (2) новый sentinel `ErrUpdateNotAtomic`; (3) type-alias `TransactionalDocumentStore` от `internal/domain/`. Никаких breaking changes.
- **Границы scope (НЕ делаем):** hybrid search для Weaviate/Milvus; eval-harness gen metrics; refactor `memory.go` 296 строк; микрооптимизация `prompt.go`; `git commit`/`push` (AGENTS.md запрет).
- **Proof signals:** `go test ./...` зелёный; `wc -l pkg/draftrag/search.go` ≤ 280; `! grep -rn 'errors\.New("question\|errors\.New("topK\|errors\.New("query' internal/application/` exit 0; `golangci-lint run ./...` exit 0; coverage domain=100%, application≥83.3%, vectorstore≥60%.
- **References (без перечитывания):** DEC-001..DEC-008, RQ-001..RQ-008, AC-001..AC-016, OQ-2 (StreamBufferSize=0 → unbuffered), OQ-3 (best-effort + ErrUpdateNotAtomic).
- **Trace-маркеры:** `@sk-task api-consistency-pass#T<n>.<m>` над owning function/method/type declaration в production-коде; `@sk-test api-consistency-pass#T<n>.<m>` — в test-файлах. Запрещено на `package`/`import`/file-header.

## Фаза 1: Основа

Цель: подготовить bootstrap-структуры (новые типы, helper-файлы), на которые опирается MVP и Фаза 3. После Фазы 1 — `go build ./...` остаётся зелёным, никаких поведенческих изменений.

- [x] T1.1 Добавить `TransactionalDocumentStore` capability и sentinel `ErrUpdateNotAtomic`. В `internal/domain/interfaces.go` объявить новый interface (`BeginTx`, `DeleteByParentIDTx`, `UpsertTx`, `Commit`, `Rollback`) рядом с существующим `DocumentStore`; в `internal/domain/models.go` — `ErrUpdateNotAtomic = errors.New("update not atomic; old chunks may be partially deleted")` в общем `var (...)` блоке. В `internal/application/atomic_update.go` (новый) — стаб `updateDocumentAtomic(ctx, store, doc, embedder, chunker, hooks) error` с runtime type-assertion на `TransactionalDocumentStore` (пока всегда false, fallback path всегда возвращает stub error). Touches: `internal/domain/interfaces.go`, `internal/domain/models.go`, `internal/application/atomic_update.go` (новый). @sk-task api-consistency-pass#T1.1 на объявлении interface и sentinel.
- [x] T1.2 Извлечь worker pool из `IndexBatch` в отдельный helper. Создать `internal/application/worker_pool.go` с `processDocsConcurrently(ctx, docs, hooks, hookOp, concurrency, rateLimit, perWorker, processor) (successful, failed, ctxErr)`, перенести туда semaphore + ticker + worker-функцию из `internal/application/batch.go:40-137` без изменения семантики. `IndexBatch` переписать как тонкую обёртку над helper'ом. Все существующие тесты `batch_test.go` (13739 строк) ДОЛЖНЫ проходить без изменений asserts. Touches: `internal/application/batch.go`, `internal/application/worker_pool.go` (новый), `internal/application/batch_test.go`. @sk-task api-consistency-pass#T1.2 на новой функции `processDocsConcurrently`.

## Фаза 2: MVP Slice

Цель: поставить 6 из 16 AC (errors cleanup + SearchBuilder refactor) до расширения scope. После Фазы 2 — спека может быть верифицирована в части MVP (AC-001..AC-006), даже если Фаза 3 ещё впереди.

- [x] T2.1 Заменить inline `errors.New("question is empty")` / `"topK must be > 0"` / `"query is empty"` на `fmt.Errorf("%w: ...", domain.ErrEmptyQueryText, ...)` / `domain.ErrInvalidQueryTopK` / `domain.ErrEmptyQueryText` во всех `.go` файлах `internal/application/`. Пройтись по `query.go`, `answer.go`, `stream.go`, `retrieval.go` (≈ 15+ sites). Сохранить wrapped message — sentinel становится дополнительным каналом классификации, текст ошибки не меняется. Новый файл `pkg/draftrag/pipeline_errors_test.go` с таблицей: `Pipeline.Answer(ctx, "   ")` → `errors.Is(err, draftrag.ErrEmptyQuery) == true`; аналогично для `Query`, `Retrieve`, `Search().Answer`, `IndexBatch` с пустым `doc.Content`, `Search().TopK(0)`. Touches: `internal/application/{query,answer,stream,retrieval}.go`, `pkg/draftrag/pipeline_errors_test.go` (новый). @sk-task api-consistency-pass#T2.1 и @sk-test api-consistency-pass#T2.1 на owning declarations.
- [x] T2.2 Переименовать `mapValidationErr` → `mapAppError` и расширить маппинг. В `pkg/draftrag/draftrag.go`: (1) заменить имя во всех 4 callsites; (2) добавить маппинги для `ErrFiltersNotSupported`, `ErrHybridNotSupported`, `ErrEmptyQuery`, `ErrInvalidTopK`, `ErrEmptyDocument`, `ErrEmbeddingDimensionMismatch`, `ErrUpdateNotAtomic`. В `pkg/draftrag/errors.go`: re-export `ErrUpdateNotAtomic = domain.ErrUpdateNotAtomic`. Новый `pkg/draftrag/error_mapping_test.go` с таблицей test cases: для каждого sentinel — verify `errors.Is` через публичный API. Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/errors.go`, `pkg/draftrag/error_mapping_test.go` (новый). @sk-task api-consistency-pass#T2.2 на `mapAppError`. AC-005, AC-006.
- [x] T2.3 Централизовать роутинг SearchBuilder через `selectRetrieval`/`selectGeneration`. В новом `pkg/draftrag/search_routing.go`: helper'ы, возвращающие `(retrievalFn, error)` и `(genFn, error)` на основе полей `SearchBuilder` (basic/HyDE/MultiQuery/Hybrid/ParentIDs/Filter). В `pkg/draftrag/search.go`: 7 публичных методов (`Retrieve`, `Answer`, `Cite`, `InlineCite`, `Stream`, `StreamSources`, `StreamCite`) переписаны как 2-3 строки делегирования. `wc -l pkg/draftrag/search.go` ≤ 280. Все существующие тесты `search_test.go` (697 строк) и `search_builder_test.go` (256 строк) — без изменений asserts. Touches: `pkg/draftrag/search.go`, `pkg/draftrag/search_routing.go` (новый), `pkg/draftrag/{search,search_builder}_test.go`. @sk-task api-consistency-pass#T2.3 на helper'ах и публичных методах SearchBuilder. AC-001, AC-002.

## Фаза 3: Основная реализация

Цель: закрыть оставшиеся 10 AC, по одному коммиту на DEC. Каждый коммит самодостаточен, тесты зелёные. Tasks T3.1..T3.5 могут выполняться в любом порядке (поверхности не пересекаются).

- [x] T3.1 `Pipeline.Index` использует worker pool из T1.2. В `internal/application/pipeline.go`: `Index` вызывает `processDocsConcurrently` (helper из T1.2) с `p.indexConcurrency`; на первой не-recoverable ошибке отменяет остальных worker'ов через `ctx` и возвращает ошибку (first-error semantics — явно зафиксировать в godoc). Поведение при `IndexConcurrency: 1` эквивалентно sequential (текущее). Timing-тест в `internal/application/pipeline_index_concurrency_test.go` (новый): 10 docs × 100мс embed при `IndexConcurrency=4` → общее время ≤ 400мс (tolerance ±20% на CI flakiness). Touches: `internal/application/pipeline.go`, `internal/application/pipeline_test.go`, `internal/application/pipeline_index_concurrency_test.go` (новый). @sk-task api-consistency-pass#T3.1 на методе `(*Pipeline).Index`. AC-007.
- [x] T3.2 Атомарный `UpdateDocument` через `TransactionalDocumentStore`. В `internal/application/atomic_update.go` (из T1.1): заменить stub на реальный код — runtime type-assertion `store.(domain.TransactionalDocumentStore)`; если true — `BeginTx` → `DeleteByParentIDTx` + `UpsertTx` всех чанков → `Commit`; при ошибке — `Rollback` + возврат wrapped error. Если false (in-memory, Qdrant, ChromaDB, Weaviate, Milvus) — best-effort path: `DeleteByParentID` + `Index`; при ошибке `Index` после успешного delete — return `ErrUpdateNotAtomic` с wrapped underlying error. В `internal/infrastructure/vectorstore/pgvector.go`: реализовать `TransactionalDocumentStore` (методы `BeginTx` через `db.BeginTx(ctx, nil)`, `DeleteByParentIDTx`/`UpsertTx` через существующие helpers + tx context, `Commit`/`Rollback` через `tx.Commit`/`tx.Rollback`). Integration-тест `internal/infrastructure/vectorstore/pgvector_atomic_update_test.go` (новый, помечен `RUN_INTEGRATION_TESTS=1`): документ проиндексирован; `UpdateDocument` с failing embedder на 3-м чанке → in-store остаются старые чанки. Unit-тест в `internal/application/pipeline_test.go`: in-memory store + failing embedder → `errors.Is(err, draftrag.ErrUpdateNotAtomic)`. Touches: `internal/application/atomic_update.go`, `internal/application/pipeline.go`, `internal/application/pipeline_test.go`, `internal/infrastructure/vectorstore/pgvector.go`, `internal/infrastructure/vectorstore/memory.go`, `internal/infrastructure/vectorstore/pgvector_atomic_update_test.go` (новый). @sk-task api-consistency-pass#T3.2 на `updateDocumentAtomic`, `UpdateDocument` (pipeline.go), pgvector BeginTx. AC-008, AC-009.
- [x] T3.3 Bounded backpressure в streaming. В `internal/application/stream.go`: рефактор `wrapStreamWithHook` — буферизированный канал с ёмкостью `p.streamBufferSize`; при `0` — unbuffered (backward-compat, OQ-2). Горутина-производитель блокируется на `case output <- token:` при заполнении буфера; select на `ctx.Done()` для отмены. В `pkg/draftrag/draftrag.go`: добавить `StreamBufferSize int` в `PipelineOptions` (default 0). Тест в `internal/application/stream_backpressure_test.go` (новый): producer 10000 токенов × 1мс; consumer с задержкой; `cap(tokenChan) == StreamBufferSize`; peak memory не превышает `cap * sizeof(string-header)`. Touches: `internal/application/stream.go`, `pkg/draftrag/draftrag.go`, `internal/application/stream_backpressure_test.go` (новый). @sk-task api-consistency-pass#T3.3 на `wrapStreamWithHook`. AC-010.
- [x] T3.4 Per-worker rate-limiter toggle. В `internal/application/batch.go` (через helper из T1.2): при `perWorker=true` — отдельный ticker на каждого worker'а с интервалом `time.Second / rateLimit`; при `false` — общий ticker (текущее поведение). В `pkg/draftrag/draftrag.go`: добавить `IndexBatchRateLimitPerWorker bool` в `PipelineOptions` (default false). В `pkg/draftrag/draftrag.go` и `docs/production.md`: обновить godoc и секцию "Index-индексация" с явным указанием "по умолчанию — shared rate-limit на пул, для per-worker — `IndexBatchRateLimitPerWorker: true`". Тест в `internal/application/batch_ratelimit_test.go` (новый): fake-embedder замеряет time-интервалы; при `PerWorker=true, Concurrency=4, RateLimit=10` — общий rate ≈ 40/сек; при `PerWorker=false` — ≈ 10/сек. Touches: `internal/application/batch.go`, `pkg/draftrag/draftrag.go`, `docs/production.md`, `internal/application/batch_ratelimit_test.go` (новый). @sk-task api-consistency-pass#T3.4 на `processDocsConcurrently`. AC-011, AC-012.
- [x] T3.5 Синхронизация документации. В `README.md`: секция "Векторные хранилища" — добавить Weaviate и Milvus (с пометкой "hybrid search не поддерживается"); ссылка на `docs/vector-stores.md`. В `docs/vector-stores.md`: добавить capability-таблицу — строки: in-memory, pgvector, qdrant, chromadb, weaviate, milvus; колонки: Basic retrieval, Metadata filter, ParentID filter, Hybrid, DeleteByParentID, Collection mgmt. Ячейки: ✅ / ❌ / N/A; несовместимые комбинации — footnote. Минимум 30 ячеек (AC-014 floor). В `ROADMAP.md`: перенести Weaviate и Milvus из "Приоритет 2 → Additional vector stores" в "Реализовано ✅" с ⚠️ "hybrid search не поддерживается". Touches: `README.md`, `docs/vector-stores.md`, `ROADMAP.md`. @sk-task api-consistency-pass#T3.5 на каждой секции (по одному маркеру над owning heading). AC-013, AC-014, AC-015.

## Фаза 4: Проверка

Цель: доказать, что фича работает (AC-016), и подготовить пакет к verify-фазе.

- [x] T4.1 Финальный gate. Выполнить последовательно: `go build ./...` (exit 0); `go vet ./...` (exit 0); `go test ./...` (exit 0); `golangci-lint run ./...` (exit 0); coverage — `internal/domain` ≥ 100%, `internal/application` ≥ 83.3%, `internal/infrastructure/vectorstore` ≥ 60%. Запустить targeted manual checks: `wc -l pkg/draftrag/search.go` ≤ 280; `! grep -rn 'errors\.New("question\|errors\.New("topK\|errors\.New("query' internal/application/` (exit 0). Все результаты зафиксировать в `verify/`-phase входе. Touches: вся репа (read-only verification). @sk-test api-consistency-pass#T4.1 на verify-чеке (в verify-фазе, не в implementation). AC-016.

## Покрытие критериев приемки

- AC-001 -> T2.3
- AC-002 -> T2.3
- AC-003 -> T2.1
- AC-004 -> T2.1
- AC-005 -> T2.2
- AC-006 -> T2.2
- AC-007 -> T3.1
- AC-008 -> T3.2
- AC-009 -> T3.2
- AC-010 -> T3.3
- AC-011 -> T3.4
- AC-012 -> T3.4
- AC-013 -> T3.5
- AC-014 -> T3.5
- AC-015 -> T3.5
- AC-016 -> T4.1

## Заметки

- Порядок задач = порядок коммитов: T1.1 → T1.2 → T2.1 → T2.2 → T2.3 → T3.1 → T3.2 → T3.3 → T3.4 → T3.5 → T4.1.
- T3.1..T3.5 можно безопасно переупорядочивать или выполнять параллельными ветками (поверхности не пересекаются после T1.2).
- Каждый task ID — phase-scoped (`T<phase>.<index>`). Phase 1 = bootstrap; Phase 2 = MVP; Phase 3 = feature work; Phase 4 = verification.
- Trace-маркеры `@sk-task`/`@sk-test` обязательны для marking задачи выполненной (см. AGENTS.md). Размещение — над owning function/method/test/type, НЕ на package/import/file-header.
- Не редактируйте исходный код на фазе tasks. Не коммитьте автоматически.
- Implement-агент читает только `tasks.md` + файлы из `Touches:` активной задачи; не перечитывайте `spec.md`/`plan.md`/`data-model.md` без необходимости.
- Если задача получает новые файлы (новые helper'ы, новые tests) — добавляйте их в Surface Map в patch-режиме, не переписывая tasks.md целиком.
