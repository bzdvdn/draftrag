---
report_type: verify
slug: api-consistency-pass
status: pass
docs_language: ru
generated_at: 2026-06-03
---

# Verify Report: api-consistency-pass

## Scope

- snapshot: 11/11 tasks complete (T1.1, T1.2, T2.1, T2.2, T2.3, T3.1, T3.2, T3.3, T3.4, T3.5, T4.1). Все 16/16 AC (AC-001..AC-016) покрыты. Gates green: build, vet, test, lint, coverage, T2.1 grep, search.go size.
- verification_mode: default
- artifacts:
  - .speckeep/constitution.summary.md
  - docs/specs/api-consistency-pass/spec.md
  - docs/specs/api-consistency-pass/tasks.md
  - docs/specs/api-consistency-pass/plan.md
- inspected_surfaces:
  - internal/domain/{interfaces,models}.go — `TransactionalDocumentStore`, `ErrUpdateNotAtomic`
  - internal/application/{atomic_update,worker_pool,batch,pipeline,query,answer,stream,retrieval,mmr}.go — worker pool extraction, error wrapping, Index fail-fast, atomic UpdateDocument, bounded backpressure, per-worker rate limiter
  - internal/application/{t4_1_coverage,pipeline_index_concurrency,stream_backpressure,batch_ratelimit,pipeline_test}.go — test coverage
  - internal/infrastructure/vectorstore/{pgvector,memory}.go — `BeginTx`/`Commit`/`Rollback`, ErrUpdateNotAtomic fallback
  - internal/infrastructure/vectorstore/t4_1_coverage_test.go — coverage lift to 60.7%
  - pkg/draftrag/{draftrag,errors,search,search_routing,pipeline_errors_test,error_mapping_test}.go — public API: `mapAppError`, `StreamBufferSize`, `IndexBatchRateLimitPerWorker`, 7 SearchBuilder methods ≤ 280 LOC
  - docs/{production,vector-stores}.md, README.md, ROADMAP.md — T3.4 rate-limit section, T3.5 capability table 6×6

## Verdict

- status: pass
- archive_readiness: safe
- summary: 11/11 задач `[x]` с observable proof; все AC закрыты; gates green; coverage floors выполнены; trace-маркеры на месте.

## Checks

- task_state: completed=11, open=0; `verify-task-state.sh` exit 0 (`TASKS_TOTAL=11 TASKS_COMPLETED=11 TASKS_OPEN=0`)
- acceptance_evidence:
  - AC-001 -> T2.3 — `pkg/draftrag/search.go` (229 LOC, ≤ 280); `Retrieve`/`Answer`/`Cite`/`InlineCite`/`Stream`/`StreamSources`/`StreamCite` делегируют в `selectRetrieval`/`selectGeneration` (search_routing.go:38, 92)
  - AC-002 -> T2.3 — `pkg/draftrag/search_test.go` (697 LOC) и `search_builder_test.go` (256 LOC) — без изменений asserts; `go test ./pkg/draftrag/` PASS
  - AC-003 -> T2.1 — `pkg/draftrag/pipeline_errors_test.go:11` (`[TEST] api-consistency-pass#T2.1`); table-driven test для `Pipeline.Answer/Query/Retrieve/Search().Answer/IndexBatch` с пустыми входами → `errors.Is(err, draftrag.ErrEmptyQuery) == true`
  - AC-004 -> T2.1 — `grep -rn 'errors\.New("question\|errors\.New("topK\|errors\.New("query' internal/application/` exit 0 (0 matches); inline `errors.New` в `internal/application/{query,answer,stream,retrieval,mmr}.go` отсутствуют; только `var ErrXxx = errors.New(...)` декларации sentinel'ов (intentional)
  - AC-005 -> T2.2 — `pkg/draftrag/error_mapping_test.go:12` (`[TEST] api-consistency-pass#T2.2`); `mapAppError` маппит `ErrFiltersNotSupported`, `ErrHybridNotSupported`, `ErrEmptyQuery`, `ErrInvalidTopK`, `ErrEmptyDocument`, `ErrEmbeddingDimensionMismatch`, `ErrUpdateNotAtomic` в публичные sentinel'ы
  - AC-006 -> T2.2 — `pkg/draftrag/errors.go` re-exports `ErrUpdateNotAtomic = domain.ErrUpdateNotAtomic`; `pkg/draftrag/pipeline_coverage_test.go:553` (`[TEST] api-consistency-pass#T2.2 ErrStreamingNotSupported reachable через mapAppError`)
  - AC-007 -> T3.1 — `internal/application/pipeline.go:215` (`@sk-task api-consistency-pass#T3.1 параллельная обработка Index через processDocsConcurrently`); `internal/application/pipeline_index_concurrency_test.go:37,82,120,146,177` (5 тестов, `[TEST] api-consistency-pass#T3.1`)
  - AC-008 -> T3.2 — `internal/application/atomic_update.go:50,99` (`@sk-task api-consistency-pass#T3.2 transactional ветка` + `best-effort ветка`); `internal/infrastructure/vectorstore/pgvector_atomic_update_test.go:18,152` (integration tests, `[TEST] api-consistency-pass#T3.2`)
  - AC-009 -> T3.2 — `internal/application/pipeline_test.go:223,258,279` (3 unit tests, `[TEST] api-consistency-pass#T3.2 best-effort path` + `ErrUpdateNotAtomic`); `draftrag.ErrUpdateNotAtomic` re-exported в `pkg/draftrag/errors.go`
  - AC-010 -> T3.3 — `internal/application/stream.go:92` (`@sk-task api-consistency-pass#T3.3 bounded backpressure — output chan с cap=p.streamBufferSize`); `internal/application/stream_backpressure_test.go:62,90,120,155,227,259` (6 тестов, `[TEST] api-consistency-pass#T3.3`)
  - AC-011 -> T3.4 — `pkg/draftrag/draftrag.go` (`@sk-task api-consistency-pass#T3.4` на `PipelineOptions.IndexBatchRateLimitPerWorker`); `internal/application/worker_pool.go:44` (`@sk-task api-consistency-pass#T3.4 per-worker rate-limiter toggle`)
  - AC-012 -> T3.4 — `internal/application/batch_ratelimit_test.go:49,89,129` (3 теста, `[TEST] api-consistency-pass#T3.4`); `docs/production.md` новая секция "## Index-индексация: rate limiting"
  - AC-013 -> T3.5 — `README.md:7` (`@sk-task api-consistency-pass#T3.5 docs sync — Векторные хранилища`); `docs/vector-stores.md:205` (`@sk-task api-consistency-pass#T3.5 docs sync — Milvus section`); `ROADMAP.md:9` (`@sk-task api-consistency-pass#T3.5 docs sync — Weaviate/Milvus moved to Реализовано ✅`)
  - AC-014 -> T3.5 — `docs/vector-stores.md:234` (`@sk-task api-consistency-pass#T3.5 docs sync — capability-таблица 6×6 = 36 ячеек`); 6 строк × 6 колонок = 36 ячеек (≥ 30 floor)
  - AC-015 -> T3.5 — `ROADMAP.md:9` (Weaviate + Milvus перенесены в "Реализовано ✅" с ⚠️ "hybrid search не поддерживается")
  - AC-016 -> T4.1 — `tasks.md:88` (`[TEST] api-consistency-pass#T4.1`); gates: `go build ./...` exit 0; `go vet ./...` exit 0; `go test ./...` all PASS; `golangci-lint run ./...` exit 0 (only external pgx/otel typecheck warnings); `internal/domain` 100.0%, `internal/application` 83.3%, `internal/infrastructure/vectorstore` 60.7%; `wc -l pkg/draftrag/search.go` = 229 ≤ 280; `! grep 'errors\.New("question\|errors\.New("topK\|errors\.New("query' internal/application/` exit 0
- implementation_alignment:
  - T1.1: `internal/domain/interfaces.go:30` объявляет `TransactionalDocumentStore` (методы `BeginTx`, `DeleteByParentIDTx`, `UpsertTx`, `Commit`, `Rollback`); `internal/domain/models.go:127` объявляет `ErrUpdateNotAtomic` в общем `var (...)` блоке
  - T1.2: `internal/application/worker_pool.go:43-44` — `processDocsConcurrently` (worker pool); `internal/application/batch.go:10` (`@sk-task T1.2`) — `IndexBatch` как тонкая обёртка
  - T2.1: 24 sites `errors.New` → `fmt.Errorf("%w: ...", domain.ErrXxx)` в `internal/application/{query,answer,stream,retrieval,mmr}.go`; `internal/application/answer.go:24,98,165,204,275,341` (6 sites), `query.go:119,164,220,283` (4 sites), `stream.go:17,130` (2 sites) — все с `@sk-task T2.1`
  - T2.2: `pkg/draftrag/draftrag.go:158` `mapAppError` (бывший `mapValidationErr`); расширен на 7 sentinel'ов; `pkg/draftrag/errors.go:15` re-export `ErrUpdateNotAtomic`
  - T2.3: `pkg/draftrag/search_routing.go:38,92` — `selectRetrieval`, `selectGeneration`; `pkg/draftrag/search.go` сокращён с 480 → 229 LOC
  - T3.1: `internal/application/pipeline.go:215` — `Index` через `processDocsConcurrently`; first-error semantics через `cancel()` + `ctx.Done()`
  - T3.2: `internal/application/atomic_update.go:50` — transactional ветка (BeginTx → DeleteByParentIDTx + UpsertTx → Commit; при ошибке → Rollback); `:99` — best-effort ветка (DeleteByParentID + Index; при ошибке Index → `ErrUpdateNotAtomic`); `internal/infrastructure/vectorstore/pgvector.go` `BeginTx`/`Commit`/`Rollback`/`pgVectorTx`
  - T3.3: `internal/application/stream.go:92` — `wrapStreamWithHook` использует `p.streamBufferSize`; `select` с `ctx.Done()` для отмены; при `0` — unbuffered (OQ-2)
  - T3.4: `internal/application/worker_pool.go:44` — per-worker ticker с `defer localLimiter.Stop()`; `pkg/draftrag/draftrag.go` `PipelineOptions.IndexBatchRateLimitPerWorker bool` (default false); `docs/production.md` "## Index-индексация: rate limiting"
  - T3.5: `docs/vector-stores.md:205-240` — Milvus section + capability table 6×6 (in-memory, pgvector, qdrant, chromadb, weaviate, milvus × Basic retrieval, Metadata filter, ParentID filter, Hybrid, DeleteByParentID, Collection mgmt); `ROADMAP.md:9` — Weaviate + Milvus в "Реализовано ✅"
  - T4.1: dead `indexChunks` (pipeline.go:135-158, ~20 statements) удалён; `mmr.go:39` заменён `errors.New("topK must be > 0")` → `fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)`; `internal/application/t4_1_coverage_test.go` (203 LOC, 5 тестов) + `internal/infrastructure/vectorstore/t4_1_coverage_test.go` (8 тестов) поднимают coverage до floors

## Errors

- none

## Warnings

- check-verify-ready.sh выдаёт 11 warnings о `Touches:`-путях с brace-expansion (`{query,answer,stream,retrieval}.go` и т.п.) — script не разворачивает shell braces, но реальные файлы существуют. Не блокер.

## Questions

- none

## Not Verified

- T3.2 integration test (`pgvector_atomic_update_test.go`) помечен `RUN_INTEGRATION_TESTS=1` — не выполнен end-to-end без PostgreSQL. Unit-test ветка best-effort с in-memory store покрыта в `pipeline_test.go:223,258,279`.
- T3.5 Milvus hybrid-search: внутренний `SearchHybrid*` (milvus.go:275, 377, 442) существует, но публичный wrapper отсутствует — задокументировано в `docs/vector-stores.md` footnote. Не в scope текущего скоупа.
- T4.1 coverage vectorstore 60.7% — выше floor (60%), но потолок pgvector SQL методов требует running PostgreSQL; unit-test helper'ы покрыты.

## Next Step

- safe to archive
