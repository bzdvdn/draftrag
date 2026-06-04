---
report_type: verify
slug: pipeline-e2e-benchmarks
status: pass
docs_language: ru
generated_at: 2026-06-04
---

# Verify Report: pipeline-e2e-benchmarks

## Scope

- snapshot: 3 benchmark-функции (Index, Query, Full) с sub-benchmarks для 10/100/1000 docs, benchmem, short mode
- verification_mode: default
- artifacts:
  - docs/specs/pipeline-e2e-benchmarks/spec.md
  - docs/specs/pipeline-e2e-benchmarks/plan.md
  - docs/specs/pipeline-e2e-benchmarks/tasks.md
  - docs/specs/pipeline-e2e-benchmarks/data-model.md
- inspected_surfaces:
  - pkg/draftrag/pipeline_bench_test.go — bench helpers, 3 Benchmark-функции

## Verdict

- status: pass
- archive_readiness: safe
- summary: 3+ benchmarks работают, short mode <1s, go vet clean

## Checks

- task_state: completed=5, open=0
- acceptance_evidence:
  - AC-001 -> `go test -bench=BenchmarkPipelineE2E_Index -benchmem` PASS — ns/op, B/op, allocs/op для 3 размеров
  - AC-002 -> `go test -bench=BenchmarkPipelineE2E_Query -benchmem` PASS — ns/op, B/op, allocs/op для 3 размеров
  - AC-003 -> `go test -bench=BenchmarkPipelineE2E_Full -benchmem` PASS — ns/op, B/op, allocs/op
  - AC-004 -> `go test -bench=PipelineE2E -benchmem -short` <6s (3 sub-benchmarks, docs10 only)
  - AC-005 -> `go vet ./pkg/draftrag/` exit 0
- implementation_alignment:
  - pipeline_bench_test.go: benchEmbedder, benchLLM, genDocs, setupBenchPipeline — helper types
  - BenchmarkPipelineE2E_Index: 3 sub-benchmarks (docs10/docs100/docs1000) с b.ReportAllocs
  - BenchmarkPipelineE2E_Query: 3 sub-benchmarks с prepopulated store
  - BenchmarkPipelineE2E_Full: 3 sub-benchmarks с полным циклом Index+Query
  - Short mode: `if testing.Short() && n > 10 { continue }`

## Errors

- none

## Warnings

- docs1000 в Index/Full медленный (~100s на 1 итерацию) — ожидаемо для 6250 chunk/embed/upsert операций
- Для CI рекомендуется `-short` или `-bench=.../docs10$`

## Questions

- none

## Not Verified

- benchstat variance <5% — требует 3+ последовательных прогона (>5 мин, не запускалось)
- Parallel benchmarks — вне scope

## Next Step

- safe to archive
