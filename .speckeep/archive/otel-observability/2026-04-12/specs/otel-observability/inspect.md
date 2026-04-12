---
report_type: inspect
slug: otel-observability
status: pass
docs_language: ru
generated_at: 2026-04-12
---

# Inspect Report: otel-observability

## Scope

- snapshot: проверка спека на OTel-интеграцию для `draftrag.Hooks` (spans+metrics по стадиям) и README пример
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/otel-observability/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- AC-004: формулировка “быстро включить” (и похожие) может трактоваться как UX/SLO-обещание; в plan уточнить, что “быстро” = “не требует форка и минимальный код”, без обещания производительности.

## Questions

- none

## Suggestions

- В plan заранее зафиксировать: какие именно атрибуты и имена метрик/инструментов будут публичным контрактом (stable naming), чтобы users могли строить дашборды без зависимости от внутренней реализации.
- Явно выбрать, где будет жить интеграция: отдельный подпакет (например, `pkg/draftrag/otel`) и какие зависимости OTel допустимы (trace+metric).

## Traceability

- tasks отсутствуют; покрытие AC будет подтверждено на фазе `/speckeep.tasks`:
  - AC-001 -> TBD
  - AC-002 -> TBD
  - AC-003 -> TBD
  - AC-004 -> TBD

## Next Step

- safe to continue to plan: `/speckeep.plan otel-observability`

