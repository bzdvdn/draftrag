# Рефакторинг SearchBuilder: Generics + единый routing — Модель данных

## Scope

- Связанные AC: AC-001, AC-002, AC-003
- Связанные решения: DEC-001
- Статус: `no-change`

## No-Change Stub

- **Статус:** `no-change`
- **Причина:** фича не добавляет и не меняет persisted entities, value objects, state transitions или contract-relevant payload shapes. Новые типы — исключительно implementation detail:
  - `router[T any]` — internal generic struct в пакете `draftrag`
  - 7 result-structs (`rRetrieve`, `rAnswer`, `rCite`, `rInlineCite`, `rStream`, `rStreamSources`, `rStreamCite`) — внутренние, не экспортируемые
- **Revisit triggers:**
  - ни один из триггеров не применим — модель данных не затрагивается
