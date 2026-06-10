# Arch Quality Pass — План

## Цель

Три независимых точечных улучшения production-safety и поддерживаемости draftRAG: (1) замена panics на errors в конструкторах Pipeline, (2) новый контракт Hooks с возвратом context из StageStart, (3) устранение дублирования PipelineConfig/PipelineOptions.

Все три workstream могут быть выполнены последовательно в одном PR. Каждый имеет свою зону ответственности, не конфликтующую с другими.

## MVP Slice

Panic→error замена (RQ-002, AC-002, AC-003). Минимальный срез: после него ни один публичный конструктор не паникует при невалидной конфигурации.

## First Validation Path

```go
// Быстрая проверка: пример с невалидной конфигурацией
_, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DefaultTopK: -1,
})
require.Error(t, err) // раньше был panic
```

После реализации: `go build ./...` и `go test ./... -count=1`.

## Scope

1. `internal/domain/hooks.go` — контракт: StageStart возвращает context.Context
2. `internal/application/hooks.go` — обновление вызовов hookStart
3. `pkg/draftrag/otel/hooks.go` — OTel реализация: span в StageStart
4. `internal/application/pipeline.go` — NewPipeline/NewPipelineWithConfig: error return
5. `pkg/draftrag/draftrag.go` — NewPipeline/NewPipelineWithChunker/NewPipelineWithOptions: error return + единый struct
6. Все `*_test.go` — миграция сигнатур
7. Все 8 `examples/*/main.go` — миграция сигнатур

**Граница**: nil-проверки store/llm/embedder остаются panic. Output-методы Pipeline (Index, Query, Answer…) не меняют сигнатуры.

## Implementation Surfaces

| Surface | Изменение | Статус |
|---|---|---|
| `internal/domain/hooks.go` | StageStart возвращает `context.Context` | существующий |
| `internal/application/hooks.go` | `hookStart` возвращает/передаёт context | существующий |
| `internal/application/pipeline.go` | error return из NewPipeline/NewPipelineWithConfig; StreamBufferSize<0 → error | существующий |
| `internal/application/pipeline.go` | `processDocumentOp`/`produceChunks`: использовать context из `hookStart` для embed/Upsert | существующий |
| `internal/domain/interfaces.go` | новый кастомный тип `HookContext` или просто return `context.Context` из `Hooks` | существующий |
| `pkg/draftrag/draftrag.go` | error return из всех NewPipeline*; удаление PipelineConfig; нормализация имён полей | существующий |
| `pkg/draftrag/otel/hooks.go` | StageStart создаёт span; StageEnd только завершает | существующий |
| `pkg/draftrag/otel/hooks_trace_test.go` | обновление теста под новый контракт | существующий |
| `*_test.go` (18 test files) | миграция вызовов `NewPipeline(…)` → `NewPipeline(…)` с `, err` | существующий |
| `examples/*/main.go` (8 файлов) | миграция вызовов с `, err` | существующий |

## Bootstrapping Surfaces

- `none` — все файлы уже существуют, новый код не создаётся.

## Влияние на архитектуру

- **Hooks контракт**: лёгкое изменение интерфейса — добавляется возврат `context.Context`. Существующие реализации перестают компилироваться (кроме OTel, которая будет обновлена). Это ожидаемый breaking change.
- **PipelineConfig удаление**: `internal/application.PipelineConfig` заменяется на `pkg/draftrag.PipelineOptions`. Тесты, использующие `PipelineConfig`, мигрируются.
- **Error return**: все конструкторы меняют сигнатуру с `*Pipeline` на `(*Pipeline, error)`. Breaking change для всех вызовов.
- **Нет новых зависимостей**: OTel уже в `go.mod`, `tracetest` уже используется.

## Acceptance Approach

### AC-001 Hooks StageStart возвращает context

- **Подход**: изменить `domain.Hooks` — `StageStart(ctx, ev) context.Context`. `StageStartEvent` может остаться без изменений (span создаётся внутри, маршаллится в context).
- **Surfaces**: `internal/domain/hooks.go`, `internal/application/hooks.go`, `pkg/draftrag/otel/hooks.go`
- **Наблюдение**: тест с `tracetest.NewSpanRecorder` из `otel/hooks_trace_test.go` — проверить, что span создаётся в StageStart, а не в StageEnd.

### AC-002 Конструкторы возвращают error вместо panic

- **Подход**: все публичные конструкторы (`pkg/draftrag/draftrag.go`) и внутренние (`internal/application/pipeline.go`) возвращают `(*Pipeline, error)`. Валидация в начале, error при невалидных параметрах.
- **Surfaces**: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
- **Наблюдение**: `TestNewPipeline_InvalidConfig` — для каждого значения <0 проверка error.

### AC-003 Обратная совместимость

- **Подход**: тесты с нулевой конфигурацией продолжают возвращать Pipeline с дефолтами.
- **Surfaces**: все тесты, использующие `NewPipeline(store, llm, embedder)` без опций.
- **Наблюдение**: `TestNewPipeline_Success` проходит с `(p, nil)`.

### AC-004 Единый struct конфигурации

- **Подход**: удалить `internal/application.PipelineConfig`. `application.NewPipelineWithConfig` принимает `draftrag.PipelineOptions`. Ввести `type PipelineConfig = draftrag.PipelineOptions` как internal alias в `internal/application/pipeline.go` для плавной миграции тестов.
- **Surfaces**: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`, 13 test files с `PipelineConfig`.
- **Наблюдение**: `grep -r "PipelineConfig" internal/` находит 0 результатов (после миграции тестов и удаления alias).

### AC-005 StageStart в OTel создаёт span

- **Подход**: в `StageStart` создать span через `h.tracer.Start`, записать его в возвращаемый `context.Context`. В `StageEnd` найти span из context или из внутреннего хранилища (spanID в context).
- **Surfaces**: `pkg/draftrag/otel/hooks.go`
- **Наблюдение**: тест `TestHooks_StageStart_CreatesSpan` — `StageStart` возвращает context с активным span; `StageEnd` завершает его. `StartTime` span'а соответствует моменту вызова StageStart.

## Данные и контракты

### Data Model

**Status**: `no-change`
**Причина**: не добавляются и не меняются доменные сущности. Hooks-события (`StageStartEvent`, `StageEndEvent`) остаются без изменений. Единственное изменение — возврат `context.Context` из метода интерфейса.

### API Contracts

- `domain.Hooks` — изменён контракт:
  ```go
  // Было:
  StageStart(ctx context.Context, ev StageStartEvent)
  // Стало:
  StageStart(ctx context.Context, ev StageStartEvent) context.Context
  ```
- `PipelineConfig` удалён из `internal/application`. Те, кто импортировал его, должны перейти на `draftrag.PipelineOptions`. Compatibility bridge: `type PipelineConfig = draftrag.PipelineOptions` (временный alias).

### No New Contracts

Контракты хранилищ, LLM, эмбеддеров не меняются.

## Стратегия реализации

### DEC-001 Последовательное выполнение трёх workstream

- **Why**: workstream-ы затрагивают пересекающиеся файлы (`internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`), но можно выполнить их в порядке: (1) Hooks contract → (2) panic→error → (3) единый struct. Каждый шаг оставляет код в компилируемом состоянии (с учётом миграции тестов).
- **Tradeoff**: последовательность означает единый большой PR, а не 3 маленьких. Альтернатива — 3 отдельных PR с временным compatibility-кодом, что сложнее поддерживать.
- **Affects**: все surfaces.
- **Validation**: `go build ./... && go test ./...` после каждого workstream.

### DEC-002 Временный type alias для PipelineConfig

- **Why**: 13 test files напрямую импортируют `application.PipelineConfig`. Вместо единовременной миграции всех тестовых файлов — временный alias `type PipelineConfig = PipelineOptions` в `internal/application/pipeline.go`. После миграции тестов alias удаляется.
- **Tradeoff**: alias удлиняет период "техдолга" — пакет временно экспортирует два имени для одного типа. Плюс: тесты не ломаются между коммитами.
- **Affects**: `internal/application/pipeline.go`, 13 test files.
- **Validation**: `go build ./...` успешен на всех промежуточных шагах.

### DEC-003 StageStart кладёт span в context, StageEnd извлекает

- **Why**: чтобы не городить внутреннее хранилище (sync.Map с ID), используем стандартную практику — span создаётся и кладётся в возвращаемый context. Вызывающий pipeline передаёт этот context через стадии, и на StageEnd span завершается.
- **Tradeoff**: требование, чтобы context из StageStart пробрасывался в StageEnd. В текущей архитектуре pipeline передаёт единый ctx через стадии — это выполняется. При нарушении цепочки (код не использует возвращённый context) span не завершится в StageEnd и утечёт.
- **Affects**: `internal/domain/hooks.go`, `internal/application/hooks.go`, `internal/application/pipeline.go`, `pkg/draftrag/otel/hooks.go`.
- **Validation**: тест `TestHooks_StageStart_CreatesSpan` проверяет span в context после StageStart.

## Incremental Delivery

### MVP (Первая ценность)

**Workstream 1: Panic→error** (достаточно для AC-002, AC-003)

1. Поменять сигнатуры всех конструкторов `internal/application` и `pkg/draftrag` на `(*Pipeline, error)`
2. Перенести валидацию параметров в начало конструктора, возвращать error
3. Обновить все call sites (~ 314) — тесты и examples
4. `go build && go test` — зелёный

Проверка: `go test -run TestNewPipeline -count=1` и ручной вызов с `DefaultTopK=-1` возвращает error.

### Итеративное расширение

**Workstream 2: Hooks context** (добавляет AC-001, AC-005)

1. Поменять `domain.Hooks`: `StageStart(ctx, ev) context.Context`
2. Обновить `mockHooks` во всех тестовых файлах (возвращать ctx)
3. Обновить `hookStart` в `internal/application/hooks.go`: принимать и передавать context
4. Обновить OTel реализацию: span в StageStart, завершение в StageEnd
5. `go build && go test`

**Workstream 3: Единый struct** (добавляет AC-004)

1. Создать временный alias `type PipelineConfig = PipelineOptions` в `internal/application/pipeline.go`
2. Поменять `application.NewPipelineWithConfig` на `NewPipelineWithConfig(store, llm, embedder, opts PipelineOptions)`
3. Мигрировать 13 test files с `PipelineConfig` на `PipelineOptions`
4. Удалить исходный `PipelineConfig` struct + alias
5. Нормализовать имя `DedupSourcesByParentID` → `DedupByParentID` в `PipelineOptions`
6. `go build && go test`

## Порядок реализации

1. **Workstream 1 (Panic→error)** — самый безопасный, минимальный scope, даёт production-safety первым.
2. **Workstream 2 (Hooks context)** — зависит от того, что тесты компилируются, но не зависит от struct-рефакторинга. Можно параллелить с WS1, если делать в разных файлах, но безопаснее последовательно.
3. **Workstream 3 (Единый struct)** — зависит от WS1 (сигнатуры уже поменялись). Выполняется последним.

## Риски

### Риск 1: Пропущенные call sites после изменения сигнатур

- **Mitigation**: `go build ./...` в CI гарантирует полную компиляцию. Изменение сигнатуры — compile-time проверяемое.
- **Наблюдение**: все ~314 call sites видны компилятору.

### Риск 2: StageStart context не пробрасывается в StageEnd

- **Mitigation**: в текущей архитектуре pipeline передаёт единый `ctx` через все стадии. После `hookStart` возвращённый context используется для embed/Upsert. На `hookEnd` передаётся тот же context. Тест с tracetest.SpanRecorder проверяет цепочку.
- **Наблюдение**: если context потерян на пути к StageEnd, span не завершится и будет висеть до завершения tracer'а.

### Риск 3: PipelineOptions.DedupSourcesByParentID переименование ломает external users

- **Mitigation**: struct field rename — compile-time проверяемое изменение. Если пользователь использует named fields, компилятор укажет на ошибку. Если использует zero-value struct — rename невидим. Добавить `Deprecated:` comment на старое имя, если оставить как alias.

## Rollout и compatibility

- **Breaking change**: все три workstream-а меняют публичный API (сигнатуры конструкторов, Hooks interface, имя поля). Это допустимо для pre-1.0 библиотеки.
- **CHANGELOG**: описать каждое изменение и миграционный путь.
- **Feature flag**: не требуется — изменения compile-time проверяемые.
- **Rollback**: откат коммита восстанавливает предыдущее поведение.

## Проверка

### Automated tests

| Уровень | Что добавить/обновить |
|---|---|
| Unit (internal/application) | `TestNewPipeline_InvalidConfig_*` — error вместо panic; `TestNewPipeline_NilStore` — panic осталась |
| Unit (pkg/draftrag) | `TestNewPipelineWithOptions_InvalidConfig_*` — все поля <0; `TestNewPipelineWithOptions_ValidConfig_Default` |
| Unit (domain) | `TestHooks_StageStart_ReturnsContext` — новый тест |
| Unit (otel) | `TestHooks_StageStart_CreatesSpan` — span в возвращённом context |
| Existing | Все существующие тесты обновлены под новые сигнатуры |
| Integration | `go build ./... && go test ./... -count=1` |

### Manual checks

1. `grep -r 'panic(' internal/application/pipeline.go pkg/draftrag/draftrag.go` — только nil-проверки
2. `grep -r 'PipelineConfig' internal/` — 0 результатов (после workstream 3)
3. `grep -r 'NewPipeline\b' examples/` — все вызовы с `, err` и обработкой

### AC/DEC coverage

| AC/DEC | Тип проверки |
|---|---|
| AC-001 | Unit: hooks_trace_test.go |
| AC-002 | Unit: test with invalid config |
| AC-003 | Unit: test with zero config |
| AC-004 | grep: PipelineConfig удалён |
| AC-005 | Unit: tracetest.SpanRecorder |
| DEC-001 | Code review: последовательность |
| DEC-002 | Code review: alias + grep |
| DEC-003 | Code review + unit test |

## Соответствие конституции

- **Clean Architecture**: не нарушается. Hooks интерфейс остаётся в domain, изменения в application/ и infrastructure/.
- **Контекст во всех публичных операциях**: RQ-001 усиливает это требование — StageStart возвращает context.
- **go vet/go fmt/golangci-lint**: все проходят.
- **Unit-тесты**: добавлены тесты под каждый AC.
- **нет конфликтов**.
