# AnswerWithCitations для draftRAG — Задачи

## Phase Contract

Inputs: `.draftspec/plans/answer-with-citations/plan.md`, `.draftspec/plans/answer-with-citations/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/draftrag.go | T1.1, T2.1 |
| internal/application/pipeline.go | T2.2 |
| pkg/draftrag/answer_with_citations_test.go | T3.1 |
| internal/application/answer_with_citations_test.go | T3.2 |
| domain.VectorStore | T2.2, T3.2 |
| domain.Embedder | T2.2, T3.2 |
| domain.LLMProvider | T2.2, T3.2 |

## Фаза 1: Публичные методы Answer*WithCitations

Цель: добавить аддитивные методы в публичный Pipeline.

- [x] T1.1 Обновить `pkg/draftrag/draftrag.go` — добавить методы `AnswerWithCitations(ctx, question)` и `AnswerTopKWithCitations(ctx, question, topK)` с русским godoc; `AnswerWithCitations` использует defaultTopK. Touches: pkg/draftrag/draftrag.go

## Фаза 2: Application use-case

Цель: реализовать use-case, который возвращает answer + retrieval evidence.

- [x] T2.1 Реализовать публичные методы `Answer*WithCitations` в `pkg/draftrag`: `ctx != nil` (panic), ранний `ctx.Err()`, trim/валидация `question` (пустой -> `ErrEmptyQuery`), `topK` валидация (`<=0` -> `ErrInvalidTopK`), делегирование в application use-case. Touches: pkg/draftrag/draftrag.go
- [x] T2.2 Обновить `internal/application/pipeline.go` — добавить метод `AnswerWithCitations(ctx, question, topK)` (или эквивалент), который выполняет retrieval (Embed+Search), строит prompt и вызывает `Generate`, возвращая `(answer string, retrieval domain.RetrievalResult, err error)`; при ошибке Generate retrieval возвращается (partial result). Touches: internal/application/pipeline.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC.

- [x] T3.1 Создать `pkg/draftrag/answer_with_citations_test.go` — compile-time тест доступности методов (AC-001) + тесты валидации (ErrEmptyQuery/ErrInvalidTopK). Touches: pkg/draftrag/answer_with_citations_test.go
- [x] T3.2 Создать `internal/application/answer_with_citations_test.go` — unit-тесты: AC-002 (retrieval evidence возвращается как из Search), AC-003 (answer string возвращается как из Generate), partial result при ошибке Generate. Touches: internal/application/answer_with_citations_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1
- AC-002 -> T2.2, T3.2
- AC-003 -> T2.2, T3.2
- AC-004 -> T1.1, T2.1, T2.2, T3.1, T3.2

## Заметки

- Все тесты должны проходить без внешней сети.
