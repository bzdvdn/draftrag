# fix-inline: план

## Phase Contract

Inputs: spec.md, inspect.md (pass), `pkg/draftrag/search.go` (строки 269-271).
Outputs: plan.md, data-model.md.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Добавить в ветку `filter.Fields` метода `SearchBuilder.InlineCite` (`pkg/draftrag/search.go`) проверку `errors.Is(err, application.ErrFiltersNotSupported)` и маппинг во внешнюю `ErrFiltersNotSupported` — по аналогии с уже существующим паттерном в `Cite` и `StreamCite`. Добавить unit-тест, воспроизводящий баг.

## Scope

- `pkg/draftrag/search.go` — одна ветка в методе `InlineCite` (строки 269-271)
- `pkg/draftrag/search_builder_test.go` — новый тест на маппинг ошибки
- `internal/application`, `internal/domain` — не затрагиваются

## Implementation Surfaces

- **`pkg/draftrag/search.go` (существующая)** — единственная поверхность изменения; метод `InlineCite`, ветка `len(b.filter.Fields) > 0`. Сейчас возвращает ошибку напрямую без unwrap/remap.
- **`pkg/draftrag/search_builder_test.go` (существующая)** — добавляется один тест-функция; моки (`fixedEmbedder`, `mockLLM`, `setupPipeline`) уже есть в файле.

## Влияние на архитектуру

- Локальное: изменение ограничено одним `if`-блоком в одном методе публичного API.
- Нет изменений в domain, application или infrastructure слоях.
- Нет breaking changes публичного API: сигнатура `InlineCite` не меняется, тип возвращаемой ошибки остаётся `error`.
- Rollout: без специальных шагов — исправление поведения в пределах публичного пакета.

## Acceptance Approach

- **AC-001** → Добавить `errors.Is(err, application.ErrFiltersNotSupported)` check в ветку `filter.Fields` `InlineCite`. Написать тест с хранилищем без поддержки фильтров (не реализующим `VectorStoreWithFilters`) и `MetadataFilter`, подтвердить `errors.Is(err, ErrFiltersNotSupported) == true`.
- **AC-002** → Существующие тесты `InlineCite` на совместимом store не должны сломаться. Подтверждается прогоном `go test ./pkg/draftrag/...`.
- **AC-003** → В тесте использовать mock-store, возвращающий `fmt.Errorf("wrap: %w", application.ErrFiltersNotSupported)`, проверить что `errors.Is` всё равно срабатывает. Подтверждает корректность `errors.Is` для wrapped errors.

## Данные и контракты

Эта фича не вводит новых сущностей, не меняет API- или event-boundaries.
`data-model.md` содержит только placeholder (см. файл).
Контракты не меняются: публичный тип `ErrFiltersNotSupported` уже существует в `pkg/draftrag/errors.go`.

## Стратегия реализации

- **DEC-001** Маппинг через `errors.Is` с ранним возвратом
  Why: Все остальные ветки в `SearchBuilder` используют именно этот паттерн (`errors.Is(err, application.ErrFiltersNotSupported)`); отклонение создало бы непоследовательный API.
  Tradeoff: Нет — это точечный паттерн-матч, нет overhead'а.
  Affects: `pkg/draftrag/search.go:269-271`
  Validation: `errors.Is(err, ErrFiltersNotSupported) == true` в unit-тесте (AC-001, AC-003).

## Incremental Delivery

### MVP (Первая ценность)

- Изменить ветку `filter.Fields` в `InlineCite` (2 строки кода).
- Добавить тест, воспроизводящий баг (AC-001, AC-003).
- Критерий: `go test ./pkg/draftrag/... -run TestSearchBuilder_InlineCite_FilterNotSupported` — зелёный.

### Итеративное расширение

Нет — фича атомарна, MVP = полная реализация.

## Порядок реализации

1. Изменить `search.go` (ветка `filter.Fields` в `InlineCite`) — обязательно первым.
2. Добавить unit-тест в `search_builder_test.go` — можно в тот же коммит.
3. Прогнать `go test ./pkg/draftrag/...` для подтверждения AC-001, AC-002, AC-003.

Параллелить нечего; всё умещается в одну атомарную задачу.

## Риски

- **Риск:** mock-store для AC-003 может случайно реализовать `VectorStoreWithFilters`, сделав тест бесполезным.
  Mitigation: Использовать тип, явно не реализующий этот интерфейс (например, `vectorstore.InMemoryStore` проверить на compile-time или использовать минимальный inline-mock без `SearchWithMetadataFilter`).

## Rollout и compatibility

Специальных rollout-действий не требуется. Изменение — исправление поведения публичного API; не вводит новые зависимости, не ломает существующий контракт.

## Проверка

- `go test ./pkg/draftrag/... -run TestSearchBuilder` — покрывает AC-001, AC-002, AC-003.
- `go test ./pkg/draftrag/...` — регрессия по всему пакету (AC-002).
- `go vet ./pkg/draftrag/...` — конституционное требование.

## Соответствие конституции

- **Чистая архитектура**: изменение только в `pkg/draftrag` (API layer), domain и application не затрагиваются ✓
- **Интерфейсная абстракция**: `VectorStoreWithFilters` используется только через `errors.Is` на уровне error propagation, интерфейсы не меняются ✓
- **Тестируемость**: добавляется unit-тест с mock ✓
- **Godoc-комментарии на русском**: метод `InlineCite` уже имеет комментарий; изменение внутри тела метода комментария не требует ✓
- **`go vet`, `go fmt`**: изменение тривиально, конфликтов нет ✓
