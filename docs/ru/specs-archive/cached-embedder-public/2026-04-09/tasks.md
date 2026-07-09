# CachedEmbedder Public API — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/cached_embedder.go | T1.1 |
| pkg/draftrag/cached_embedder_test.go | T3.1 |
| docs/embedders.md | T3.2 |

## Фаза 1: Основа

- [x] T1.1 Создать `pkg/draftrag/cached_embedder.go` с `CacheOptions`, `CachedEmbedder`, `EmbedCacheStats`; делегировать в `cache.EmbedderCache`. Touches: pkg/draftrag/cached_embedder.go

## Фаза 3: Проверка

- [x] T3.1 Написать `cached_embedder_test.go`: CachesResults, Stats, ImplementsEmbedder, NilBaseError, PropagatesError. Touches: pkg/draftrag/cached_embedder_test.go
- [x] T3.2 Задокументировать в `docs/embedders.md`: секция CachedEmbedder со Stats и композицией с RetryEmbedder. Touches: docs/embedders.md
- [x] T3.3 Убедиться что `go test ./pkg/draftrag/...` проходит.

## Покрытие критериев приемки

- AC-001 → T1.1, T3.1
- AC-002 → T1.1, T3.1
- AC-003 → T1.1, T3.1
- AC-004 → T1.1, T3.1
