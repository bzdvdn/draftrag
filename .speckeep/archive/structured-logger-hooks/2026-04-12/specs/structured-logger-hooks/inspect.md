---
report_type: inspect
slug: structured-logger-hooks
status: concerns
docs_language: ru
generated_at: 2026-04-12T02:33:44+03:00
---

# Inspect Report: structured-logger-hooks

## Scope

- snapshot: проверка спеки на наблюдаемость-улучшение (структурный логгер/хуки) без расширения scope
- artifacts:
  - .speckeep/constitution.md
  - .speckeep/specs/structured-logger-hooks/spec.md

## Verdict

- status: concerns

## Errors

- none

## Warnings

- AC-004 требует изоляции от паник в логгере; в spec это заявлено корректно, но в plan важно зафиксировать явную стратегию (например, `recover`-wrapper вокруг вызова логгера), иначе implement-фаза может “угадывать” форму защиты.

## Questions

- Нужна ли единая схема полей (минимальный обязательный набор ключей) или достаточно зафиксировать только “минимум” из AC-002/AC-003?

## Suggestions

- В plan явно выбрать форму публичного интерфейса логгера: либо “универсальный” `Log(ctx, level, msg, fields...)`, либо `Debug/Info/Warn/Error` методы; и определить как передаются поля (пары `key,value` vs typed attrs).
- В plan зафиксировать правило “логгер никогда не паникует наружу”: единый helper (`safeLog`) и где он применяется (cache + resilience).

## Traceability

- tasks.md отсутствует — traceability `AC-* -> T*` появится после `/speckeep.tasks structured-logger-hooks`.

## Next Step

- safe to continue to plan: `/speckeep.plan structured-logger-hooks`

