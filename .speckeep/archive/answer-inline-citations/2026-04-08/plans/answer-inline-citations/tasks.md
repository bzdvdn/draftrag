# Answer: inline citations в тексте ответа (v1) — Задачи

## Phase Contract

Inputs: plan и supporting артефакты для фичи.  
Outputs: исполнимые задачи с покрытием AC.  
Stop if: задачи нельзя проверить unit-тестами без сети.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/models.go | T1.1 |
| internal/application/pipeline.go | T2.1 |
| internal/application/answer_inline_citations_test.go | T3.1 |
| pkg/draftrag/draftrag.go | T2.2 |
| pkg/draftrag/answer_inline_citations_test.go | T3.2 |

## Фаза 1: Основа

- [x] T1.1 Добавить доменный тип `InlineCitation` (аддитивно) для маппинга `n → RetrievedChunk`. Touches: `internal/domain/models.go`.

## Фаза 2: Основная реализация

- [x] T2.1 Реализовать `Pipeline.AnswerWithInlineCitations` в `internal/application` и prompt builder с нумерацией источников `[n]`. Touches: `internal/application/pipeline.go`. (AC-001)
- [x] T2.2 Добавить публичные методы `AnswerWithInlineCitations`/`AnswerTopKWithInlineCitations` и re-export типа `InlineCitation` в `pkg/draftrag`. Touches: `pkg/draftrag/draftrag.go`. (AC-002)

## Фаза 3: Проверка

- [x] T3.1 Добавить unit-тесты для application: prompt содержит `[1]`, возвращается корректный `citations` и соблюдается лимит `MaxContextChunks`. Touches: `internal/application/answer_inline_citations_test.go`. (AC-001)
- [x] T3.2 Добавить unit-тесты для публичного API: входная валидация (empty question, invalid topK). Touches: `pkg/draftrag/answer_inline_citations_test.go`. (AC-002)
- [x] T3.3 Прогнать `go test ./...` и устранить проблемы форматирования (`gofmt`). Touches: repo.

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1, T3.1
- AC-002 -> T2.2, T3.2, T3.3
