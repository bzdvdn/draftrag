# Pipeline.Answer для draftRAG — Задачи

## Phase Contract

Inputs: `.speckeep/plans/pipeline-answer/plan.md`, `.speckeep/plans/pipeline-answer/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/draftrag.go | T1.1, T2.1 |
| internal/application/pipeline.go | T2.2 |
| pkg/draftrag/pipeline_answer_test.go | T3.1 |
| internal/application/pipeline_answer_test.go | T3.2 |
| domain.LLMProvider | T2.2, T3.2 |
| domain.VectorStore | T2.2, T3.2 |
| domain.Embedder | T2.2, T3.2 |
| httptest.Server | none |

## Фаза 1: Публичный API каркас

Цель: добавить публичные методы `Answer*` в `pkg/draftrag` и зафиксировать контракт валидации.

- [x] T1.1 Обновить `pkg/draftrag/draftrag.go` — добавить методы `(*Pipeline) Answer(ctx, question)` и `(*Pipeline) AnswerTopK(ctx, question, topK)` с русским godoc; `Answer` делегирует в `AnswerTopK` с defaultTop=5. Touches: pkg/draftrag/draftrag.go

## Фаза 2: Application use-case и prompt contract

Цель: реализовать RAG-цикл в application слое и обеспечить детерминированный Prompt Contract v1.

- [x] T2.1 Реализовать публичные методы `Answer*` в `pkg/draftrag`: `ctx != nil` (panic), ранний `ctx.Err()`, `question` trim/валидация (пустой -> `ErrEmptyQuery`), `topK` валидация (`<=0` -> `ErrInvalidTopK`), делегирование в application use-case. Touches: pkg/draftrag/draftrag.go
- [x] T2.2 Обновить `internal/application/pipeline.go` — добавить метод `Answer(ctx, question, topK)` (или эквивалент), который выполняет: `Embed(question)` → `Search(embedding, topK)` → формирование prompt (system+user по контракту v1) → `LLMProvider.Generate` → возврат ответа. Touches: internal/application/pipeline.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC через unit-тесты с заглушками зависимостей.

- [x] T3.1 Создать `pkg/draftrag/pipeline_answer_test.go` — тесты публичного API: AC-001 (compile-time/наличие методов), AC-004 (валидация question/topK), AC-005 (ctx cancel/deadline ≤ 100мс в тестовом сценарии). Touches: pkg/draftrag/pipeline_answer_test.go
- [x] T3.2 Создать `internal/application/pipeline_answer_test.go` — unit-тесты use-case: AC-002 (порядок вызовов Embed→Search→Generate и возврат результата), AC-003 (Prompt Contract v1: system prompt + user message формат “Контекст/Вопрос”), AC-005 (ctx cancel/deadline). Touches: internal/application/pipeline_answer_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1
- AC-002 -> T2.2, T3.2
- AC-003 -> T2.2, T3.2
- AC-004 -> T2.1, T3.1
- AC-005 -> T2.1, T2.2, T3.1, T3.2

## Заметки

- Все тесты должны проходить без внешней сети.
- В v1 prompt contract фиксированный; кастомизация — out of scope.
