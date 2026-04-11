---
report_type: verify
slug: retrieval-deduplication
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Verify Report: retrieval-deduplication

## Scope

- snapshot: проверена opt-in дедупликация retrieval sources по ParentID (без изменения поведения по умолчанию) и тестовое покрытие
- verification_mode: default
- artifacts:
  - .draftspec/constitution.md
  - .draftspec/specs/retrieval-deduplication/spec.md
  - .draftspec/plans/retrieval-deduplication/tasks.md
- inspected_surfaces:
  - pkg/draftrag/draftrag.go
  - internal/application/pipeline.go
  - internal/application/retrieval_deduplication_test.go

## Verdict

- status: pass
- archive_readiness: safe
- summary: дедупликация включается опционально, по умолчанию поведение не меняется, `go test ./...` проходит

## Checks

- task_state: completed=5, open=0
- acceptance_evidence:
  - AC-001 -> unit-тест `TestPipeline_Query_DedupByParentID_Enabled` подтверждает, что при включённой опции остаётся ≤1 chunk на ParentID и выбирается лучший по score
  - AC-002 -> unit-тест `TestPipeline_Query_DedupByParentID_Disabled_NoChanges` подтверждает, что при выключенной опции `RetrievalResult.Chunks` не меняется; `go test ./...` проходит
- implementation_alignment:
  - опция `PipelineOptions.DedupSourcesByParentID` прокидывается в application config и применяется на retrieval-result перед использованием в prompt/возвратом evidence

## Errors

- none

## Warnings

- v1 дедупликация применяется к retrieval результату в application слое; если пользователю нужен “raw retrieval” без дедуп, он должен не включать опцию.

## Questions

- none

## Not Verified

- Дедупликация на уровне `VectorStore` (это вне scope).

## Next Step

- safe to archive

