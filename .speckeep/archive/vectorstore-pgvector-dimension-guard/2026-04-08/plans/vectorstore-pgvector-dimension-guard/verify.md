---
report_type: verify
slug: vectorstore-pgvector-dimension-guard
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Verify Report: vectorstore-pgvector-dimension-guard

## Scope

- snapshot: проверена типизированная ошибка несоответствия размерности embeddings для pgvector store (Upsert/Search) и её доступность через `errors.Is`
- verification_mode: default
- artifacts:
  - .draftspec/constitution.md
  - .draftspec/specs/vectorstore-pgvector-dimension-guard/spec.md
  - .draftspec/plans/vectorstore-pgvector-dimension-guard/tasks.md
- inspected_surfaces:
  - internal/domain/models.go
  - pkg/draftrag/errors.go
  - internal/infrastructure/vectorstore/pgvector.go
  - pkg/draftrag/pgvector.go
  - pkg/draftrag/pgvector_dimension_guard_test.go

## Verdict

- status: pass
- archive_readiness: safe
- summary: dimension mismatch классифицируется через sentinel error (errors.Is), тесты проходят без внешней сети

## Checks

- task_state: completed=6, open=0
- acceptance_evidence:
  - AC-001 -> `internal/infrastructure/vectorstore/pgvector.go` возвращает wrap `%w` на `domain.ErrEmbeddingDimensionMismatch`; unit-тест `pkg/draftrag/pgvector_dimension_guard_test.go` проверяет `errors.Is(..., draftrag.ErrEmbeddingDimensionMismatch)` для Upsert и Search
  - AC-002 -> existing `go test ./...` проходит; unit-тест проверяет, что при корректной размерности ошибка mismatch не возникает (даже если БД-операция затем падает в тестовом драйвере)
- implementation_alignment:
  - `draftrag.ErrEmbeddingDimensionMismatch` re-export’ится из domain и может использоваться пользователем как стабильный классификатор

## Errors

- none

## Warnings

- Happy-path тест проверяет только отсутствие классификации mismatch (и не выполняет реальный SQL), что соответствует требованиям “без внешней сети/БД”.

## Questions

- none

## Not Verified

- Реальная pgvector инсталляция/SQL-выполнение в рамках этой фичи (это покрывается opt-in DSN тестами и отдельной инфраструктурной проверкой).

## Next Step

- safe to archive

