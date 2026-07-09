# Retrieval Strategies — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/interfaces.go | T1.1 |
| internal/application/pipeline.go | T1.2, T2.1, T2.2, T2.3 |
| pkg/draftrag/draftrag.go | T1.3 |
| pkg/draftrag/search.go | T1.4 |
| pkg/draftrag/reranker_test.go | T3.1 |
| pkg/draftrag/search_builder_test.go | T3.2 |

## Фаза 1: Основа

- [x] T1.1 Добавить `domain.Reranker` интерфейс в `internal/domain/interfaces.go`. Touches: internal/domain/interfaces.go
- [x] T1.2 Добавить `reranker` поле в `Pipeline` struct и `PipelineConfig`; реализовать `maybeRerank` helper. Touches: internal/application/pipeline.go
- [x] T1.3 Экспортировать `type Reranker = domain.Reranker` и добавить `Reranker` в `PipelineOptions`. Touches: pkg/draftrag/draftrag.go
- [x] T1.4 Добавить builder-методы `HyDE()` и `MultiQuery(n)` в `SearchBuilder`. Touches: pkg/draftrag/search.go

## Фаза 2: Основная реализация

- [x] T2.1 Реализовать `QueryHyDE` / `AnswerHyDE` с `hydeSystemPrompt` константой. Touches: internal/application/pipeline.go
- [x] T2.2 Реализовать `rrfMergeMultiple` (RRF k=60 по N спискам). Touches: internal/application/pipeline.go
- [x] T2.3 Реализовать `QueryMulti` / `AnswerMulti` с `parseMultiQueryLines`. Touches: internal/application/pipeline.go

## Фаза 3: Проверка

- [x] T3.1 Написать `reranker_test.go`: `reverseReranker` mock, IsCalled, NoReranker_Works. Touches: pkg/draftrag/reranker_test.go
- [x] T3.2 Дополнить `search_builder_test.go` тестами HyDE, HyDE_Answer, MultiQuery, MultiQuery_Answer. Touches: pkg/draftrag/search_builder_test.go
- [x] T3.3 Убедиться что `go test ./...` проходит без ошибок.

## Покрытие критериев приемки

- AC-001 → T2.1, T1.4, T3.2
- AC-002 → T2.2, T2.3, T1.4, T3.2
- AC-003 → T1.2, T3.1
- AC-004 → T1.2, T3.1
