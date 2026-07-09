# arch-generics: План

## Цель

Рефакторинг публичного API пакета `pkg/draftrag/`: устранить дублирование routing handler'ов через generics, заменить `panic` на error return, обновить trace-маркеры. Все изменения — внутренние, сигнатуры SearchBuilder и Pipeline остаются без изменений.

## MVP Slice

1. **Generic handler factory** — единый builder для handler maps, устраняющий повторяющиеся closures в каждом output-формате (AC-001, AC-004).
2. **Nil context guard** — замена `panic("nil context")` на `return error` во всех публичных методах `pkg/draftrag/draftrag.go` и `pkg/draftrag/search.go` (AC-002).
3. **Trace markers update** — `@sk-task searchbuilder-generics#*` → `@sk-task arch-generics#*` (AC-005).

MVP закрывает AC-001, AC-002, AC-003, AC-004, AC-005.

## First Validation Path

```bash
go build ./...
go test ./...
go vet ./...
grep -r "searchbuilder-generics" pkg/draftrag/  # ожидаем 0 совпадений
grep -r "panic.*nil context" pkg/draftrag/      # ожидаем 0 совпадений
wc -l pkg/draftrag/search_routing.go            # ожидаем ≤110 строк
```

## Scope

1. `pkg/draftrag/search_routing.go` — handler maps: 7 map'ов с 6 routes каждый → единый registry + per-output-type factory.
2. `pkg/draftrag/search_router.go` — существующий `router[T]` не меняется; туда же добавляются helper-типы.
3. `pkg/draftrag/search.go` — замена `panic("nil context")` на `return error` во всех 7 SearchBuilder методах.
4. `pkg/draftrag/draftrag.go` — замена `panic("nil context")` на `return error` в Index, Query, Answer, Retrieve, DeleteDocument, UpdateDocument, IndexBatch.
5. `pkg/draftrag/errors.go` — опционально: добавить sentinel `ErrNilContext`.
6. Trace markers: `@sk-task searchbuilder-generics#*` → `@sk-task arch-generics#*` в указанных файлах.

Не меняется: `internal/application/`, `internal/domain/`, `internal/infrastructure/`, существующие тесты.

## Performance Budget

- `none` — рефакторинг не меняет runtime-поведение: generic router уже существует, handler closures выполняются при `init()`, nil-check — одна инструкция.

## Implementation Surfaces

| Файл | Роль | Вид |
|------|------|-----|
| `pkg/draftrag/search_routing.go` | 7 handler maps, ~270 строк — основной target рефакторинга | существующий |
| `pkg/draftrag/search_router.go` | generic `router[T]` + result types + новые helper types | существующий |
| `pkg/draftrag/search.go` | 7 SearchBuilder методов с panic | существующий |
| `pkg/draftrag/draftrag.go` | 7 Pipeline методов с panic | существующий |
| `pkg/draftrag/errors.go` | опциональный sentinel | существующий |

## Bootstrapping Surfaces

- `none` — все нужные структуры (router[T], result types) уже существуют в `search_router.go`.

## Влияние на архитектуру

- **Локальное**: только `pkg/draftrag/` — ни один internal-пакет не затрагивается.
- **Интеграции**: отсутствуют — публичный API (SearchBuilder, Pipeline) сохраняет полную обратную совместимость.
- **Migration/compatibility**: не требуется — breaking changes нет.

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|----|--------|----------|------------|
| AC-001 | Generic handler factory: 7 map'ов строятся через общий builder | `search_routing.go`, `search_router.go` | `git diff --stat` показывает `search_routing.go` ≤110 строк |
| AC-002 | Nil context guard: каждый публичный метод проверяет ctx==nil и возвращает error | `search.go`, `draftrag.go` | `grep -r "panic.*nil context" pkg/draftrag/` — 0 совпадений |
| AC-003 | Тесты проходят без изменений | все | `go test ./pkg/draftrag/...` pass |
| AC-004 | Каждый handler map строится из registry + factory | `search_routing.go` | code review: новый route = 1 entry в registry |
| AC-005 | Trace markers обновлены | `search_routing.go`, `search.go`, `search_router.go` | `grep -r "searchbuilder-generics" pkg/draftrag/` — 0 совпадений |

## Данные и контракты

- `data-model.md`: `no-change` — ни одна модель не меняется.
- API-контракты: не меняются — все публичные сигнатуры остаются.
- Event contracts: не применимо.

## Стратегия реализации

### DEC-001 Handler Factory (generic)

- **Why**: текущие 7 handler map'ов (retrieveHandlers, answerHandlers, citeHandlers, inlineCiteHandlers, streamHandlers, streamSourcesHandlers, streamCiteHandlers) идентичны по структуре: каждая содержит 6 closures, отличающихся только вызываемым методом core Pipeline и типом result-структуры. Выделение factory устраняет 42 повторяющихся closure.
- **Tradeoff**: factory требует объявления per-output-type builder (7 builders × ~3 строки = 21 строка против 42 closures), но net-saving ≈ 120 строк + единая точка расширения.
- **Affects**: `search_routing.go`, `search_router.go`.
- **Validation**: количество строк `search_routing.go` ≤110, добавление нового route требует 1 entry, а не 7 closures.

### DEC-002 Nil Context Guard Helper

- **Why**: 14+ повторяющихся `if ctx == nil { panic(...) }` в публичном API. Единый helper сокращает дублирование до одной строки на метод и гарантирует единый error-формат.
- **Tradeoff**: дополнительная функция-обёртка.
- **Affects**: `search.go`, `draftrag.go`, опционально `errors.go` (sentinel).
- **Validation**: `grep -r "panic.*nil context" pkg/draftrag/` — 0 совпадений.

### DEC-003 Trace Markers — Mechanical Rename

- **Why**: SpeckKeep требует актуальных маркеров. Замена `searchbuilder-generics#*` → `arch-generics#*` — mechanical find-and-replace.
- **Tradeoff**: нет.
- **Affects**: `search_routing.go`, `search.go`, `search_router.go`, `search_builder_test.go`, `search_router.go`.
- **Validation**: `grep -r "searchbuilder-generics" pkg/draftrag/` — 0 совпадений.

## Incremental Delivery

### MVP (Первая ценность)

1. `search_router.go`: добавить handler factory + per-output-type builder.
2. `search_routing.go`: переписать 7 handler map'ов через factory.
3. `search.go` + `draftrag.go`: заменить panic на error return.
4. Trace markers update.
5. `go build ./... && go test ./... && go vet ./...`.

Критерий готовности: `go test ./pkg/draftrag/...` pass, 0 panics, grep не находит старые маркеры.

### Итеративное расширение

- `none` — все AC покрываются MVP.

## Порядок реализации

1. **Handler factory** — фундамент: без него нельзя оценить итоговую структуру.
2. **Nil context guard** — независим, можно параллельно с п.1 после code review.
3. **Trace markers** — механический финальный шаг после подтверждения структуры.

## Риски

| Риск | Mitigation |
|------|------------|
| Generic factory не компилируется в Go 1.23 | `router[T]` уже существует и работает; factory — то же самое на один уровень абстракции выше |
| Тесты могут ожидать panic recovery | SearchBuilder тесты используют `require.NotPanics`; замена на error return — контрактное изменение в тестах, требует их обновления (входит в scope AC-002) |
| Пропущенный panic в неочевидном методе | После рефакторинга: `grep -r "panic.*nil context" pkg/draftrag/` |

## Rollout и compatibility

- Специальных rollout-действий не требуется.
- Breaking changes: нет — публичный API сохраняет сигнатуры.
- Monitoring: не применимо (библиотека, не сервис).

## Проверка

| Шаг | Команда | AC |
|-----|---------|----|
| Build | `go build ./...` | AC-003 |
| Test | `go test ./pkg/draftrag/...` | AC-003 |
| Vet | `go vet ./pkg/draftrag/...` | AC-001, AC-002 |
| Lint | `golangci-lint run ./pkg/draftrag/...` | AC-001, AC-002 |
| Panic check | `grep -r "panic.*nil context" pkg/draftrag/` | AC-002 |
| Trace check | `grep -r "searchbuilder-generics" pkg/draftrag/` | AC-005 |
| Line count | `wc -l pkg/draftrag/search_routing.go` | AC-001 (≤110) |

## Соответствие конституции

- нет конфликтов. Рефакторинг соответствует:
  - **Интерфейсная абстракция**: публичный API не меняется.
  - **Чистая архитектура**: изменения только в публичном слое `pkg/draftrag/`.
  - **Контекстная безопасность**: nil-контекст больше не вызывает panic — возвращает error.
  - **Поддерживаемость > cleverness**: generics — стандартная возможность Go 1.23+, не «метапрограммирование».
