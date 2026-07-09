# CachedEmbedder Public API

## Scope Snapshot

- In scope: публичная обёртка `CachedEmbedder` над `internal/infrastructure/embedder/cache` в `pkg/draftrag`.
- Out of scope: Redis second-level cache (из внутренней спеки embedding-cache), новая кэш-логика.

## Цель

Разработчики получают доступ к LRU-кэшированию embedding-вызовов через публичный API без импорта `internal/` пакетов. Снижает затраты на API при повторных запросах и переиндексации.

## Основной сценарий

1. `cached, err := draftrag.NewCachedEmbedder(base, draftrag.CacheOptions{MaxSize: 1000})`
2. Передаётся в `NewPipeline(store, llm, cached)` вместо base embedder.
3. Повторные embed-вызовы с одинаковым текстом → cache hit, без API.
4. `cached.Stats()` — hits, misses, hit rate.

## Scope

- `pkg/draftrag/cached_embedder.go` — новый файл
- `CacheOptions` struct, `CachedEmbedder` struct, `EmbedCacheStats` type alias
- Нетронутым остаётся `internal/infrastructure/embedder/cache`

## Требования

- **RQ-001** `NewCachedEmbedder(nil, ...)` → ошибка (не panic).
- **RQ-002** `CachedEmbedder` реализует `Embedder` интерфейс.
- **RQ-003** Повторный `Embed` с тем же текстом → cache hit (base не вызывается второй раз).
- **RQ-004** `Stats()` возвращает корректные Hits, Misses, HitRate.
- **RQ-005** Ошибка base embedder'а пробрасывается без потери (`errors.Is`).
- **RQ-006** `CacheOptions.MaxSize = 0` → дефолтный размер (не нулевой кэш).

## Вне scope

- Redis second-level cache.
- TTL для кэш-записей (внутренняя реализация поддерживает, но не экспонируется в v1).
- Инвалидация кэша вручную.

## Критерии приемки

### AC-001 Cache hit

- **Given** `CachedEmbedder` с `countEmbedder` (считает вызовы)
- **When** `Embed("hello")` дважды
- **Then** `countEmbedder.calls == 1`
- **Evidence**: `TestCachedEmbedder_CachesResults` pass

### AC-002 Stats корректны

- **Given** 1 miss + 1 hit + 1 miss
- **When** `Stats()`
- **Then** Hits=1, Misses=2, HitRate≈0.33
- **Evidence**: `TestCachedEmbedder_Stats` pass

### AC-003 Nil base error

- **Given** `NewCachedEmbedder(nil, ...)`
- **Then** возвращает ошибку
- **Evidence**: `TestCachedEmbedder_NilBaseError` pass

### AC-004 Error propagation

- **Given** base embedder всегда возвращает ошибку
- **When** `Embed`
- **Then** `errors.Is(err, wantErr)` true
- **Evidence**: `TestCachedEmbedder_PropagatesError` pass

## Допущения

- Внутренняя реализация LRU уже корректно протестирована.
- Thread-safety обеспечивается внутренним `cache.EmbedderCache`.

## Открытые вопросы

- none
