---
report_type: inspect
slug: public-examples
status: pass
docs_language: ru
generated_at: 2026-04-12
---

# Inspect Report: public-examples

## Scope

- snapshot: проверка спека на добавление production-ready примеров в README (таймауты/контекст, кеш, ретраи/CB, pgvector/Qdrant wiring)
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/public-examples/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- RQ-002/AC-003: “рекомендуемые таймауты” не зафиксированы числами в spec; уточнить конкретные значения на фазе plan, чтобы избежать двусмысленности в README.
- RQ-004: формулировка про Redis L2 (“при наличии места”) подразумевает optional-вставку; при планировании явно решить, будет ли короткий snippet/псевдокод для L2, чтобы не разъехаться с “ДОЛЖЕН”.

## Questions

- none

## Suggestions

- В plan зафиксировать конкретные таймауты (индексация vs запрос/ответ) и минимальный набор опций retry/CB, которые стоит показать в README как “безопасный старт”.
- Держать примеры копипастабельными: `context.Background()` только как parent, а все операции — через `context.WithTimeout(...); defer cancel()`.

## Traceability

- tasks отсутствуют; покрытие AC будет подтверждено на фазе `/speckeep.tasks`:
  - AC-001 -> TBD
  - AC-002 -> TBD
  - AC-003 -> TBD

## Next Step

- safe to continue to plan: `/speckeep.plan public-examples`

