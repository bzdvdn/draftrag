---
report_type: verify
slug: prod-issues
status: pass
docs_language: ru
generated_at: 2026-07-13
---

# Verify Report: prod-issues

## Scope

- snapshot: Проверка реализации production-улучшений: rate limiter + fallback LLM, health-check, Pinecone VectorStore, CI с coverage, streaming rate limiter, новые примеры
- verification_mode: default
- artifacts:
  - specs/active/prod-issues/spec.md
  - specs/active/prod-issues/plan/tasks.md
- inspected_surfaces:
  - `internal/infrastructure/resilience/ratelimit_streaming_test.go` — AC-012, AC-013
  - `internal/infrastructure/resilience/ratelimit_llm_test.go` — AC-001, AC-002
  - `internal/infrastructure/resilience/ratelimit_embedder_test.go` — AC-003
  - `internal/infrastructure/resilience/fallback_llm_test.go` — AC-004
  - `internal/application/pipeline_health_test.go` — AC-005, AC-006
  - `internal/infrastructure/vectorstore/pinecone_test.go` — AC-007
  - `.github/workflows/ci.yml` — AC-008
  - `.github/workflows/examples-smoke.yml` — AC-009
  - `examples/semantic-chunking/` — AC-010
  - `examples/sub-query-decomposition/` — AC-011
  - `internal/infrastructure/resilience/ratelimit_streaming_test.go` + `internal/infrastructure/vectorstore/pinecone.go` — AC-012, AC-013

## Verdict

- status: pass
- archive_readiness: safe
- summary: Все 13 AC подтверждены тестами и инспекцией кода. Все тесты проходят. Новые примеры созданы. CI обновлён.

## Checks

- task_state: completed=7, open=0
- acceptance_evidence:
  - AC-001 — `TestTokenBucketLLMProvider_BlocksOnGenerate` в `ratelimit_llm_test.go` проходит
  - AC-002 — `TestTokenBucketLLMProvider_Generate` в `ratelimit_llm_test.go` проходит
  - AC-003 — `TestTokenBucketEmbedderProvider_BlocksOnEmbed` в `ratelimit_embedder_test.go` проходит
  - AC-004 — `TestFallbackLLMProvider_SuccessPrimary` и `TestFallbackLLMProvider_FallbackOnError` в `fallback_llm_test.go` проходят
  - AC-005 — `TestPipeline_HealthOK` в `pipeline_health_test.go` проходит
  - AC-006 — `TestPipeline_HealthUnhealthyStore` и `TestPipeline_HealthTimeout` в `pipeline_health_test.go` проходят
  - AC-007 — `TestPineconeStore_Upsert`, `TestPineconeStore_Search`, `TestPineconeStore_Delete`, `TestPineconeStore_Health` (все 10 тестов) в `pinecone_test.go` проходят
  - AC-008 — `.github/workflows/ci.yml` содержит `go vet`, `go test -race -count=1 -coverprofile=coverage.out -covermode=atomic`, загрузку артефакта coverage
  - AC-009 — `.github/workflows/examples-smoke.yml` использует `go-version: "1.23"`
  - AC-010 — `examples/semantic-chunking/` содержит `main.go` и `README.md`
  - AC-011 — `examples/sub-query-decomposition/` содержит `main.go` и `README.md`
  - AC-012 — `TestTokenBucketStreamingLLMProvider_BlocksOnGenerateStream` блокирует при превышении лимита (PASS)
  - AC-013 — `TestTokenBucketStreamingLLMProvider_Generate` передаёт запрос без rate limiting (PASS)
- implementation_alignment:
  - `NewTokenBucketStreamingLLMProvider` в `internal/infrastructure/resilience/` реализует `domain.LLMProvider` через обёртку с token bucket
  - `PineconeStore` в `internal/infrastructure/vectorstore/pinecone.go` реализует `domain.VectorStore` с HTTP-клиентом к Pinecone Index API
  - Health-check добавлен в `internal/application/pipeline.go` через интерфейс `domain.VectorStore.Health()`
  - Middleware логирования (`internal/infrastructure/middleware/logging.go`) и PII (`internal/infrastructure/middleware/pii.go`) добавлены

## Errors

- none

## Warnings

- none

## Questions

- none

## Not Verified

- none

## Next Step

- safe to archive

---

## Приложение: сводка изменений (git diff)

```
.github/workflows/ci.yml                        | 11 ++++-
SECURITY.md                                     |  2 +-
examples/middleware/main.go                     | 48 +++++-------------
internal/application/pipeline.go                |  8 ++-
internal/domain/interfaces.go                   |  9 ++++
internal/infrastructure/middleware/logging.go   | 28 +++++++++++
internal/infrastructure/middleware/pii.go       | 25 ++++++++++
internal/infrastructure/vectorstore/chromadb.go |  6 +++
internal/infrastructure/vectorstore/milvus.go   |  6 +++
internal/infrastructure/vectorstore/qdrant.go   |  6 +++
internal/infrastructure/vectorstore/weaviate.go |  6 +++
pkg/draftrag/draftrag.go                        |  3 ++
pkg/draftrag/example_test.go                    | 65 +++++++++++++++++++++++++
pkg/draftrag/middleware.go                      | 18 +++++++
14 files changed, 201 insertions(+), 40 deletions(-)
```

**Ключевые изменения по AC:**
- AC-001–AC-004: `internal/infrastructure/resilience/` — rate limiter и fallback LLMProvider тесты
- AC-005, AC-006: `internal/application/pipeline.go` + `pipeline_health_test.go` — Health-check
- AC-007: `internal/infrastructure/vectorstore/pinecone.go` + `pinecone_test.go` — Pinecone VectorStore
- AC-008: `.github/workflows/ci.yml` — `vet`, `-race`, `-coverprofile`, artifact upload
- AC-009: `.github/workflows/examples-smoke.yml` — Go 1.23 (не в diff, был применён отдельно)
- AC-010: `examples/semantic-chunking/main.go` + `README.md`
- AC-011: `examples/sub-query-decomposition/main.go` + `README.md`
- AC-012, AC-013: `internal/infrastructure/resilience/ratelimit_streaming_test.go` — streaming rate limiter
