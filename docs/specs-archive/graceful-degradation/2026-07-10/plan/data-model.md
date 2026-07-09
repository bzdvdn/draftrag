# Graceful Degradation Модель данных

## Scope

- Связанные `AC-*`: AC-008
- Связанные `DEC-*`: DEC-001
- Статус: `no-change`

## No-Change Stub

- Статус: `no-change`
- Причина: фича не добавляет и не меняет persisted entities, value objects, state transitions или contract-relevant payload shapes. `FallbackStats` — runtime-структура (in-memory счётчики, не сериализуется), `ErrAllProvidersFailed` — sentinel error.
- Revisit triggers:
  - появляется сохраняемое состояние fallback-цепи
  - FallbackStats начинает экспортироваться в метрики/Prometheus