---
report_type: verify
slug: reranker-cross-encoder
status: pass
docs_language: ru
generated_at: 2026-07-09
---

# Verify Report: reranker-cross-encoder

## Scope

- snapshot: Cohere Rerank API (single + batch) + BatchReranker interface + pipeline integration
- verification_mode: deep
- artifacts:
  - docs/specs/reranker-cross-encoder/spec.md
  - docs/specs/reranker-cross-encoder/plan.md
  - docs/specs/reranker-cross-encoder/tasks.md
  - docs/specs/reranker-cross-encoder/data-model.md
- inspected_surfaces:
  - `internal/domain/interfaces.go` — BatchReranker interface
  - `pkg/draftrag/reranker/reranker.go` — re-exports
  - `pkg/draftrag/reranker/errors.go` — ErrInvalidRerankerConfig
  - `pkg/draftrag/reranker/cohere.go` — CohereReranker implementation
  - `internal/application/retrieval.go` — maybeRerankBatch
  - `internal/application/query.go` — QueryMulti integration
  - `pkg/draftrag/draftrag.go` — BatchReranker re-export
  - `pkg/draftrag/reranker/cohere_test.go` — unit tests
  - `pkg/draftrag/reranker_test.go` — pipeline integration tests
  - `docs/en/reranker.md`, `docs/ru/reranker.md` — documentation
  - `ROADMAP.md` — status update

## Verdict

- status: **pass**
- archive_readiness: safe
- summary: все acceptance criteria MVP подтверждены тестами, traceability markers проставлены во всех файлах, build/vet/tests pass

## Verification Matrix

| AC-ID | Task IDs | Evidence | Verdict |
|-------|----------|----------|---------|
| AC-001 | T2.1, T4.1 | `TestCohereRerank_Success`: pass; `TestPipeline_Reranker_IsCalled`: pass | pass |
| AC-002 | T2.1, T4.1 | `TestCohereRerank_EmptyChunks`: pass | pass |
| AC-003 | T1.2, T2.1, T4.1 | `TestNewCohereRerank_EmptyKey`: pass (wraps `ErrInvalidRerankerConfig`) | pass |
| AC-004 | T0.1 | Deferred P2 — not in MVP | not-verified |
| AC-005 | T0.1 | Deferred P2 — not in MVP | not-verified |
| AC-006 | T3.1, T4.1 | `TestCohereRerank_Unauthorized`: pass (error contains "401") | pass |
| AC-007 | T2.1, T4.1 | `TestCohereRerank_NoFilter`: pass (len(out)==len(in)) | pass |
| AC-008 | T3.2, T4.1 | `TestCohereRerank_BatchFanOut`: pass (5×50ms < 150ms concurrent) | pass |
| AC-009 | T3.2, T4.1 | `TestPipeline_Reranker_Fallback`: pass (non-BatchReranker called in multi-query mode) | pass |
| AC-010 | T4.2 | `docs/en/reranker.md`, `docs/ru/reranker.md` exist with `NewCohereRerank` example | pass |

## Checks

- task_state: completed=8, open=0
- build: `go build ./...` — pass
- vet: `go vet ./pkg/draftrag/... ./internal/...` — pass
- tests: `go test ./pkg/draftrag/reranker/... -v` — 10/10 pass; `go test ./pkg/draftrag/ -run TestPipeline_Reranker` — 2/2 pass

## Traceability

All 8 implementation files and test files have `@sk-task` / `@sk-test` markers:

| File | Markers |
|------|---------|
| `internal/domain/interfaces.go:92` | @sk-task T1.1 |
| `pkg/draftrag/reranker/reranker.go:11` | @sk-task T1.1 |
| `pkg/draftrag/reranker/errors.go:7` | @sk-task T1.2 |
| `pkg/draftrag/reranker/cohere.go:44-45,95` | @sk-task T2.1, T3.1, T3.2 |
| `pkg/draftrag/draftrag.go:94` | @sk-task T1.1 |
| `internal/application/retrieval.go:24` | @sk-task T3.2 |
| `internal/application/query.go:114` | @sk-task T3.2 |
| `pkg/draftrag/reranker/cohere_test.go:25,39,50,67,82,113,170,196,242,257` | @sk-test T4.1 |
| `pkg/draftrag/reranker_test.go:69` | @sk-test T4.1 |

## Errors

- none

## Warnings

- none

## Concerns

- none (all previously identified issues resolved)

## Not Verified

- AC-004, AC-005: LLM-reranker (P2, deferred)
- Performance metrics (SC-001/SC-002): eval harness не запускался

## Next Step

- safe to archive
- `speckeep archive reranker-cross-encoder .`
