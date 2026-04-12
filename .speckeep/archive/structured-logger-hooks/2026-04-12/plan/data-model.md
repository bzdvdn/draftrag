# Structured logger hooks — Data Model

## Logger

Минимальный публичный интерфейс (без внешних зависимостей), переэкспортируемый из `pkg/draftrag`:

- `Logger` — один метод `Log(ctx, level, msg, fields...)`
- `LogLevel` — уровни (минимум: `debug`, `info`, `warn`, `error`)
- `LogField` — структурированное поле `{Key string, Value any}`

## Safe Logging

Инвариант: логирование best-effort и не ломает основной поток.

- Любой call site должен использовать единый safe wrapper, который:
  - проверяет `logger != nil`
  - вызывает `logger.Log(...)`
  - защищён `recover`, чтобы паника логгера не “пробивала” наружу

## Event Schema (минимум)

Общий набор полей для фильтрации:

- `component`:
  - `embedder_cache` (кэш эмбеддингов, Redis L2)
  - `resilience_retry` (retry + circuit breaker)
- `operation`:
  - `redis_get`, `redis_set`, `redis_decode`
  - `embed`, `generate`
- Доп. поля по месту:
  - `attempt` (int) — для retry
  - `rejected` (bool) — для CB rejection
  - `err` (error/строка) — для ошибок
  - `key_prefix` (string) — при наличии (Redis keyspace)

