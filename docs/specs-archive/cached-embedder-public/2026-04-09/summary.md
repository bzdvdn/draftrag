# Сводка архива

## Спецификация

- snapshot: экспортирован CachedEmbedder (LRU-кэш embedding вызовов) в pkg/draftrag
- slug: cached-embedder-public
- archived_at: 2026-04-09
- status: completed

## Причина

Внутренняя LRU-реализация кэширования эмбеддингов была недоступна пользователям библиотеки. Публичная обёртка открывает её без дублирования логики.

## Результат

- `pkg/draftrag/cached_embedder.go` с `CachedEmbedder`, `CacheOptions`, `EmbedCacheStats`.
- 5 unit-тестов: hit/miss, stats, nil base error, error propagation, interface check.
- Документация в `docs/embedders.md` с примером Stats и композиции с RetryEmbedder.

## Продолжение

- Опция TTL для записей кэша.
- Redis second-level cache (уже специфицировано в `embedding-cache` спеке, не реализовано).
