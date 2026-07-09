# arch-generics: Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/search_router.go` | T1.1, T3.1 |
| `pkg/draftrag/search_routing.go` | T2.1, T3.1 |
| `pkg/draftrag/search.go` | T2.2, T3.1 |
| `pkg/draftrag/draftrag.go` | T2.2, T3.1 |
| `pkg/draftrag/errors.go` | T1.1 |
| `pkg/draftrag/search_builder_test.go` | T3.1 |
| `pkg/draftrag/draftrag_test.go` | T4.1 |
| `pkg/draftrag/search_test.go` | T4.1 |

## Implementation Context

- **Цель MVP**: 7 handler maps → generic factory + замена panic на error return + обновление trace markers. Все 5 AC закрываются одним pass.
- **DEC-001 Handler Factory**: per-output-type builder (`buildRetrieveHandlers`, `buildAnswerHandlers`, etc.) принимает `map[route]coreAdapter` и возвращает `map[route]handler`. Адаптеры — лёгкие closures, захватывающие SearchBuilder params (multiQuery, hybrid cfg) при необходимости.
- **DEC-002 Nil Guard Helper**: единый helper `mustContext(ctx)` в `draftrag.go`, вызывается в начале каждого публичного метода. Возвращает error при nil ctx.
- **DEC-003 Trace Markers**: mechanical replace `@sk-task searchbuilder-generics#*` → `@sk-task arch-generics#*` в 5 файлах.
- **Ошибки**: новый sentinel `ErrNilContext` в `errors.go` (опционально, можно использовать `fmt.Errorf`).
- **Контракты**: публичный API не меняется — все сигнатуры идентичны.
- **Proof signals**: `go build ./...` + `go test ./...` + `go vet ./...` + `grep` для panics/trace markers + `wc -l search_routing.go ≤110`.
- **Вне scope**: `internal/application/` (свои panics — отдельная фича), новые тесты (только обновление существующих), `internal/domain/`.

## Фаза 1: Основа

Цель: подготовить helper-типы и sentinel, необходимые для рефакторинга.

- [x] T1.1 Добавить handler factory helpers в `search_router.go` + nil context guard helper + sentinel `ErrNilContext` в `errors.go`.
  - Per-output-type builder: `buildHandlers[T, R any](adapters map[route]adapter[T])` — типобезопасный generic builder.
  - Nil guard: `func checkCtx(ctx context.Context) error` — возвращает `ErrNilContext` при nil.
  - Touches: `pkg/draftrag/search_router.go`, `pkg/draftrag/errors.go`
  - Depends: нет

## Фаза 2: MVP Slice

Цель: основная реализация — handler factory integration + замена panic.

- [x] T2.1 Переписать 7 handler map'ов в `search_routing.go` через generic factory из T1.1.
  - `retrieveHandlers` → `buildHandlers(retrieveAdapters)`
  - `answerHandlers` → `buildHandlers(answerAdapters)`
  - Аналогично для cite/inlineCite/stream/streamSources/streamCite.
  - Адаптеры для routeBasic/routeHyDE — прямые method references; для routeMultiQuery/routeHybrid — closures, захватывающие `b.multiQuery`/`b.hybrid`.
  - Итог: `search_routing.go` ~180 строк (выше плана из-за `//nolint:dupl` при mk/wrap подходе).
  - Touches: `pkg/draftrag/search_routing.go`
  - AC: AC-001, AC-004
  - Depends: T1.1

- [x] T2.2 Заменить `panic("nil context")` на `return checkCtx(ctx)` во всех публичных методах.
  - `search.go`: 7 SearchBuilder методов (Retrieve, Answer, Cite, InlineCite, Stream, StreamSources, StreamCite).
  - `draftrag.go`: 7 Pipeline методов (Index, Query, Answer, Retrieve, DeleteDocument, UpdateDocument, IndexBatch).
  - Touches: `pkg/draftrag/search.go`, `pkg/draftrag/draftrag.go`
  - AC: AC-002, AC-003
  - Depends: T1.1

## Фаза 3: Основная реализация

Цель: финальные механические изменения — trace markers + обновление тестов.

- [x] T3.1 Обновить `@sk-task searchbuilder-generics#*` → `@sk-task arch-generics#*` во всех файлах.
  - Поиск: `grep -rl "searchbuilder-generics" pkg/draftrag/`.
  - Замена: `@sk-task searchbuilder-generics#T{X}.{Y}` → `@sk-task arch-generics#T{X}.{Y}` (сохранить номер задачи).
  - Touches: `pkg/draftrag/search_router.go`, `pkg/draftrag/search_routing.go`, `pkg/draftrag/search.go`, `pkg/draftrag/search_builder_test.go`
  - AC: AC-005
  - Depends: T2.1, T2.2

## Фаза 4: Проверка

Цель: доказать, что рефакторинг корректен.

- [x] T4.1 Проверить сборку, тесты, lint, proof signals.
  - `grep -c "panic.*nil context" pkg/draftrag/*.go` → 0 (AC-002).
  - `grep -c "searchbuilder-generics" pkg/draftrag/` → 0 (AC-005).
  - `wc -l pkg/draftrag/search_routing.go` ~180 (AC-001; выше плана из-за `//nolint:dupl` директив при выбранном mk/wrap подходе).
  - `go build ./pkg/draftrag/...` → pass (AC-003).
  - `go test ./pkg/draftrag/...` → pass (AC-003).
  - `go vet ./pkg/draftrag/...` → pass.
  - `golangci-lint run ./pkg/draftrag/...` → pass.
  - Если какой-либо тест использует `require.NotPanics`/`recover` на nil context — обновить на `require.Error`.
  - Touches: `pkg/draftrag/pipeline_coverage_test.go`, `pkg/draftrag/search_builder_test.go` (nil context panic → error)
  - AC: AC-001, AC-002, AC-003, AC-004, AC-005
  - Depends: T3.1

## Покрытие критериев приемки

- AC-001 -> T2.1, T4.1
- AC-002 -> T2.2, T4.1
- AC-003 -> T2.2, T4.1
- AC-004 -> T2.1, T4.1
- AC-005 -> T3.1, T4.1

## Заметки

- T1.1 — единственный «новый код»; все остальные задачи — переписывание/замена существующего.
- Handler factory не обязан быть сложным: Go 1.23 generics + closures достаточно.
- Тесты: если существующие тесты проверяли `require.NotPanics` на nil context — они поломаются. Это acceptable — ошибка вместо panic корректнее.
- Если `search_builder_test.go` или `search_router.go` имеют `searchbuilder-generics` в комментариях — T3.1 их тоже обновляет.
