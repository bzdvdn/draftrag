---
report_type: verify
slug: fuzz-property-tests
status: pass
docs_language: ru
generated_at: 2026-06-04
---

# Verify Report: fuzz-property-tests

## Scope

- snapshot: 4 fuzz-функции (3 domain + 1 SearchBuilder) + 1 property roundtrip-тест VectorStore
- verification_mode: default
- artifacts:
  - docs/specs/fuzz-property-tests/spec.md
  - docs/specs/fuzz-property-tests/plan.md
  - docs/specs/fuzz-property-tests/tasks.md
  - docs/specs/fuzz-property-tests/data-model.md
- inspected_surfaces:
  - internal/domain/fuzz_test.go — FuzzValidateDocument, FuzzValidateChunk, FuzzValidateQuery
  - pkg/draftrag/fuzz_test.go — FuzzSearchBuilderValidate
  - pkg/draftrag/roundtrip_test.go — TestVectorStoreRoundtrip

## Verdict

- status: pass
- archive_readiness: safe
- summary: 4 fuzz-функции 0 panics за 15s, roundtrip 100/100 PASS, go vet clean

## Checks

- task_state: completed=4, open=0
- acceptance_evidence:
  - AC-001 -> `go test -fuzz=FuzzValidate -fuzztime=15s ./internal/domain/` — 5.7M execs, 0 panics, PASS
  - AC-002 -> `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=10s ./pkg/draftrag/` — 439K execs, 57 new interesting, 0 panics, PASS
  - AC-003 -> `go test -run TestVectorStoreRoundtrip -count=100 ./pkg/draftrag/` — 0 failures, PASS
  - AC-004 -> seed-корпуса заданы через f.Add() во всех 4 fuzz-функциях (пустые строки, unicode, null bytes, MinInt/MaxInt)
  - AC-005 -> `go vet ./internal/domain/ ./pkg/draftrag/` exit 0
- implementation_alignment:
  - domain/fuzz_test.go: 3 fuzz-функции с seed corpora (10 seeds each), f.Add → f.Fuzz
  - draftrag/fuzz_test.go: FuzzSearchBuilderValidate с оптимизированным setup (store/pipeline однажды)
  - draftrag/roundtrip_test.go: 100 итераций, random Chunk → Upsert → Search → ID match

## Errors

- none

## Warnings

- none

## Not Verified

- Fuzz-тесты для HTTP-клиентов — вне scope
- Parallel fuzzing — Go fuzzer сам управляет

## Next Step

- safe to archive
