# Observability: хуки/метрики для pipeline стадий (v1) — Модель данных

## Scope

- Связанные `AC-*`: `AC-001`, `AC-002`
- Изменение модели данных — только добавление типов для hooks; персистентных данных нет.

## Сущности

### DM-001 HookEvent

- Назначение: payload события для hooks на стадии pipeline.
- Поля:
  - `Operation` - `string`, например `Answer`, `Index`, `Query`.
  - `Stage` - enum/константа (`Chunk`, `Embed`, `Search`, `Generate`).
  - `StartTime`/`Duration` - `time.Time`/`time.Duration` (в end событии достаточно Duration).
  - `Err` - `error` (может быть nil).

## Вне scope

- Встроенная интеграция с Prometheus/OTel.
- Асинхронные хуки/батчинг.

