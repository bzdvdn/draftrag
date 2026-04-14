# AnswerWithCitations для draftRAG — План

## Phase Contract

Inputs: `.speckeep/specs/answer-with-citations/spec.md`, `.speckeep/specs/answer-with-citations/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно добавить методы аддитивно без нарушения существующего API.

## Цель

Добавить методы `AnswerWithCitations`/`AnswerTopKWithCitations`, которые возвращают:
- `answer string` (результат LLM),
- `RetrievalResult` (retrieval evidence: чанки + score + query text),
с сохранением текущих `Answer*` методов.

## Scope

- Public API: новые методы на `pkg/draftrag.Pipeline`.
- Application: use-case метод, возвращающий `(answer, retrieval, err)`.
- Testing: unit-тесты на то, что retrieval evidence возвращается и ответ не теряется.

## Implementation Surfaces

- `pkg/draftrag/draftrag.go` — добавить методы:
  - `AnswerWithCitations(ctx, question) (string, RetrievalResult, error)`
  - `AnswerTopKWithCitations(ctx, question, topK) (string, RetrievalResult, error)`
  Валидация как в `AnswerTopK` (T1.1, T2.1).
- `internal/application/pipeline.go` — добавить use-case метод `AnswerWithCitations(ctx, question, topK)` (или аналог) (T2.2).
- `pkg/draftrag/answer_with_citations_test.go` — compile-time тест + проверки публичной валидации ошибок (T3.1).
- `internal/application/answer_with_citations_test.go` — unit-тесты: retrieval evidence возвращается, LLM ответ возвращается, ctx cancel (T3.2).

## Влияние на архитектуру

- Никаких новых внешних зависимостей.
- Clean Architecture сохраняется: orchestration остаётся в application, pkg — тонкий wrapper.
- Публичный API расширяется аддитивно.

## Acceptance Approach

- AC-001 -> compile-time тест, что методы доступны и компилируются: `pkg/draftrag/answer_with_citations_test.go`.
- AC-002 -> unit-тест use-case: `Search` возвращает `RetrievalResult`, и он возвращается наружу без потери: `internal/application/answer_with_citations_test.go`.
- AC-003 -> unit-тест use-case: `Generate` возвращает `"ok"` и метод возвращает `"ok"`: `internal/application/answer_with_citations_test.go`.
- AC-004 -> `go test ./...` проходит (существующие методы не ломаются).

## Данные и контракты

- Возвращаемый `RetrievalResult` совпадает с тем, что вернул `VectorStore.Search`, включая `Chunks` и `QueryText`.
- Partial-result поведение:
  - Если retrieval уже выполнен, а `Generate` вернул ошибку — возвращаем `RetrievalResult` (для диагностики) и пустой/частичный `answer` (скорее пустой) вместе с `err`.

## Стратегия реализации

- DEC-001 Возвращать retrieval evidence даже при ошибке Generate
  Why: упрощает отладку и UI “Sources” (можно показать источники, даже если LLM упал).
  Tradeoff: API возвращает частичный результат при ошибке.
  Affects: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
  Validation: unit-тест, что retrieval возвращён при ошибке Generate.

- DEC-002 Не менять существующие `Answer*` методы
  Why: backward compatibility.
  Tradeoff: больше методов в API.
  Affects: `pkg/draftrag/draftrag.go`
  Validation: `go test ./...`.

## Incremental Delivery

### MVP (Первая ценность)

- Добавить методы и use-case + unit-тесты AC-001..AC-004.

### Итеративное расширение

- (Out of scope) авто-нумерация источников и вставка ссылок в текст ответа.

## Порядок реализации

1. Application: добавить use-case метод и unit-тесты.
2. pkg: добавить публичные методы и unit-тесты.
3. `go test ./...`.

## Риски

- Риск 1: неоднозначность “partial result” контракта.
  Mitigation: зафиксировать DEC-001 и отразить в тестах.

## Rollout и compatibility

- Rollout не требуется.
- Compatibility: аддитивное расширение.

## Проверка

- `go test ./...`
- unit-тесты по AC-001..AC-003.

## Соответствие конституции

- нет конфликтов: зависимости — интерфейсы domain, ctx safety сохраняется, тестируемость обеспечена.

