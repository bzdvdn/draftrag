# Fluent Search API — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/search.go | T1.1, T2.1, T2.2, T2.3 |
| pkg/draftrag/draftrag.go | T1.2, T1.3 |
| pkg/draftrag/eval/harness.go | T1.4 |
| pkg/draftrag/search_builder_test.go | T3.1 |

## Фаза 1: Основа

- [x] T1.1 Создать `pkg/draftrag/search.go` с типом `SearchBuilder` и entry point `Pipeline.Search()`. Touches: pkg/draftrag/search.go
- [x] T1.2 Добавить `Pipeline.Retrieve(ctx, q, topK)` как реализацию `eval.RetrievalRunner`. Touches: pkg/draftrag/draftrag.go
- [x] T1.3 Удалить 15+ verbose методов (QueryTopK, AnswerTopK, QueryTopKWithParentIDs и т.д.) из Pipeline. Touches: pkg/draftrag/draftrag.go
- [x] T1.4 Обновить `eval.RetrievalRunner` интерфейс — заменить `QueryTopK` на `Retrieve`. Touches: pkg/draftrag/eval/harness.go

## Фаза 2: Основная реализация

- [x] T2.1 Реализовать builder-методы: `TopK`, `Filter`, `ParentIDs`, `Hybrid`. Touches: pkg/draftrag/search.go
- [x] T2.2 Реализовать terminal-методы `Retrieve` и `Answer` с routing по флагам (HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic). Touches: pkg/draftrag/search.go
- [x] T2.3 Реализовать terminal-методы `Cite`, `InlineCite`, `Stream`, `StreamCite`. Touches: pkg/draftrag/search.go

## Фаза 3: Проверка

- [x] T3.1 Написать `pkg/draftrag/search_builder_test.go`: validation (EmptyQuery, InvalidTopK, NilContext), Retrieve, ParentIDs, Filter, Answer, Cite, Stream cancellation, HyDE, MultiQuery. Touches: pkg/draftrag/search_builder_test.go
- [x] T3.2 Убедиться что `go test ./...` проходит без ошибок.

## Покрытие критериев приемки

- AC-001 → T2.2, T3.1
- AC-002 → T2.2, T3.1
- AC-003 → T2.1, T2.2, T3.1
- AC-004 → T2.3, T3.1
- AC-005 → T2.3, T3.1
