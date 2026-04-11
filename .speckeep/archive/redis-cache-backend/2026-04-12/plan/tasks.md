# Redis backend для кэша эмбеддингов Задачи

## Phase Contract

Inputs: `.speckeep/specs/redis-cache-backend/plan/plan.md`, `.speckeep/specs/redis-cache-backend/plan/data-model.md`, `.speckeep/specs/redis-cache-backend/spec.md`.
Outputs: упорядоченные исполнимые задачи с покрытием критериев `AC-*`.
Stop if: хотя бы один `AC-*` нельзя сопоставить с выполнимой задачей.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/embedder/ | `T1.1, T2.1, T2.2, T2.3, T2.4, T3.1` |
| internal/infrastructure/embedder/cache/ | `T1.1, T2.1, T2.2, T2.3, T2.4, T3.1` |
| internal/infrastructure/embedder/cache/cache.go | `T2.2, T2.3, T2.4` |
| internal/infrastructure/embedder/cache/options.go | `T1.1` |
| internal/infrastructure/embedder/cache/redis.go | `T2.1` |
| internal/infrastructure/embedder/cache/cache_test.go | `T3.1` |
| internal/infrastructure/embedder/cache/redis_test.go | `T3.1` |
| pkg/draftrag/cached_embedder.go | `T1.2` |
| README.md | `T3.2` |
| docs/embedders.md | `T3.2` |

## Фаза 1: API и каркас

Цель: зафиксировать публичную конфигурацию и минимальные интерфейсы, чтобы реализация и тесты не “плыли”.

- [x] T1.1 Перевести Redis-клиент на адаптер-интерфейс `GetBytes/SetBytes` и расширить internal опции (ttl, prefix, default prefix) — AC-005. Touches: internal/infrastructure/embedder/cache/options.go
- [x] T1.2 Расширить публичные `CacheOptions` и `NewCachedEmbedder`, чтобы включать Redis через адаптер-интерфейс без import internal пакетов — AC-001, AC-005. Touches: pkg/draftrag/cached_embedder.go

## Фаза 2: Основная реализация Redis L2

Цель: реализовать поведение L2 (Redis) с warming L1 и безопасной деградацией (treat-as-miss + логирование).

- [x] T2.1 Реализовать `redisCache` (key builder с prefix, msgpack encode/decode `[]float64`, TTL) и API `Get/Set` — AC-005, AC-006. Touches: internal/infrastructure/embedder/cache/redis.go
- [x] T2.2 Интегрировать L2 read path в `EmbedderCache.Embed`: L1 miss → Redis GET; при hit вернуть значение без вызова embedder; warming L1 — AC-002, AC-003. Touches: internal/infrastructure/embedder/cache/cache.go
- [x] T2.3 Реализовать поведение treat-as-miss: ошибки Redis GET/SET и decode → логирование и fallback на embedder без ошибки `Embed` — AC-004, AC-006. Touches: internal/infrastructure/embedder/cache/cache.go, internal/infrastructure/embedder/cache/redis.go
- [x] T2.4 Реализовать best-effort write path: при miss после вызова embedder — запись в Redis с TTL и prefix, ошибки только логируются — AC-004, AC-005. Touches: internal/infrastructure/embedder/cache/cache.go

## Фаза 3: Доказательства и упаковка

Цель: доказать критерии приемки unit-тестами и (если нужно) дать пользователю понятный usage.

- [x] T3.1 Добавить unit-тесты для AC-001..AC-006 (fake redis client + mock embedder; проверка счётчиков вызовов, ключей, TTL, warming L1, treat-as-miss) — AC-001..AC-006. Touches: internal/infrastructure/embedder/cache/cache_test.go, internal/infrastructure/embedder/cache/redis_test.go
- [x] T3.2 Документировать пример включения Redis и написания адаптера (опционально, если без этого API неочевиден) — AC-001. Touches: README.md, docs/embedders.md

## Покрытие критериев приемки

- AC-001 -> T1.2, T3.1, T3.2
- AC-002 -> T2.2, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T2.3, T2.4, T3.1
- AC-005 -> T1.1, T1.2, T2.1, T2.4, T3.1
- AC-006 -> T2.1, T2.3, T3.1
