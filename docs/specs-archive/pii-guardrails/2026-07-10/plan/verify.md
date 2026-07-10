---
report_type: verify
slug: pii-guardrails
status: pass
docs_language: ru
generated_at: 2026-07-10
---

# Verify Report: pii-guardrails

## Scope

- snapshot: PII detection + redaction на входе Index и выходе Query/Retrieve/Answer/Cite/RewrittenQuery
- verification_mode: default
- artifacts:
  - docs/specs/pii-guardrails/spec.md
  - docs/specs/pii-guardrails/plan.md
  - docs/specs/pii-guardrails/tasks.md
  - docs/specs/pii-guardrails/data-model.md
- inspected_surfaces:
  - internal/domain/pii.go
  - internal/infrastructure/piidetector/ (email, phone, ssn, creditcard, composite)
  - internal/application/pipeline.go (PipelineOptions.PIIDetector)
  - pkg/draftrag/draftrag.go (Index/Query/Retrieve redaction)
  - pkg/draftrag/search.go (SearchBuilder.Retrieve/Cite/InlineCite redaction)
  - pkg/draftrag/search_routing.go (RewrittenQuery redaction)
  - pkg/draftrag/pii.go (public API re-exports)
  - pkg/draftrag/pii_test.go (integration tests)
  - examples/pii-guardrails/main.go

## Verdict

- status: pass
- archive_readiness: ready
- summary: все AC покрыты тестами, все задачи выполнены; phone-детектор расширен для РФ-форматов (+7-900-XXX-XX-XX); добавлен изолированный тест AC-007

## Verification Matrix

| AC-ID | Task IDs | Evidence | Verdict |
|-------|----------|----------|---------|
| AC-001 | T1.1, T2.1, T2.2, T2.4 | TestPIIRedactIndex: pass | pass |
| AC-002 | T2.3, T2.4 | TestPIIRedactQuery: pass | pass |
| AC-003 | T3.1, T4.1 | TestPIIRedactCite: pass | pass |
| AC-004 | T1.2, T2.2, T2.4 | TestPIIRedactSelectiveCategories: pass | pass |
| AC-005 | T1.1, T3.4, T4.1 | TestPIIRedactCustomDetector: pass | pass |
| AC-006 | T2.2, T2.3, T2.4 | TestPIIRedactBackwardCompat: pass | pass |
| AC-007 | T3.2, T4.1, T5.2 | TestPIIRedactRewrittenQuery: pass, TestPIIRedactRewrittenQueryPipelineLevel: pass | pass |

## Checks

- task_state: completed=15, open=0
- acceptance_evidence:
  - AC-001: Index redaction — store.Search показывает `<redacted>` вместо PII (TestPIIRedactIndex)
  - AC-002: Query redaction — результат Query не содержит PII (TestPIIRedactQuery)
  - AC-003: Cite/Answer — источники Cite не содержат PII (TestPIIRedactCite)
  - AC-004: Selective categories — только email redacted, phone остаётся (TestPIIRedactSelectiveCategories)
  - AC-005: Custom detector — passportDetector в тесте корректно срабатывает (TestPIIRedactCustomDetector)
  - AC-006: Backward compat — nil PIIDetector не меняет содержимое (TestPIIRedactBackwardCompat)
  - AC-007: RewrittenQuery redaction — изолированные тесты TestPIIRedactRewrittenQuery и TestPIIRedactRewrittenQueryPipelineLevel (per-request и pipeline-level rewriter)
- implementation_alignment:
  - DEC-001 выполнен: PII-детекция на публичном слое (pkg/draftrag), application.Pipeline чист
  - DEC-002 выполнен: CompositePIIDetector композирует под-детекторы
  - DEC-003 выполнен: RewrittenQuery redaction в rewriterResult (post-factum)
- traceability:
  - 38+ маркеров @sk-task/@sk-test найдены (trace script подтверждает)
  - Все 15 задач имеют trace-маркеры в коде и тестах
- code_quality:
  - `go vet ./...` — без ошибок
  - `golangci-lint` — без ошибок
  - `go test ./...` — все тесты проходят
  - BenchmarkPIIDetectors: ~10µs/op, 1.1 KB/op, 18 allocs/op

## Resolved Concerns

1. **Phone detector для РФ-форматов** — добавлен multi-segment E.164 паттерн (3-я альтернатива в `phone.go`). Подтверждено тестами `TestPhoneDetector/RF_format`, `/UK_format`, `/DE_format`, `/RF_with_parens`.
2. **AC-007 (RewrittenQuery)** — добавлены изолированные тесты `TestPIIRedactRewrittenQuery` (per-request rewriter) и `TestPIIRedactRewrittenQueryPipelineLevel` (pipeline-level rewriter).

## Not Verified

- Streaming-методы (Stream, StreamSources, StreamCite) — out of scope согласно spec
- Metadata redaction — out of scope
- PII-детекция на ML-based/API-based — out of scope
- Производительность на документах >1 MB (benchmark только для ~150B текста)

## Next Step

- Все concerns устранены, verify: pass.
- `speckeep archive pii-guardrails .`
