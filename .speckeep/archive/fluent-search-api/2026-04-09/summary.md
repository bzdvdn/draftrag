# Сводка архива

## Спецификация

- snapshot: заменены 15+ verbose Pipeline методов единым fluent builder API `Search(q).TopK(n).<terminal>(ctx)`
- slug: fluent-search-api
- archived_at: 2026-04-09
- status: completed

## Причина

Комбинаторный взрыв методов (QueryTopK, AnswerTopKWithParentIDs, QueryWithMetadataFilter и т.д.) делал API нечитаемым и требовал нового метода на каждую комбинацию опций. Fluent builder решает это composable цепочками.

## Результат

- Добавлен `SearchBuilder` в `pkg/draftrag/search.go` с методами `TopK`, `Filter`, `ParentIDs`, `Hybrid`, `HyDE`, `MultiQuery` и терминалами `Retrieve`, `Answer`, `Cite`, `InlineCite`, `Stream`, `StreamCite`.
- Удалены 15+ deprecated методов из `Pipeline`.
- Обновлён `eval.RetrievalRunner` интерфейс.
- Написаны 13 unit-тестов в `search_builder_test.go`.

## Продолжение

- Возможное расширение: `Explain(ctx)` — возврат reasoning trace от LLM; `Rerank` builder-метод для per-query reranker переопределения.
