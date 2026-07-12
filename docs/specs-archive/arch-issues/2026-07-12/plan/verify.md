---
report_type: verify
slug: arch-issues
status: pass
docs_language: ru
generated_at: 2026-07-12
---

# Verify Report: arch-issues

## Scope

- snapshot: 4 architectural hardening workstreams (PII, Health/Shutdown, Tool calling, Router codegen) — 20 задач, 8 AC
- verification_mode: default
- artifacts:
  - .speckeep/constitution.summary.md
  - docs/specs/arch-issues/spec.md
  - docs/specs/arch-issues/tasks.md
- inspected_surfaces:
  - internal/domain/models.go (ToolDefinition, ToolCall, ToolResult)
  - internal/domain/interfaces.go (ToolCallingLLMProvider)
  - internal/application/pipeline.go (ErrPipelineClosed, redact, Health, Close, ExecuteWithTools)
  - internal/application/query.go (PII redact in 6 Query variants)
  - internal/application/answer.go (PII redact in 6 Answer variants)
  - pkg/draftrag/draftrag.go (PII delegation, Health/Close facade, type aliases)
  - pkg/draftrag/errors.go (ErrToolsNotSupportedInStream)
  - pkg/draftrag/search.go (Tools, ToolHandler methods)
  - pkg/draftrag/search_routing.go (routeTools, pickRoute, handler functions)
  - pkg/draftrag/search_routes_gen.go (7 generated handler maps)
  - pkg/draftrag/routergen/routes.go (route table), main.go (generator)
  - pkg/draftrag/gen.go (go:generate directive)
  - Makefile (generate-router target)
  - internal/application/pipeline_pii_test.go, pipeline_health_test.go, pipeline_close_test.go
  - pkg/draftrag/draftrag_pii_test.go, search_builder_test.go
  - pkg/draftrag/search_routes_gen_test.go
  - pkg/draftrag/routergen/routergen_test.go

## Verdict

- status: pass
- archive_readiness: safe
- summary: все 20 задач выполнены, все 8 AC подтверждены observable evidence, тесты arch-issues проходят с -race

## Verification Matrix

| AC-ID | Task IDs | Evidence | Verdict |
|-------|----------|----------|---------|
| AC-001 | T2.1, T2.3 | `p.redact()` во всех Query/Answer методах + `processDocumentOp`; `TestPipeline_PIIRedact_Index/TestPipeline_PIIRedact_Query/TestPipeline_PIIRedact_Answer` с mock PIIDetector — `go test -race -run TestPipeline_PII ./internal/application/...` pass | pass |
| AC-002 | T2.2, T2.3 | `draftrag.go:537-540` — `redactRetrievalResult` делегирует в `p.core.RedactRetrievalResult`; нет вызовов `piidetector.Detect()` в `draftrag.go`; `TestPublicPII*` с counter-обёрткой — `go test -race -run TestPublicPII ./pkg/draftrag/...` pass | pass |
| AC-003 | T4.1, T4.2, T4.6 | `internal/domain/interfaces.go:109` — `ToolCallingLLMProvider`; `pipeline.go:370` — `ExecuteWithTools`; `search_builder_test.go:468-539` — mock ToolCallingLLMProvider + 4 tool execution subtests pass | pass |
| AC-004 | T4.3, T4.4, T4.5, T4.6 | `search_routing.go:25` — `routeTools` + `pickRoute` case; `search.go:134-141` — `Tools()`/`ToolHandler()`; routeTools во всех 7 generated handler maps; `search_builder_test.go:468-618` — 10 subtests pass | pass |
| AC-005 | T5.1, T5.2, T5.4 | `search_routes_gen.go:1` — `// Code generated` header; `search_routing.go` — нет map-литералов (только var-ссылки на generated maps); `gen.go:3` — `//go:generate go run ./routergen/`; `TestGeneratedMaps_HaveCorrectEntryCount/TestGeneratedMaps_AllRoutesPresent/TestGeneratedMaps_RouterIntegration` pass | pass |
| AC-006 | T5.1, T5.4 | `routergen/routes.go` — единая таблица (9 entries × 7 output columns); `routergen/routergen_test.go` — 4 subtests (route table consistency, unique names, non-empty output); `search_routes_gen_test.go:14` — 9 entries во всех 7 maps | pass |
| AC-007 | T3.1, T3.3 | `pipeline.go:181` — `Health()` c fan-out store/llm/embedder + `errors.Join`; `pipeline_health_test.go` — 3 subtests (OK, unhealthy store, timeout) pass | pass |
| AC-008 | T3.1, T3.2, T3.3 | `pipeline.go:202` — `Close()` с `sync.Once`; `draftrag.go:547` — Health facade; closed guard в Index/Query/Answer/UpdateDocument; `pipeline_close_test.go` — 5 subtests (sentinel, double close, guard × 3) pass | pass |

## Checks

- task_state: completed=20, open=0
- acceptance_evidence: см. Verification Matrix
- implementation_alignment:
  - PII: `application.Pipeline.redact()` вызывается во всех Query (query.go:185,254,331,412,504) и Answer (answer.go:35,151,295,395,489) variants
  - ToolCallingLLMProvider: optional type assertion в `ExecuteWithTools` (pipeline.go:370)
  - Router gen: `go generate ./pkg/draftrag/...` → `search_routes_gen.go` (87 строк, 7 map-литералов, 9 entries each)
  - Health/Close: `Health()` fan-out (1s timeout, errors.Join); `Close()` sentinel + closed guard
  - Data race fix: `recordingStore` в `pipeline_answer_test.go` — добавлен `sync.Mutex`, `go test -race -count=1 ./internal/application/...` pass

## Errors

- none (arch-issues scope)

## Warnings

- `verify-task-state.sh` обнаружил `Touches:` пути без полного qualified path (e.g. `Query`, `Answer)` — косметические проблемы в tasks.md, не влияющие на реализацию
- `golangci-lint` сообщает о pre-existing замечаниях из других spec (comment format, `resp.Body.Close`, `context-as-argument`, unused params) — не связаны с arch-issues

## Questions

- none

## Not Verified

- (none — все проверки пройдены)

## Traceability Gaps

- T4.2: `@sk-task arch-issues#T4.2: tool execution loop (AC-003)` присутствует в `pipeline.go:370` ✅
- T5.1: `@sk-task arch-issues#T5.1: route table for code generator` присутствует в `routes.go:3` и `main.go:11` ✅
- Все `@sk-test` маркеры присутствуют над owning функциями — gaps не обнаружены ✅

## Next Step

- safe to archive

Готово к: speckeep archive arch-issues .
