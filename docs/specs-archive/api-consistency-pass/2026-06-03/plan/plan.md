# api-consistency-pass План

## Phase Contract

Inputs: spec (`docs/specs/api-consistency-pass/spec.md`), inspect report (`docs/specs/api-consistency-pass/inspect.md`, status: concerns, не блокирует).
Outputs: plan, data-model.md (no-change stub).
Stop if: spec размыт — не наш случай (16 AC, 8 RQ, 4 OQ задокументированы).

## Цель

Снять 8 архитектурных долгов draftRAG v0.1.0 фокусными изменениями в `internal/application/` и `pkg/draftrag/` без расширения публичного API за пределы явно зафиксированного в spec (новые опции `PipelineOptions` и новая опциональная capability `TransactionalDocumentStore`). План реорганизует роутинг `SearchBuilder`, нормализует sentinel-ошибки, делает `Index`/`UpdateDocument` консистентными с `IndexBatch` и синхронизирует документацию.

## MVP Slice

- **Наименьший независимо поставляемый инкремент:** RQ-002 (типизированные sentinel-ошибки в application) + RQ-003 (корректный error mapping в публичном API). Они работают как один логический коммит "errors-cleanup" и закрывают 4 AC (AC-003, AC-004, AC-005, AC-006).
- **Расширение до full-MVP:** добавить RQ-001 (SearchBuilder refactor) → закрывает AC-001, AC-002. Это 2 логических коммита, оба ≤ 300 строк diff, оба под `go test ./...` зелёные. После full-MVP спека готова к verify, даже если RQ-004..RQ-008 ещё не сделаны.

## First Validation Path

- После MVP (errors-cleanup + SearchBuilder refactor):
  ```sh
  go build ./... && go vet ./... && go test ./pkg/draftrag/... ./internal/application/... ./internal/domain/...
  ```
  Плюс targeted test:
  ```sh
  go test -run TestPipelineAnswer_TrimmedQuestion -v ./pkg/draftrag/
  go test -run TestSearchBuilder_RetrievalCount -v ./pkg/draftrag/
  ```
  Если оба зелёные и `wc -l pkg/draftrag/search.go` ≤ 280 — MVP валидирован.
- Manual check: добавить в `SearchBuilder` фиктивный метод `StepBack() *SearchBuilder`, пройтись компилятором, `git diff` показать только `search.go` и `internal/application/`. Если затронуты > 2 файлов — рефакторинг не доделан.

## Scope

- `internal/application/{query,answer,stream,retrieval,batch,pipeline}.go` — sentinel-ошибки, вынос роутинга, worker pool, atomic DocumentStore, streaming backpressure, rate-limiter.
- `internal/domain/interfaces.go` — новый опциональный capability `TransactionalDocumentStore`.
- `pkg/draftrag/{draftrag,search,errors}.go` — реэкспорт `TransactionalDocumentStore`, рефактор `mapValidationErr`, новые опции `PipelineOptions` (`IndexBatchRateLimitPerWorker`, `StreamBufferSize`), переэкспорт `ErrUpdateNotAtomic`.
- `internal/infrastructure/vectorstore/pgvector.go` — реализация `TransactionalDocumentStore` для pgvector (transaction-based delete+reinsert).
- `internal/infrastructure/vectorstore/memory.go` — fallback: при отсутствии `TransactionalDocumentStore` используем best-effort с `ErrUpdateNotAtomic`.
- `README.md`, `docs/vector-stores.md`, `ROADMAP.md` — синхронизация.
- НЕ трогаем: `internal/infrastructure/vectorstore/{chromadb,weaviate,milvus,qdrant}.go` (кроме `DocumentStore` capability — там уже есть), `internal/infrastructure/llm/*`, `internal/infrastructure/embedder/*`, `internal/infrastructure/resilience/*`, `pkg/draftrag/eval/*`, `pkg/draftrag/otel/*`.

## Implementation Surfaces

| Surface | Type | Why участвует |
|---|---|---|
| `pkg/draftrag/search.go` | existing | основной носитель 7×6 матрицы; рефактор AC-001/AC-002 |
| `pkg/draftrag/search_routing.go` (новый) | new | изоляция `selectRetrieval`/`selectGeneration`; отдельно от публичного API |
| `internal/application/query.go` | existing | Query* методы — 6 сайтов `errors.New` для замены; Query*With* как building blocks для роутинга |
| `internal/application/answer.go` | existing | Answer* методы — 9+ сайтов `errors.New`; extract retrieval + generate helper |
| `internal/application/stream.go` | existing | AnswerStream* методы — 4+ сайтов `errors.New`; `wrapStreamWithHook` рефактор для буфера |
| `internal/application/pipeline.go` | existing | `Index` → worker pool reuse; `UpdateDocument` → atomic path |
| `internal/application/batch.go` | existing | `IndexBatch` — вынос worker pool в helper; rate-limiter per-worker |
| `internal/application/worker_pool.go` (новый) | new | выделение worker pool из `batch.go` для переиспользования в `Index` |
| `internal/application/atomic_update.go` (новый) | new | общий код atomic UpdateDocument (best-effort fallback + transactional path) |
| `internal/domain/interfaces.go` | existing | добавление `TransactionalDocumentStore` |
| `internal/domain/errors.go` (или `models.go`) | existing | новый sentinel `ErrUpdateNotAtomic` |
| `internal/infrastructure/vectorstore/pgvector.go` | existing | реализация `TransactionalDocumentStore` |
| `internal/infrastructure/vectorstore/memory.go` | existing | doc-comment: best-effort, всегда возвращает `ErrUpdateNotAtomic` при сбое после delete |
| `pkg/draftrag/draftrag.go` | existing | `mapAppError` rename + расширение маппинга; новые поля `PipelineOptions` |
| `pkg/draftrag/errors.go` | existing | re-export `ErrUpdateNotAtomic` |
| `README.md`, `docs/vector-stores.md`, `ROADMAP.md` | existing | синхронизация документации |
| `internal/application/{query,answer,stream,pipeline,batch}_test.go` | existing | расширение тестов под новые sentinel'ы и поведение |
| `pkg/draftrag/{search,pipeline,error_mapping}_test.go` | existing + new | новый `error_mapping_test.go`; расширение `search_test.go` под 1-файловый routing |

## Bootstrapping Surfaces

- `internal/application/worker_pool.go` (новый) — должен существовать до изменения `pipeline.Index`; экспортирует `processDocsConcurrently(ctx, docs, hookOp) ([]Successful, []Failed)` (или эквивалент). Извлекается из текущего `batch.go:17-148` без изменения контракта `IndexBatch`.
- `internal/application/atomic_update.go` (новый) — должен существовать до изменения `pipeline.UpdateDocument`; экспортирует `updateDocumentAtomic(ctx, store, doc, embedder, chunker, hooks) error` с runtime-выбором transactional/best-effort.
- `internal/domain/interfaces.go` — добавление `TransactionalDocumentStore` ДО реализации в pgvector.
- `internal/domain/errors.go` — `ErrUpdateNotAtomic` ДО `atomic_update.go`.
- `pkg/draftrag/errors.go` — re-export `ErrUpdateNotAtomic` ДО `mapAppError` начнёт его маппить.

Все остальные surfaces могут создаваться параллельно.

## Влияние на архитектуру

- **Локальное (слой `application`):** Извлечение worker pool и atomic-update helper снижает cyclomatic complexity двух методов и создаёт две новые внутренние поверхности. Роутинг SearchBuilder'а переезжает из публичного слоя в helper-функцию — публичный API сжимается с ~480 строк роутинга до ~280 строк декларативного builder'а.
- **Граница `domain ↔ infrastructure`:** Вводится новая опциональная capability (`TransactionalDocumentStore`). По конституции это допустимо (capability-интерфейсы). Реализация pgvector добавляется; остальные store остаются на best-effort. Обратной несовместимости нет: type-assertion в `application.UpdatedDocument` использует классический `if impl, ok := store.(TransactionalDocumentStore); ok` паттерн.
- **Backward compatibility:**
  - `PipelineOptions` — additive: новые поля с zero-value defaults не ломают существующий код.
  - `mapValidationErr` rename — internal to `pkg/draftrag/`, не экспортируется. Пользовательский код его не вызывает.
  - Streaming — default `StreamBufferSize=0` сохраняет unbuffered поведение (backward-compatible по OQ-2).
  - Rate-limiter — default `IndexBatchRateLimitPerWorker=false` сохраняет shared-rate поведение.
  - `ErrUpdateNotAtomic` — новый sentinel; пользовательский код, не проверяющий его через `errors.Is`, продолжает работать (sentinel-skipping).
- **Migration steps:** не требуются. Никакие persisted entities не меняются.
- **Feature flag:** не нужен. Поведение постепенное и controlled zero-value defaults.

## Acceptance Approach

| AC | Подход | Surfaces | Observation |
|---|---|---|---|
| AC-001 | `search.go` ≤ 280 строк; добавление фиктивной стратегии (e.g. `StepBack()`) затрагивает ≤ 2 файла | `search.go`, `search_routing.go` (новый) | `wc -l pkg/draftrag/search.go` ≤ 280; `git diff` показывает ≤ 2 файла при добавлении `StepBack()` |
| AC-002 | см. AC-001 | то же | то же |
| AC-003 | `errors.Is(err, draftrag.ErrEmptyQuery)` возвращает true для пробельного вопроса | `internal/application/{query,answer,stream}.go` (замена `errors.New` на `fmt.Errorf("%w", ...)`) | новый `pipeline_errors_test.go` с 6 test cases (Query/Answer/Retrieve/InlineCite/Stream/IndexBatch) |
| AC-004 | grep `errors.New("question\|errors.New("topK\|errors.New("query` по `internal/application/` возвращает 0 | то же | shell-команда в `Makefile` (`make check-no-inline-errors`) или в CI |
| AC-005 | все sentinel'ы достижимы через `errors.Is` для пользователя | `pkg/draftrag/draftrag.go` (rename + расширение `mapAppError`) | новый `error_mapping_test.go` с таблицей (FilterNotSupported, HybridNotSupported, StreamingNotSupported, DeleteNotSupported, EmptyQuery, InvalidTopK, EmptyDocument, EmbeddingDimensionMismatch) |
| AC-006 | `mapValidationErr` rename + 0 dead branches | `pkg/draftrag/draftrag.go` | `grep "mapValidationErr" pkg/draftrag/` = 0 совпадений (или 1 для определения новой функции) |
| AC-007 | `Index` использует worker pool | `internal/application/{pipeline.go, worker_pool.go (новый), batch.go}` | timing-тест с in-memory embedder; 10 документов × 100мс при concurrency=4 → ≤ 400мс |
| AC-008 | `UpdateDocument` атомарен для pgvector | `internal/application/{pipeline.go, atomic_update.go (новый)}`, `internal/infrastructure/vectorstore/pgvector.go` | integration-тест `pgvector_atomic_update_test.go` (docker-compose) + unit-тест с моком `TransactionalDocumentStore` |
| AC-009 | best-effort + `ErrUpdateNotAtomic` для не-транзакционных | `internal/application/atomic_update.go` (новый) | unit-тест с in-memory store + failing embedder |
| AC-010 | streaming с буфером | `internal/application/stream.go` (`wrapStreamWithHook` рефактор) | `stream_backpressure_test.go` с cap(channel) ≤ StreamBufferSize |
| AC-011 | per-worker rate-limiter | `internal/application/batch.go` (`rateLimiter` с per-worker token-bucket) | `batch_ratelimit_test.go` с fake-embedder'ом, замеряющим rate |
| AC-012 | документация default | `README.md`, `docs/production.md` | grep `IndexBatchRateLimit` README → содержит "shared" |
| AC-013 | README перечисляет 6 stores | `README.md` | grep `Weaviate\|Milvus` README в секции хранилищ ≥ 2 |
| AC-014 | capability-таблица | `docs/vector-stores.md` (предпочтительно) или `docs/capability-matrix.md` (новый) | grep `✅\|❌\|N/A` ≥ 30 (с floor'ом для совместимости с W-001 inspect warning) |
| AC-015 | ROADMAP | `ROADMAP.md` | grep `Weaviate.*✅\|Milvus.*✅` ≥ 2 |
| AC-016 | базовый gate | весь репозиторий | `go build/vet/test/lint` exit 0; coverage не ниже 100%/83.3%/60% |

## Данные и контракты

- **`PipelineOptions`:** расширяется двумя полями (`StreamBufferSize int`, `IndexBatchRateLimitPerWorker bool`). Zero-values сохраняют текущее поведение (см. DEC-007, DEC-008).
- **`TransactionalDocumentStore` (новый capability-интерфейс):** `internal/domain/interfaces.go`. Методы: `BeginTx(ctx) (TransactionalTx, error)`, `DeleteByParentIDTx(tx, parentID) error`, `UpsertTx(tx, chunk) error`. Определение в `internal/domain/` (не public), type-aliased в `pkg/draftrag/draftrag.go` (public). Mock — internal test-double (см. DEC-006).
- **`ErrUpdateNotAtomic` (новый sentinel):** `internal/domain/models.go` (рядом с другими sentinel'ами в `var (...)` блоке). Re-export в `pkg/draftrag/errors.go`.
- **`mapValidationErr` → `mapAppError`:** сигнатура та же, тело расширено (все sentinel-маппинги). Имя отражает назначение.
- **Data model:** не меняется. `data-model.md` = no-change stub.
- **No new contracts/api.md или contracts/events.md:** это не API/transport-фича, изменения в Go-интерфейсах документируются в spec.md.

## Стратегия реализации

- **DEC-001 Routing strategy via internal select helper**
  - Why: устранить 7×6 матрицу; централизовать выбор retrieval и generation стратегий в одном месте
  - Tradeoff: дополнительный indirection (один `selectRetrieval()` + `selectGeneration()` приватный вызов); новый файл `search_routing.go`
  - Affects: `pkg/draftrag/search.go`, новый `pkg/draftrag/search_routing.go`, `internal/application/{query,answer,stream}.go` (вынос `Answer*WithX` building blocks)
  - Validation: AC-001 (≤ 2 файла diff при добавлении стратегии), AC-002 (≤ 280 строк)

- **DEC-002 Single `mapAppError` для всего error mapping**
  - Why: устранить misnamed `mapValidationErr`; одна точка истины для sentinel-маппинга
  - Tradeoff: длинный switch/if-блок (≈ 10-12 case'ов); нужно покрыть таблицей тестов
  - Affects: `pkg/draftrag/draftrag.go`, `pkg/draftrag/error_mapping_test.go` (новый)
  - Validation: AC-005, AC-006

- **DEC-003 Sentinel-ошибки в application через `fmt.Errorf("%w: ...", domain.Err*, ...)`**
  - Why: устранить `errors.New` inline; сделать sentinel'ы достижимыми
  - Tradeoff: verbose wrapping (каждый сайт +1 строка); требует review каждого call site
  - Affects: 15+ sites в `internal/application/{query,answer,stream,pipeline,batch}.go`
  - Validation: AC-003, AC-004

- **DEC-004 Worker pool extraction**
  - Why: переиспользовать пул в `Index` без копи-пасты; сохранить семантику `IndexBatch` partial results
  - Tradeoff: новая внутренняя surface (`worker_pool.go`); Index теряет first-error semantics (становится как IndexBatch — partial results) — должно быть явно зафиксировано в godoc
  - Affects: `internal/application/{pipeline.go, batch.go, worker_pool.go (новый)}`
  - Validation: AC-007 (timing-тест) + ручной review Index godoc (фиксирует изменение contract)

- **DEC-005 `TransactionalDocumentStore` как новая опциональная capability**
  - Why: атомарность `UpdateDocument` для pgvector без ломки остальных store
  - Tradeoff: +1 интерфейс в domain; pgvector получает новую реализацию; остальные store не обязаны реализовывать
  - Affects: `internal/domain/interfaces.go`, `internal/infrastructure/vectorstore/pgvector.go`, `internal/application/atomic_update.go` (новый), `pkg/draftrag/draftrag.go` (re-export)
  - Validation: AC-008 (integration-тест pgvector), AC-009 (best-effort unit-тест с in-memory)

- **DEC-006 Streaming: bounded buffer with `StreamBufferSize`**
  - Why: предотвращение OOM при медленном consumer; сохранение backward compatibility (default 0 → unbuffered)
  - Tradeoff: +1 поле в `PipelineOptions`; рефактор `wrapStreamWithHook` в helper
  - Affects: `internal/application/stream.go` (refactor), `pkg/draftrag/draftrag.go` (new field)
  - Validation: AC-010 (memory cap тест)

- **DEC-007 Rate-limiter per-worker toggle**
  - Why: конфигурируемость под разные сценарии; сознательный выбор default (shared)
  - Tradeoff: +1 поле в `PipelineOptions`; условная логика в `batch.go`
  - Affects: `internal/application/batch.go`, `pkg/draftrag/draftrag.go`
  - Validation: AC-011 (rate-timing тест), AC-012 (doc check)

- **DEC-008 Capability-таблица: edit-in-place `docs/vector-stores.md` (без нового файла)**
  - Why: single-source-of-truth; новая страница создаст двусмысленность "где искать"
  - Tradeoff: длиннее `vector-stores.md` (но это уже специфичный документ про stores)
  - Affects: `docs/vector-stores.md`, `README.md`, `ROADMAP.md`
  - Validation: AC-013, AC-014, AC-015

## Incremental Delivery

### MVP (Первая ценность)

- **Iteration 1 — Errors cleanup (RQ-002 + RQ-003):**
  - 1 commit: "feat(api): typed sentinel errors in application layer + mapAppError"
  - Surfaces: `internal/application/{query,answer,stream}.go` (15+ replacements), `internal/domain/models.go` (verify sentinels exist), `pkg/draftrag/draftrag.go` (rename + extend `mapAppError`), `pkg/draftrag/error_mapping_test.go` (new), `pkg/draftrag/pipeline_errors_test.go` (new или extend)
  - Закрывает: AC-003, AC-004, AC-005, AC-006
  - Валидация: `go test ./pkg/draftrag/... ./internal/application/...` + `make check-no-inline-errors`

- **Iteration 2 — SearchBuilder refactor (RQ-001):**
  - 1 commit: "refactor(search): centralize routing via selectRetrieval/selectGeneration"
  - Surfaces: `pkg/draftrag/search.go` (slim down), `pkg/draftrag/search_routing.go` (new), `internal/application/{query,answer,stream}.go` (verify no behavior change)
  - Закрывает: AC-001, AC-002
  - Валидация: `wc -l pkg/draftrag/search.go` ≤ 280 + manual `StepBack()` test

### Итеративное расширение

- **Iteration 3 — Index concurrency (RQ-004):**
  - Commit: "feat(pipeline): Index uses worker pool"
  - Surfaces: `internal/application/{pipeline.go, batch.go, worker_pool.go (new)}`
  - Закрывает: AC-007
  - Валидация: timing-тест

- **Iteration 4 — UpdateDocument atomicity (RQ-005):**
  - Commit: "feat(pipeline): atomic UpdateDocument via TransactionalDocumentStore capability"
  - Surfaces: `internal/domain/{interfaces.go, models.go}` (new interface + sentinel), `internal/application/{pipeline.go, atomic_update.go (new)}`, `internal/infrastructure/vectorstore/pgvector.go` (impl), `pkg/draftrag/{draftrag.go, errors.go}` (re-exports)
  - Закрывает: AC-008, AC-009
  - Валидация: integration-тест pgvector + unit-тест best-effort

- **Iteration 5 — Streaming backpressure (RQ-006):**
  - Commit: "feat(stream): bounded buffer with StreamBufferSize option"
  - Surfaces: `internal/application/stream.go`, `pkg/draftrag/draftrag.go`
  - Закрывает: AC-010
  - Валидация: memory cap тест

- **Iteration 6 — Rate-limiter per-worker (RQ-007):**
  - Commit: "feat(batch): per-worker rate limit option + docs"
  - Surfaces: `internal/application/batch.go`, `pkg/draftrag/draftrag.go`, `README.md`, `docs/production.md`
  - Закрывает: AC-011, AC-012
  - Валидация: rate-timing тест + grep docs

- **Iteration 7 — Docs sync (RQ-008):**
  - Commit: "docs: sync README/ROADMAP/capability-matrix with implemented stores"
  - Surfaces: `README.md`, `docs/vector-stores.md` (capability-таблица), `ROADMAP.md`
  - Закрывает: AC-013, AC-014, AC-015
  - Валидация: grep checks

- **Final — Gate (AC-016):**
  - Не commit, а verification step: `go build/vet/test/lint` зелёные; coverage не ниже baseline

## Порядок реализации

1. **Сначала:** Iteration 1 (errors) — он подготавливает sentinel-инфраструктуру, на которой будут опираться Iteration 2 (SearchBuilder использует ту же систему sentinel'ов).
2. **Потом:** Iteration 2 (SearchBuilder) — рефакторинг без поведенческих изменений; errors cleanup уже в каркасе.
3. **Параллелить безопасно:** Iterations 3-6 (Index concurrency, UpdateDocument, Streaming, Rate-limiter) не пересекаются по surfaces. Могут быть в любом порядке или параллельными ветками.
4. **В конце:** Iteration 7 (docs) — синхронизирует реальность кода с документацией. Должна быть ПОСЛЕ всех feature-commits, чтобы документировать актуальное состояние.
5. **За флагом/feature-toggle:** ничего не требуется; все defaults — backward-compatible zero-values.
6. **Guarded rollout:** pgvector integration-тест требует docker-compose; помечен `RUN_INTEGRATION_TESTS=1` (как существующие pgvector-тесты). Default CI = skip.

## Риски

- **R-1 Sentinel-replacement может вскрыть скрытые зависимости пользовательского кода на error string content**
  - Mitigation: оставить wrapped message в `fmt.Errorf("%w: ...", sentinel, ...)` — текст ошибки сохраняется, sentinel становится дополнительным каналом классификации. Грепом по репозиторию убедиться, что нет тестов на точный текст ошибки в `internal/application/*_test.go`.
- **R-2 SearchBuilder refactor — высокий риск регрессии в публичном API**
  - Mitigation: итерация построчная (slim down + ensure tests green на каждом коммите); существующие тесты `search_test.go` (697 строк) и `search_builder_test.go` (256 строк) — primary regression detector. Если они зелёные, публичный API не сломан.
- **R-3 Index worker pool — изменение семантики first-error vs partial results**
  - Mitigation: явно зафиксировать в godoc `Pipeline.Index`: "использует worker pool; partial failures логируются через hooks; первая критическая ошибка возвращается в качестве возвращаемого значения". Принятое решение: **first-error semantics** (как в текущем `Index`), а не partial results. Реализация: worker pool начнёт обработку всех документов, но если хотя бы один worker упал с не-recoverable ошибкой, остальные отменяются через `ctx`.
- **R-4 TransactionalDocumentStore — pgvector-specific transaction logic может конфликтовать с runtime timeout semantics**
  - Mitigation: `BeginTx` берёт context (deadline propagation); runtime timeout для transaction не выставляется (полагаемся на context); integration-тест проверяет timeout scenario.
- **R-5 Streaming buffer — изменение наблюдаемого поведения latency**
  - Mitigation: `StreamBufferSize=0` (default) = unbuffered (текущее поведение). Backward-compatible. AC-010 фиксирует только memory safety, не latency.
- **R-6 Rate-limiter per-worker — эффективный rate = N×rateLimit, что может быть неожиданным для пользователя**
  - Mitigation: AC-012 явно документирует default и поведение per-worker; README и `docs/production.md` обновляются.
- **R-7 Capability-таблица быстро устаревает**
  - Mitigation: добавить в `docs/contributing.md` (или новый `docs/capability-matrix.md` README) policy "при добавлении нового store — обязательное обновление таблицы в том же PR". Не enforced в коде, но явный process.

## Rollout и compatibility

- **Backfill/migration:** не требуется. Никакие persisted entities, value objects, state transitions не меняются.
- **Feature flag:** не требуется. Все новые options — zero-value defaults с backward-compatible поведением.
- **Compatibility:**
  - Поведение `Search().HyDE().Stream(ctx)` — сохраняется (тесты зелёные).
  - Поведение `Index(ctx, docs)` с `IndexConcurrency=0` (default) — sequential, как сейчас.
  - Поведение `IndexBatch` rate-limiter — shared, как сейчас (default).
  - `StreamBufferSize=0` (default) — unbuffered, как сейчас.
  - `UpdateDocument` для non-transactional store — best-effort с `ErrUpdateNotAtomic` (НОВОЕ), раньше возвращалось просто `error` без классификации. Это **дополнение**, не поломка: пользователи, не проверяющие `ErrUpdateNotAtomic`, продолжают работать; пользователи, проверяющие — получают новый канал.
- **Monitoring:** не требуется (библиотека). Хуки observability уже есть; новые стадии не добавляются.
- **Operational check:** `golangci-lint run ./...` остаётся зелёным; существующие CI-скрипты (`Makefile` lint/test) — без изменений.

## Проверка

- **Automated tests:**
  - Новые: `pkg/draftrag/error_mapping_test.go`, `pkg/draftrag/pipeline_errors_test.go`, `internal/application/stream_backpressure_test.go`, `internal/application/batch_ratelimit_test.go`, `internal/infrastructure/vectorstore/pgvector_atomic_update_test.go` (integration)
  - Расширенные: `pkg/draftrag/{search_test,search_builder_test,pipeline_test}.go`, `internal/application/{query,answer,stream,pipeline_*,batch}_test.go`
  - Lint: `golangci-lint run ./...` без новых warnings
- **Targeted manual checks:**
  - `wc -l pkg/draftrag/search.go` ≤ 280 (AC-002)
  - `! grep -rn 'errors\.New("question\|errors\.New("topK\|errors\.New("query' internal/application/` (AC-004)
  - Manual добавление `StepBack() *SearchBuilder` → `git diff` показывает ≤ 2 файла (AC-001)
  - `grep "IndexBatchRateLimit" README.md` содержит "shared" (AC-012)
  - `grep -E "Weaviate|Milvus" README.md` ≥ 2 в секции stores (AC-013)
  - `grep "✅\|❌\|N/A" docs/vector-stores.md` ≥ 30 (AC-014)
- **Operational checks:**
  - `RUN_INTEGRATION_TESTS=1 go test ./...` — full pass (с docker-compose)
  - Без `RUN_INTEGRATION_TESTS=1` — все non-integration тесты зелёные

## Соответствие конституции

- **Clean Architecture:** новые surfaces (`worker_pool.go`, `atomic_update.go`, `search_routing.go`) живут в правильных слоях (application/ для первых двух, pkg/ для routing helper). Никаких обратных импортов (infrastructure → application).
- **Capability interfaces:** `TransactionalDocumentStore` — новый optional capability, не ломает существующие.
- **Context.Context:** все новые методы принимают ctx как первый параметр. `BeginTx(ctx)` propagation покрыта в pgvector реализации.
- **Sentinel errors + `errors.Is`:** RQ-002/003 делают это центральной нормой. Покрытие `errors.Is(err, draftrag.ErrEmptyQuery)` и др. — AC-003, AC-005.
- **Тестируемость (mock-реализации):** для `TransactionalDocumentStore` mock — internal test-double в test-файлах (см. DEC-005, W-003 из inspect). Конституция говорит "Каждый публичный интерфейс ДОЛЖЕН иметь мок-реализацию для тестирования" — это означает "тестируемость", а не "публичный mock-пакет". В существующем коде моки приватные (в `*_test.go`), это consistent.
- **Test coverage:** AC-016 явно требует coverage не ниже baseline (100% domain, ≥83.3% application, ≥60% vectorstore).
- **Language policy:** комментарии, godoc, документация — на русском. Все новые комментарии в spec/plan, README/docs — на русском.
- **Workflow:** feature-ветка `feature/api-consistency-pass` уже создана; commits вручную; AGENTS.md запрещает auto-commit.
- **Conflict:** нет. Все 8 RQ соответствуют принципам конституции.
- **Deferral:** capability-таблица (W-007 risk) — process documentation отложена в `docs/contributing.md` (отдельный housekeeping PR, не блокирует verify).
