# arch-generics: Модель данных

## Scope

- Статус: `no-change`
- Причина: фича не добавляет и не меняет persisted entities, value objects, state transitions или API/event payload shapes. Все изменения — исключительно в организации и типизации существующего кода (рефакторинг handler maps, замена panic на error return).

## No-Change Stub

- Статус: `no-change`
- Причина: refactoring-only — ни одна сущность (Document, Chunk, Query, RetrievalResult) не меняется
- Revisit triggers:
  - появляется новое сохраняемое состояние или модель
  - API/event payload shape требует отслеживания
