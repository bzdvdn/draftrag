---
report_type: inspect
slug: eval-harness-basic
status: pass
docs_language: ru
generated_at: 2026-04-08
---

# Inspect Report: eval-harness-basic

## Scope

- snapshot: проверена спецификация на достаточность для планирования и соответствие конституции (addivitve API, детерминированные метрики, тесты без сети)
- artifacts:
  - .draftspec/constitution.md
  - .draftspec/specs/eval-harness-basic/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- none

## Questions

- В v1 считаем метрики по `ParentID` (устойчивее к чанкингу) или по `Chunk.ID`? (предложение: поддержать оба варианта через enum/опцию, но в MVP протестировать `ParentID` как дефолт)

## Suggestions

- Держать harness библиотечным (пакет `pkg/draftrag/eval`), без CLI, с чистыми моделями данных.
- Встроить подробный per-case отчёт (got ranks, matched ids) — это сильно упрощает дебаг.

## Traceability

- AC-001: будет покрыт синтетическим датасетом с заранее известными позициями релевантных источников; проверяем hit@k и MRR.
- AC-002: добавление нового пакета/типов аддитивно, `go test ./...` остаётся зелёным.

## Next Step

- safe to continue to plan

