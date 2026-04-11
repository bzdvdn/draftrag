# search-builder-stream-sources: план

## Phase Contract

Inputs: spec.md, inspect.md (pass), `pkg/draftrag/search.go`, `internal/application/pipeline.go` (строки 1652–1740).
Outputs: plan.md, data-model.md.

## Цель

Добавить метод `StreamSources` в `SearchBuilder` (`pkg/draftrag/search.go`) и 6 тонких wrapper-методов `Answer*StreamWithSources` в `internal/application/pipeline.go`. Метод возвращает `(<-chan string, RetrievalResult, error)` — потоковый ответ с синхронно готовым списком источников, без inline-разметки.

## Scope

- `internal/application/pipeline.go` — добавить 6 методов `Answer*StreamWithSources`, реализованных по паттерну `Query* + streamFromResult + return result`
- `pkg/draftrag/search.go` — добавить метод `StreamSources`, routing-структура идентична `Stream`
- `pkg/draftrag/search_builder_test.go` — тест на `ErrStreamingNotSupported` (аналог существующих тестов для `Stream`/`StreamCite`)

## Implementation Surfaces

- **`internal/application/pipeline.go` (существующая)** — добавить 6 методов после строки 1694 (после `AnswerStreamWithMetadataFilter`) и до строки 1696 (начало `AnswerHyDEStreamWithInlineCitations`). Каждый метод — 4–6 строк: вызов `Query*`, `streamFromResult`, возврат `(tokenChan, result, err)`.
  - `AnswerStreamWithSources(ctx, question, topK) (<-chan string, domain.RetrievalResult, error)` — базовый маршрут
  - `AnswerHyDEStreamWithSources(ctx, question, topK)` — HyDE
  - `AnswerMultiStreamWithSources(ctx, question, n, topK)` — MultiQuery
  - `AnswerHybridStreamWithSources(ctx, question, topK, cfg)` — Hybrid
  - `AnswerStreamWithParentIDsWithSources(ctx, question, topK, parentIDs)` — ParentIDs
  - `AnswerStreamWithMetadataFilterWithSources(ctx, question, topK, filter)` — Filter

- **`pkg/draftrag/search.go` (существующая)** — добавить `StreamSources` после `Stream` (строка 344). Структура routing switch идентична `Stream`; вместо `return tokenChan, err` возвращает `return tokenChan, sources, err`, где `sources = toPublicResult(result)` (тот же хелпер, что используют `Cite` и `InlineCite`).

- **`pkg/draftrag/search_builder_test.go` (существующая)** — добавить `TestSearchBuilder_StreamSources_StreamingNotSupported` по образцу существующих тестов на `ErrStreamingNotSupported`.

## Acceptance Approach

- **AC-001** (канал + RetrievalResult, error == nil) → `AnswerStreamWithSources` возвращает `(chan, result, nil)`; `StreamSources` передаёт `result` как второй аргумент; тест читает канал до закрытия и проверяет `len(result.Chunks) > 0`.
- **AC-002** (все 6 routing-веток) → 6 application-методов × 6 веток в switch `StreamSources`; `go build ./...` подтверждает покрытие компилятором.
- **AC-003** (ErrStreamingNotSupported) → каждый `Answer*StreamWithSources` проверяет `domain.StreamingLLMProvider` assertion (или делегирует в `streamFromResult`, который уже проверяет); тест с `noStreamLLM` mock.

## Данные и контракты

Фича не вводит новых сущностей, не затрагивает API или event boundaries. `data-model.md` — placeholder.

## Стратегия реализации

### DEC-001 Добавить 6 application-методов `Answer*StreamWithSources`, а не переиспользовать `WithInlineCitations`

Why: `streamFromResult` и `streamInlineFromResult` строят разные prompts — `streamInlineFromResult` передаёт LLM инструкцию добавлять citation-маркеры (`[1]`, `[2]`). Если переиспользовать `WithInlineCitations`, эти маркеры попадут в `<-chan string` и будут видны пользователю — семантически неверно.

Tradeoff: 6 новых методов вместо 0, но каждый — 4–6 строк; копипаст паттерна `Query* + streamFromResult` уже используется в существующих `Answer*Stream` методах.

Affects: `internal/application/pipeline.go`

Validation: `go build ./...` + тест `TestSearchBuilder_StreamSources_StreamingNotSupported` не должен видеть маркеров в тексте.

### DEC-002 `StreamSources` в `pkg/draftrag` следует routing-структуре `Stream` дословно

Why: routing-логика в `Stream` уже покрывает все 6 веток с корректной обработкой `ErrHybridNotSupported` и `ErrFiltersNotSupported`; дублирование этой структуры — единственный способ не пропустить ветку.

Tradeoff: ~35 строк копипаста — но это тот же трейдофф, что уже сделан при добавлении `StreamCite`.

Affects: `pkg/draftrag/search.go`

Validation: code review switch содержит все 6 веток; `go build ./...` ok.

## Влияние на архитектуру

- Группа A (`internal/application/pipeline.go`): 6 новых тонких wrapper-методов; не меняют публичный интерфейс `Pipeline`; не затрагивают `domain` или `infrastructure`.
- Группа B (`pkg/draftrag/search.go`): 1 новый метод на `*SearchBuilder`; публичный API расширяется, без breaking changes.
- Нет migration, нет feature flags, нет breaking changes.

## Порядок реализации

1. Добавить 6 `Answer*StreamWithSources` в `pipeline.go` — `go build ./...` ok.
2. Добавить `StreamSources` в `search.go` — `go build ./...` ok.
3. Добавить тест в `search_builder_test.go` — `go test ./pkg/draftrag/... -run TestSearchBuilder_StreamSources` ok.

Шаги 1 и 2 зависят последовательно (2 использует методы из 1). Шаг 3 независим от 2, но логичнее после.

## Риски

- **Риск:** пропущенная ветка в routing switch `StreamSources`.
  Mitigation: копировать switch дословно из `Stream`; после добавления — `grep -c "StreamWithSources\|HyDEStreamWithSources\|MultiStreamWithSources\|HybridStreamWithSources\|ParentIDsWithSources\|MetadataFilterWithSources" pipeline.go` должен дать 6.

- **Риск:** `streamFromResult` не проверяет `StreamingLLMProvider` — проверка может быть только внутри `AnswerStream`.
  Mitigation: перед реализацией убедиться, что `streamFromResult` возвращает `ErrStreamingNotSupported` (или вызов `streamFromResult` из `AnswerStream` делает assertion до вызова). По коду `AnswerStreamWithInlineCitations` видно, что assertion выполняется до `streamInlineFromResult` — применить тот же паттерн в `AnswerStreamWithSources`.

## Rollout и compatibility

Специальных rollout-действий не требуется. `StreamSources` — новый метод; никакой существующий код не ломается.

## Проверка

- `go build ./...` → ok (AC-002).
- `go test ./pkg/draftrag/... -run TestSearchBuilder_StreamSources` → ok (AC-003).
- Ручная проверка: routing switch содержит 6 веток (AC-002).
- Интеграционный тест (если доступен mock streaming LLM): канал возвращает токены, `RetrievalResult.Chunks` непуст (AC-001).

## Соответствие конституции

- **Чистая архитектура**: `pkg/draftrag` (public API) вызывает `internal/application` — направление зависимости внутрь ✓; `internal/application` не ссылается на `pkg` ✓; `internal/domain` не затрагивается ✓
- **Интерфейсная абстракция**: `StreamingLLMProvider` assertion сохраняется ✓; новые методы не вводят конкретных LLM-зависимостей ✓
- **Контекстная безопасность**: `ctx context.Context` — первый параметр во всех новых методах ✓
- **Тестируемость**: тест на `ErrStreamingNotSupported` с mock ✓
- **Godoc**: все новые публичные методы получат godoc на русском ✓
- **`go build`, `go vet`, `go fmt`**: тонкие wrappers и routing copy — vet-рисков нет ✓
