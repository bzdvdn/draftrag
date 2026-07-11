---
report_type: verify
slug: sub-query-decomposition
status: pass
docs_language: ru
generated_at: 2026-07-11
---

# Verify Report: sub-query-decomposition

## Scope

- snapshot: полная верификация всех 4 фаз (T1.1–T4.2)
- artifacts:
  - docs/specs/sub-query-decomposition/tasks.md
  - код: см. Surface Map

## Verdict

- status: pass
- archive_readiness: ready
- summary: Все 12 задач выполнены. Все 9 AC имеют observable proof. `go vet` — clean, `go test ./...` — все проходят.

## Verification Matrix

| AC-ID | Task IDs | Evidence | Verdict |
|-------|----------|----------|---------|
| AC-001 | T1.1, T1.2, T2.2, T2.3 | `SearchBuilder.SubDecompose()` (search.go:126), `routeSubDecompose` в pickRoute (search_routing.go:28), handler во всех 7 router maps | pass |
| AC-002 | T1.3, T2.1, T2.3 | `QuerySubDecompose` (subdecompose.go:30), `LLMQueryDecomposer` (llm.go:43), tolerant JSON-парсинг, unit-тесты | pass |
| AC-003 | T3.1, T4.1 | `RuleQueryDecomposer` (rule.go:29) — разбиение по "и","или",","; 7 unit-тестов | pass |
| AC-004 | T1.3, T4.1 | `mergeSubResults` (subdecompose.go:227) — dedup по Chunk.ID, max score, sort desc; unit-тест `TestPipeline_QuerySubDecompose_MergeDedup` | pass |
| AC-005 | T3.1, T3.3, T4.1 | `CompositeDecomposer` (composite.go:44) — LLM→Rule→single fallback chain; graceful degradation при err/nil decomposer; 7 unit-тестов | pass |
| AC-006 | T1.1, T1.2, T3.3, T4.1 | `ErrSubDecomposeNotSupported` при nil decomposer; per-request override: `TestSearchBuilder_SubDecompose_PerRequestOverride` | pass |
| AC-007 | T1.3, T2.3 | Параллельный embed+search через goroutines + WaitGroup в `QuerySubDecompose`; goroutine tracking в тестах | pass |
| AC-008 | T2.2, T4.1 | `AnswerSubDecompose` (subdecompose.go:142), `AnswerSubDecomposeWithCitations` (subdecompose.go:152), `AnswerSubDecomposeWithInlineCitations` (subdecompose.go:173); unit-тесты | pass |
| AC-009 | T2.2, T3.2, T4.1 | Все 7 handler-записей в router maps; `subDecomposeCite/InlineCite/Stream/StreamSources/StreamCite`; `TestSearchBuilder_SubDecompose_AllOutputMethods` | pass |

## Task State

| Task | Status | Artifacts |
|------|--------|-----------|
| T1.1 | done | `internal/domain/interfaces.go`, `pkg/draftrag/errors.go` |
| T1.2 | done | `pkg/draftrag/draftrag.go`, `pkg/draftrag/search.go`, `pkg/draftrag/search_routing.go` |
| T1.3 | done | `internal/application/subdecompose.go` |
| T2.1 | done | `internal/infrastructure/decomposer/llm.go` |
| T2.2 | done | `pkg/draftrag/search_routing.go`, `internal/application/subdecompose.go` |
| T2.3 | done | `internal/application/subdecompose_test.go`, `internal/infrastructure/decomposer/llm_test.go` |
| T3.1 | done | `internal/infrastructure/decomposer/rule.go`, `internal/infrastructure/decomposer/composite.go` |
| T3.2 | done | `pkg/draftrag/search_routing.go` |
| T3.3 | done | `internal/application/subdecompose.go` |
| T4.1 | done | `internal/application/subdecompose_test.go`, `internal/infrastructure/decomposer/rule_test.go`, `internal/infrastructure/decomposer/composite_test.go`, `pkg/draftrag/search_builder_test.go` |
| T4.2 | done | `go vet` — clean, `go test ./...` — pass |

## Errors

- none

## Warnings

- none

## Next Step

- `speckeep archive sub-query-decomposition .`
