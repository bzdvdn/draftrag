---
slug: search-builder-stream-sources
status: completed
archived_at: 2026-04-10
---

# Archive Summary: search-builder-stream-sources

## Status

completed

## Reason

Gap в стриминговом покрытии SearchBuilder закрыт: добавлен метод `StreamSources` — потоковый аналог `Cite`, возвращающий `(<-chan string, RetrievalResult, error)`.

## Completed Scope

- `internal/application/pipeline.go` — 6 новых методов `Answer*StreamWithSources` (basic, HyDE, MultiQuery, Hybrid, ParentIDs, Filter); каждый = `Query* + streamFromResult + return (chan, result, err)`
- `pkg/draftrag/search.go` — метод `StreamSources` с полным routing switch (6 веток), аналогичным `Stream`
- `pkg/draftrag/search_builder_test.go` — `TestSearchBuilder_StreamSources_StreamingNotSupported`

## Acceptance

- AC-001: `StreamSources` возвращает канал и RetrievalResult; `go build ./...` ok
- AC-002: все 6 routing-веток покрыты в pipeline.go и search.go
- AC-003: `errors.Is(err, ErrStreamingNotSupported)` — PASS; ch == nil, sources пуст

## Notable Deviations

- `toPublicResult` helper не существовал — конвертация не нужна, т.к. `RetrievalResult = domain.RetrievalResult` (type alias).
