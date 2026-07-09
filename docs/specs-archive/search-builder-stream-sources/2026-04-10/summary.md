---
slug: search-builder-stream-sources
generated_at: 2026-04-10
---

## Goal

Добавить метод `StreamSources` в `SearchBuilder` — потоковый аналог `Cite`, возвращающий токен-канал и список источников без inline-разметки.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | StreamSources возвращает канал и RetrievalResult | канал содержит токены, `len(Chunks) > 0` |
| AC-002 | Покрыты все 6 routing-веток | `go build ./...` ok; switch содержит все ветки |
| AC-003 | ErrStreamingNotSupported маппируется корректно | `errors.Is(err, ErrStreamingNotSupported)` в тесте |

## Out of Scope

- Стриминговый InlineCite (уже покрыт StreamCite)
- Стриминговый Retrieve (retrieval стриминга не требует)
- Новые application-методы (решается в plan)
- Изменение Stream, StreamCite, Cite
