# CachedEmbedder Public API — План

## Цель

Создать тонкий публичный фасад над `internal/infrastructure/embedder/cache.EmbedderCache`. Один новый файл, нет изменений во внутренней логике.

## Scope

- `pkg/draftrag/cached_embedder.go` — новый файл

## Стратегия реализации

- DEC-001 Делегирование через приватное поле
  Why: `type CachedEmbedder struct { impl *cache.EmbedderCache }` не раскрывает внутренний тип в публичном API
  Tradeoff: небольшой boilerplate; но чистый публичный контракт
  Affects: pkg/draftrag/cached_embedder.go
  Validation: пользователь видит только `CachedEmbedder`, `CacheOptions`, `EmbedCacheStats`

- DEC-002 `EmbedCacheStats = cache.CacheStats` — type alias
  Why: сохраняет identity; stats struct не нужно дублировать
  Tradeoff: пользователь видит `cache.CacheStats` в godoc — незначительно
  Affects: pkg/draftrag/cached_embedder.go

## Порядок реализации

1. Создать `cached_embedder.go`
2. Написать тесты
3. Задокументировать

## Rollout и compatibility

- Additive; нет breaking changes.
