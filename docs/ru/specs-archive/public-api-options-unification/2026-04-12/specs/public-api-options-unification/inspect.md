---
report_type: inspect
slug: public-api-options-unification
status: concerns
docs_language: ru
generated_at: 2026-04-12T12:06:03+03:00
---

# Inspect Report: public-api-options-unification

## Scope

- snapshot: проверка спеки на консистентность, проверяемость и удержание границ (публичный API options pattern)
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/public-api-options-unification/spec.md

## Verdict

- status: concerns

## Errors

- none

## Warnings

- В спеке есть предупреждение ambiguity (“быстр”) от helper-скрипта; в тексте это скорее риторика (“ускоряет onboarding”), но лучше избегать подобных формулировок в acceptance-части и держать критерии наблюдаемыми.
- `## Открытые вопросы` содержит ключевое архитектурное решение (какой паттерн canonical). Это нормально для spec, но downstream планирование будет заблокировано, если решение не зафиксировать в `DEC-*` на фазе plan.

## Questions

- Какой паттерн выбираем canonical для публичного API: `Options struct` (как сейчас в большинстве `pkg/draftrag`) или functional options? Нужен один выбор, иначе AC-001/AC-005 будут не верифицируемы.

## Suggestions

- В plan сразу добавить `DEC-001` с выбором canonical паттерна и правилами backward compatibility (например: “struct Options остаётся canonical; functional options допускаются только internal” или наоборот).
- В plan определить формат guardrail (AC-005): простой unit-test/rg check, который перечисляет публичные `New*` и проверяет сигнатуры/наличие `...Options`, чтобы консистентность была measurable.

## Traceability

- tasks.md отсутствует — traceability `AC-* -> T*` появится после `/speckeep.tasks public-api-options-unification`.

## Next Step

- safe to continue to plan: `/speckeep.plan public-api-options-unification`

