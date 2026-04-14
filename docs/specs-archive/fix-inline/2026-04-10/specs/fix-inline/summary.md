---
slug: fix-inline
generated_at: 2026-04-10
---

## Goal

Исправить пропущенный маппинг `application.ErrFiltersNotSupported` → `draftrag.ErrFiltersNotSupported` в ветке `filter.Fields` метода `SearchBuilder.InlineCite`.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | InlineCite возвращает публичный ErrFiltersNotSupported | `errors.Is(err, draftrag.ErrFiltersNotSupported) == true` |
| AC-002 | Happy path InlineCite не сломан | Тесты на совместимом store проходят |
| AC-003 | Маппинг работает с обёрнутыми ошибками | `errors.Is` корректен при `fmt.Errorf("%w", ...)` |

## Out of Scope

- Другие методы SearchBuilder (уже корректны)
- Изменение сигнатуры InlineCite
- Рефакторинг routing-логики SearchBuilder
- Другие ошибочные маппинги (streaming, circuit breaker)
- Изменения в internal/application
