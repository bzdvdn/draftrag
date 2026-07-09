# Answer: inline citations в тексте ответа (v1) — План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи.  
Outputs: plan и tasks; data model/contract изменения только если они реально нужны.  
Stop if: нельзя однозначно определить публичный API и формат результата.

## Цель

Добавить новый режим Answer*, который:
- просит LLM вставлять inline-цитаты `[n]` в текст ответа,
- возвращает детерминированный маппинг `n → retrieved chunk`,
- не меняет поведение существующих методов.

## Scope

- Добавить новые публичные методы `Answer*WithInlineCitations` в `pkg/draftrag`.
- Реализовать генерацию prompt с нумерованными источниками в `internal/application`.
- Добавить тип для маппинга citations и тесты без внешней сети.
- Граница: существующий контракт `Answer`/`AnswerWithCitations` и их prompt остаются неизменными.

## Implementation Surfaces

- `internal/application/pipeline.go`: новый метод AnswerWithInlineCitations, отдельный builder userMessage для режима inline citations.
- `internal/domain/models.go`: новый доменный тип `InlineCitation` (аддитивно).
- `pkg/draftrag/draftrag.go`: новые публичные методы и re-export типа `InlineCitation`, валидация входных данных как в остальных Answer*.
- `internal/application/*_test.go`, `pkg/draftrag/*_test.go`: unit-тесты нового режима и входной валидации.

## Влияние на архитектуру

- Аддитивное расширение application и публичного API.
- Domain расширяется новым value-type, без зависимостей на внешние пакеты.
- Compatibility: существующие методы не меняются.

## Acceptance Approach

- AC-001: prompt нумерует источники, метод возвращает `citations` с номерами и соответствующими `RetrievedChunk`, длина соответствует источникам, реально попавшим в prompt.
- AC-002: новые методы/типы добавляются аддитивно; существующие тесты и методы остаются без изменений.

## Данные и контракты

- Data model: только новый тип `InlineCitation` (внутренний/публичный re-export).
- Public API: добавляются новые методы Answer*WithInlineCitations; существующие API без изменений.

## Стратегия реализации

- DEC-001 Нумерация источников на стороне backend
  Why: детерминированный маппинг `n → chunk` не зависит от структуры ответа LLM.
  Tradeoff: не гарантирует “идеальную” привязку фактов к источникам; это out-of-scope.
  Affects: `internal/application/pipeline.go`, `internal/domain/models.go`, `pkg/draftrag/draftrag.go`.
  Validation: unit-тест проверяет наличие `[1]`, `[2]` в prompt и корректность возвращаемого массива citations.

- DEC-002 Лимит источников через topK и MaxContextChunks
  Why: простая управляемая граница объёма контекста и числа доступных `[n]`.
  Tradeoff: если `MaxContextChars` режет контекст, часть чанка может быть усечена.
  Affects: builder userMessage режима inline citations.
  Validation: тест с `MaxContextChunks=1` возвращает 1 citation.

## Incremental Delivery

### MVP (Первая ценность)

- Новый метод в application + публичные методы + базовые тесты (AC-001, AC-002).

### Итеративное расширение

- (Не в MVP) опциональная мягкая/строгая валидация корректности номеров `[n]` в ответе LLM.

## Риски

- LLM может генерировать номера, которых нет (например `[999]`).
  Mitigation: v1 не ломаем вызов; возвращаем маппинг допустимых номеров, а валидацию добавим опционально позже.

## Проверка

- `go test ./...` локально.
- Тесты должны не использовать сеть (fake LLM / fake store / fake embedder).

## Соответствие конституции

- Нет конфликтов: изменения аддитивны, через интерфейсы и с unit-тестами; domain остаётся без внешних импортов.

