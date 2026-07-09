# Рефакторинг SearchBuilder: Generics + единый routing — План

## Phase Contract

Inputs: `docs/specs/searchbuilder-generics/spec.md`, `docs/specs/searchbuilder-generics/inspect.md`
Outputs: `plan.md`, `data-model.md`

## Цель

Устранить дублирование 42 switch-кейсов в `search_routing.go` через generic-маршрутизатор. Каждый output-метод получает единую точку входа с валидацией и `mapAppError`, а не дублирует switch. `internal/application/pipeline.go` и публичный API SearchBuilder не меняются.

## MVP Slice

- Generic-тип `router[T any]` с методом `execute`, выполняющим `pickRoute → вызов handler[r] → mapAppError`
- 7 result-structs (по одному на каждый output-метод)
- Ручная регистрация handler-ов в каждом output-методе (42 closures → 1 line каждый)
- AC-001, AC-002, AC-004

## First Validation Path

```bash
go test ./pkg/draftrag/ -run TestSearchBuilder -count=1 -v
go vet ./pkg/draftrag/...
golangci-lint run ./pkg/draftrag/...
```

## Scope

- `pkg/draftrag/search_routing.go` — удаление 7 switch-функций, добавление `router[T]`
- `pkg/draftrag/search_router.go` — **новый файл**: `router[T]`, result-structs, `execute`
- `pkg/draftrag/search.go` — вызов `b.pickRoute()` вынесен в `router.execute`; output-методы становятся тоньше
- `pkg/draftrag/search_builder_test.go` — добавление AC-002 table-driven test `TestSearchBuilder_RouteMatrix`

## Implementation Surfaces

| Surface | Тип | Зачем |
|---------|-----|-------|
| `pkg/draftrag/search_router.go` | новый | Определение `router[T any]`, result-structs, `execute` |
| `pkg/draftrag/search_routing.go` | переработка | Удалить `runRetrieve/runAnswer/runCite/runInlineCite/runStream/runStreamSources/runStreamInline`; регистрация handler-ов |
| `pkg/draftrag/search.go` | минимально | Output-методы: убрать `b.pickRoute()`, вызывать `router.execute` вместо `run*` |
| `pkg/draftrag/search_builder_test.go` | расширение | Новый `TestSearchBuilder_RouteMatrix` (table-driven, 42 subtests) |
| `pkg/draftrag/draftrag.go` | нет | Без изменений |

## Bootstrapping Surfaces

`pkg/draftrag/search_router.go` — новый файл, должен существовать до изменений в `search_routing.go`.

## Влияние на архитектуру

- Локальное: только `pkg/draftrag/`, слой public API.
- Никаких изменений в `internal/application/pipeline.go`, `internal/domain/`.
- Никаких изменений публичного API — SearchBuilder, его методы и их сигнатуры сохраняются.
- `pickRoute()` остаётся на SearchBuilder (зависит от полей экземпляра), не выносится в `router.execute`.

## Acceptance Approach

### AC-001 Все output-методы работают через generic router

- **Подход:** рефакторинг `search_routing.go` + `search.go`; существующие тесты доказывают сохранность поведения
- **Surfaces:** `search_router.go`, `search_routing.go`, `search.go`
- **Наблюдение:** `go test ./pkg/draftrag/ -run TestSearchBuilder -count=1` pass

### AC-002 Покрытие всех комбинаций маршрут × output-метод

- **Подход:** новый `TestSearchBuilder_RouteMatrix` с распаковкой 6×7 = 42 subtests через `t.Run`
- **Surfaces:** `search_builder_test.go`
- **Наблюдение:** каждый subtest вызывает конкретный output-метод с конкретным маршрутом и проверяет отсутствие `ErrEmptyQuery`/`ErrInvalidTopK`/panic

### AC-003 Добавление нового output-метода — 5 строк

- **Подход:** prototype `Analyze(ctx) (Analysis, error)` в `search_builder_test.go` внутри того же файла (не в основном коде)
- **Surfaces:** `search_builder_test.go` (prototype), `search_routing.go` (измерение LoC)
- **Наблюдение:** code review — строки, добавляемые в `search_routing.go` в теле output-метода (не struct definition, не handler registration)

### AC-004 `go vet` и `golangci-lint` без errors

- **Подход:** CI gate
- **Surfaces:** все
- **Наблюдение:** CI artefact или локальный прогон

## Данные и контракты

Создан `data-model.md` со статусом `no-change`:
- Никакие доменные модели не меняются
- Никакие публичные типы не меняются
- Единственные новые типы — внутренние `router[T]` и result-structs, не экспортируемые

## Стратегия реализации

### DEC-001 Result-structs для multi-return

**Why:** Go generics не параметризуют разную арность возврата. Единственный type-safe способ — именованные struct, каждая под конкретный output-метод.

```go
type rRetrieve struct{ Result domain.RetrievalResult }
type rAnswer struct{ Text string }
type rCite struct{ Text string; Sources domain.RetrievalResult }
type rInlineCite struct{ Text string; Sources domain.RetrievalResult; Citations []domain.InlineCitation }
type rStream struct{ Ch <-chan string }
type rStreamSources struct{ Ch <-chan string; Sources domain.RetrievalResult }
type rStreamCite struct{ Ch <-chan string; Sources domain.RetrievalResult; Citations []domain.InlineCitation }
```

**Tradeoff:** +1 allocation на вызов (struct vs tuple return). Не значимо — struct на стёке, escape только если возвращается в interface.

**Affects:** `search_router.go`

**Validation:** `go test -bench=BenchmarkSearchBuilder -benchmem` no regression (p > 0.05 vs baseline)

### DEC-002 Handler-ы регистрируются в var, не в хот-пасе

**Why:** `SearchBuilder` — мутабельный (fluent API), нельзя pre-compute handler-ы на уровне пакета. Каждый output-метод определяет свой slice handler-ов как package-level `var`, инициализируемый через `init()` или `sync.Once`.

Альтернатива: инициализация при первом вызове (lazy via `sync.OnceValue`). Выбрана — предпочтительнее `init()` т.к. не разогревает рантайм без необходимости.

**Tradeoff:** `sync.Once` — пара атомиков на первую регистрацию, незначительно.

**Affects:** `search_routing.go`

**Validation:** race-free при параллельных вызовах из разных горутин.

### DEC-003 `pickRoute` остаётся на SearchBuilder

**Why:** `pickRoute` зависит от полей экземпляра `SearchBuilder` (`.hyDE`, `.multiQuery`, `.hybrid`, `.parentIDs`, `.filter`). Вынос в `router[T]` потребовал бы копирования или передачи билдера. Оставляем как метод SearchBuilder.

**Tradeoff:** `execute` принимает `route route` как аргумент вместо вычисления. Дополнительное действие для каждой публичной функции, но тривиальное.

**Affects:** `search.go`, `search_routing.go`

### DEC-004 mapAppError в `execute`, не в хендлере

**Why:** `mapAppError` применяется ко всем возвращаемым ошибкам единообразно. Вынос в `execute` устраняет 7 копий.

**Tradeoff:** handler-ы возвращают application-level ошибки, execute маппит в public — чистая ответственность.

**Affects:** `search_router.go`

### DEC-005 Handler-ы принимают `*application.Pipeline`, не `*SearchBuilder`

**Why:** handler-у нужен только `pipeline.core` для вызова `Query*`/`Answer*`. SearchBuilder содержит лишний контекст (временные настройки флюентного API). Передаём pipeline.core напрямую.

**Tradeoff:** для handler-ов с extra-параметрами (multiQuery n, hybrid cfg) нужен захват из SearchBuilder на момент регистрации — но эти параметры меняются при каждом fluent-вызове. Решение: передавать `b.topK` и `b.multiQuery`/`b.hybrid` как аргументы execute или в билдере.

Уточнение: handler сигнатура `func(ctx, q string, topK int, b *SearchBuilder) (T, error)` — передаём билдер целиком, handler сам решает, какие поля использовать.

**Affects:** `search_router.go`

## Incremental Delivery

### MVP (Первая ценность)

P1 из spec: все 7 output-методов работают через `router[T]`.

- Создать `search_router.go` с `router[T]`, result-structs, `execute`
- Переписать `search_routing.go`: удалить `run*`, добавить handler-ы
- Output-методы в `search.go`: `b.pickRoute()` → `router.execute()`
- AC-001, AC-002, AC-004

Проверка: `go test ./pkg/draftrag/` pass, `go vet`, `golangci-lint`.

### Итеративное расширение

P2 из spec: prototype нового output-метода `Analyze`.

- Добавить result-struct `rAnalyze` и handler-ы
- Написать `TestSearchBuilder_Analyze` в `search_builder_test.go`
- Измерить количество добавляемых строк
- AC-003

## Порядок реализации

1. **`search_router.go`** — `router[T]`, result-structs, `execute`. Независим, может быть написан первым.
2. **`search_routing.go`** — удалить `run*`, добавить handler registration. Зависит от (1).
3. **`search.go`** — output-методы переключаются на `router.execute`. Зависит от (1, 2).
4. **`search_builder_test.go`** — `TestSearchBuilder_RouteMatrix`. Параллельно с (2, 3).
5. **Prototype `Analyze`** — после MVP. Можно безопасно параллельно с (4).

## Риски

| Риск | Mitigation |
|------|------------|
| Generic-инстанциирование может не инлайниться (escape result-struct) | DEC-001: struct на стёке, если не содержит `chan`. Проверить `-benchmem` в AC-001/SC-002 |
| Handler-ы в `sync.OnceValue` создают замыкания с циклом | Использовать `[7]func` literal без цикла, или явно перечислить 7 handler-ов |
| `pickRoute` должен быть вызван до `execute` — easy to forget | `execute` принимает `route` как обязательный параметр; не скомпилируется без него |
| Регресс производительности при большом числе аллокаций | SC-002: benchstat-сравнение до/после |

## Rollout и compatibility

Breaking changes нет — весь рефакторинг internal. Единственный observable эффект — возможная микро-деградация производительности (SC-002). Никаких миграций или feature flags.

## Проверка

| Шаг | Что проверяем | AC/DEC |
|-----|---------------|--------|
| `go test ./pkg/draftrag/ -run TestSearchBuilder -count=1` | Все output-методы работают, старые тесты pass | AC-001 |
| `go test ./pkg/draftrag/ -run TestSearchBuilder_RouteMatrix -count=1 -v` | Покрытие 42 комбинаций | AC-002 |
| `go vet ./pkg/draftrag/... && golangci-lint run ./pkg/draftrag/...` | Идиоматичность, безопасность | AC-004, DEC-002 |
| `go test -bench=BenchmarkSearchBuilder -benchmem -count=10 > old.txt && ... > new.txt && benchstat old.txt new.txt` | p > 0.05 регресс | DEC-001, SC-002 |
| Code review prototype `Analyze` | ≤5 строк в теле output-метода | AC-003 |
| `go test -race ./pkg/draftrag/ -count=1` | Race-free | DEC-002 |

## Соответствие конституции

- нет конфликтов
- Сохранены: Clean Architecture (нижние слои не тронуты), `context.Context` во всех публичных операциях, unit-тесты для всех изменений
- Явное решение DEC-001 (простота > расширяемость — result-structs проще, чем кодогенерация или reflection)
