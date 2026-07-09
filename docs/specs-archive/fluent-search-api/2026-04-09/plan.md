# Fluent Search API — План

## Цель

Ввести `SearchBuilder` как единую точку входа для всех поисковых операций, удалив 15+ verbose методов. Изменение затрагивает только `pkg/draftrag` (публичный API) — внутренняя логика остаётся нетронутой.

## Scope

- Новый файл `pkg/draftrag/search.go` с типом `SearchBuilder`
- Изменения в `pkg/draftrag/draftrag.go`: удаление старых методов, добавление `Search()`, `Retrieve()`, `DeleteDocument()`, `UpdateDocument()`
- Изменения в `pkg/draftrag/eval/harness.go`: обновление интерфейса `RetrievalRunner`
- Нетронутым остаётся `internal/application/pipeline.go`

## Implementation Surfaces

- `pkg/draftrag/search.go` — новая поверхность, строит и выполняет запрос
- `pkg/draftrag/draftrag.go` — существующая поверхность, упрощается
- `pkg/draftrag/eval/harness.go` — существующая поверхность, обновляется интерфейс

## Влияние на архитектуру

- Breaking change публичного API: все методы `QueryTopK*`, `AnswerTopK*` и т.д. удалены.
- `eval.RetrievalRunner` меняет сигнатуру метода с `QueryTopK` на `Retrieve`.
- Нет изменений в `internal/` — только `pkg/` публичная поверхность.

## Acceptance Approach

- AC-001 → `SearchBuilder.Retrieve` делегирует в `p.core.Query`
- AC-002 → валидация в начале каждого terminal-метода
- AC-003 → `ParentIDs(...)` сохраняет ids в builder, при Retrieve проксирует в `QueryWithParentIDs`
- AC-004 → terminal `Answer` делегирует в `p.core.Answer*`; `Cite` вызывает `AnswerWithCitations`
- AC-005 → Stream проверяет `ctx.Err()` перед делегированием

## Стратегия реализации

- DEC-001 Routing в terminal методах, не в builder-методах
  Why: builder-методы только накапливают параметры; routing по флагам в одном месте упрощает понимание
  Tradeoff: terminal-методы содержат if-цепочку; но она компактна
  Affects: `search.go` terminal methods
  Validation: все AC-тесты проходят; нет дублирования логики

- DEC-002 Приоритет стратегий: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic
  Why: HyDE меняет embedding запроса — самое раннее решение; остальные — фильтры поверх
  Tradeoff: нельзя одновременно HyDE+Hybrid (HyDE побеждает)
  Affects: `Retrieve` и `Answer` routing
  Validation: `TestSearchBuilder_HyDE` pass

## Порядок реализации

1. Добавить `SearchBuilder` struct и builder-методы
2. Реализовать terminal `Retrieve` с routing
3. Реализовать terminal `Answer`, `Cite`, `InlineCite`
4. Реализовать terminal `Stream`, `StreamCite`
5. Удалить старые методы из `draftrag.go`
6. Обновить `eval/harness.go`
7. Написать тесты

## Риски

- Риск: удаление публичных методов — breaking change для пользователей
  Mitigation: задокументировано в changelog; новый API покрывает все старые случаи

## Rollout и compatibility

- Breaking change; нет deprecation wrapper'ов.
- Rollout одним PR.

## Проверка

- `go test ./pkg/draftrag/...` — все тесты pass
- `go test ./pkg/draftrag/eval/...` — harness тесты pass
- `go build ./...` — нет compilation errors
