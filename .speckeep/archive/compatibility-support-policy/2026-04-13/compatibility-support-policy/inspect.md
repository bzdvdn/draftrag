---
report_type: inspect
slug: compatibility-support-policy
status: pass
docs_language: ru
generated_at: 2026-04-12
---

# Inspect Report: compatibility-support-policy

## Scope

- snapshot: проверка спека на документ политики поддержки/совместимости (Go versions, semver/deprecation, матрицы backend’ов и возможностей) и ссылку из README
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/compatibility-support-policy/spec.md

## Verdict

- status: pass

## Errors

- none

## Warnings

- RQ-002/AC-002: правило поддержки Go-версий требует конкретики (“N последних minor” или иной явный принцип) — зафиксировать в plan, чтобы не оставить двусмысленность.
- RQ-003/RQ-004: матрицы должны отражать текущее состояние репозитория; в plan определить минимальный набор backend’ов и фич для таблиц, чтобы не расползтись.

## Questions

- none

## Suggestions

- В plan заранее выбрать путь документа (например, `docs/compatibility.md`) и структуру: `Go`, `SemVer & Deprecation`, `Backends`, `Features`, `Support window`.
- Указать, где будет “источник истины” для матриц (docs + необходимость обновлять в релизах/PR’ах).

## Traceability

- tasks отсутствуют; покрытие AC будет подтверждено на фазе `/speckeep.tasks`:
  - AC-001 -> TBD
  - AC-002 -> TBD
  - AC-003 -> TBD

## Next Step

- safe to continue to plan: `/speckeep.plan compatibility-support-policy`

