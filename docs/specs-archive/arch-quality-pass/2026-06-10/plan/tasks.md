# Arch Quality Pass — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/hooks.go` | T1.2, T3.1 |
| `internal/application/hooks.go` | T1.2, T3.1 |
| `internal/application/pipeline.go` | T1.1, T2.1, T3.2, T3.3 |
| `pkg/draftrag/draftrag.go` | T1.1, T2.2, T3.3 |
| `pkg/draftrag/otel/hooks.go` | T1.2, T3.1 |
| `pkg/draftrag/otel/hooks_trace_test.go` | T4.2 |
| `internal/application/pipeline_constructors_test.go` | T1.2, T2.3, T4.1 |
| `internal/application/stream_backpressure_test.go` | T1.2, T2.3, T3.3 |
| `internal/infrastructure/resilience/embedder_test.go` | T1.2 |
| `internal/infrastructure/resilience/llm_test.go` | T1.2 |
| `pkg/draftrag/pipeline_coverage_test.go` | T2.3 |
| `pkg/draftrag/pipeline_errors_test.go` | T2.3 |
| `internal/application/pipeline_test.go` | T2.3 |
| `internal/application/batch_test.go` | T2.3, T3.3 |
| `internal/application/pipeline_index_concurrency_test.go` | T2.3, T3.3 |
| `internal/application/stream_backpressure_test.go` | T2.3, T3.3 |
| `internal/application/t4_1_coverage_test.go` | T2.3, T3.3 |
| `internal/application/prompt_context_limit_test.go` | T2.3, T3.3 |
| `internal/application/pipeline_methods_test.go` | T2.3, T3.3 |
| `internal/application/answer_with_citations_test.go` | T2.3, T3.3 |
| `internal/application/answer_inline_citations_test.go` | T2.3, T3.3 |
| `internal/application/retrieval_deduplication_test.go` | T2.3, T3.3 |
| `internal/application/retrieval_reranker_mmr_test.go` | T2.3, T3.3 |
| `internal/application/pipeline_options_test.go` | T2.3, T3.3 |
| `internal/application/observability_hooks_test.go` | T2.3 |
| `pkg/draftrag/search_builder_test.go` | T2.3 |
| `pkg/draftrag/pipeline_bench_test.go` | T2.3 |
| `pkg/draftrag/fuzz_test.go` | T2.3 |
| `pkg/draftrag/mistral_embedder_test.go` | T2.3 |
| `examples/*/main.go` (8 files) | T2.3 |
| `internal/infrastructure/resilience/embedder_test.go` | T2.3 |
| `internal/infrastructure/resilience/llm_test.go` | T2.3 |

## Implementation Context

- **Цель MVP**: убрать panics из конструкторов Pipeline, заменить на error return (AC-002, AC-003)
- **Инварианты/семантика**:
  - nil store/llm/embedder → panic (contract violation, остаётся как есть)
  - Все panics по валидации конфигурации → error return
  - `StageStart(ctx, ev)` → `StageStart(ctx, ev) context.Context`
  - Span создаётся в `StageStart`, завершается в `StageEnd` (без ретроспективного расчёта)
  - `PipelineConfig` заменяется на `PipelineOptions` с временным type alias
  - `DedupSourcesByParentID` → `DedupByParentID`
- **Ошибки/коды**: существующие sentinel-ы (domain.ErrEmptyQuery, domain.ErrEmbeddingDimensionMismatch etc.) не меняются. Для невалидной конфигурации — возвращаемые error-строки (конкретный тип не вводится).
- **Контракты/протокол**:
  - `func NewPipeline(store, llm, embedder) *Pipeline` → `func NewPipeline(store, llm, embedder) (*Pipeline, error)`
  - `func NewPipelineWithChunker(store, llm, embedder, chunker) *Pipeline` → `func NewPipelineWithChunker(store, llm, embedder, chunker) (*Pipeline, error)`
  - `func NewPipelineWithOptions(store, llm, embedder, opts) *Pipeline` → `func NewPipelineWithOptions(store, llm, embedder, opts) (*Pipeline, error)`
  - `type Hooks interface { StageStart(ctx, ev); StageEnd(ctx, ev) }` → `StageStart(ctx, ev) context.Context`
  - Все изменения compile-time проверяемые, breaking changes для pre-1.0
- **Границы scope**: не меняем output-методы Pipeline (Index/Query/Answer/Retrieve/DeleteDocument/UpdateDocument/IndexBatch). Не меняем store/llm/embedder интерфейсы.
- **DEC ссылки**: DEC-001 (последовательность 3 workstream), DEC-002 (type alias), DEC-003 (span → context)
- **Proof signals**: `go build ./...` успешен; `go test ./... -count=1` успешен; grep не находит panics в конструкторах; grep не находит PipelineConfig в internal/

## Фаза 1: Основа

Цель: подготовить hooks-контракт и compatibility alias, чтобы последующие фазы не ломали компиляцию на промежуточных шагах.

- [x] T1.1 Добавить re-export alias `type PipelineConfig = application.PipelineConfig` в `pkg/draftrag/draftrag.go`. Примечание: type alias `PipelineConfig = PipelineOptions` внутри `internal/application` невозможен из-за циклического импорта (`pkg/draftrag` → `internal/application`, обратный импорт запрещён). Alias в `pkg/draftrag` — корректный эквивалент, позволяющий внешним пользователям получать доступ к типу. Touches: `pkg/draftrag/draftrag.go`

- [x] T1.2 Изменить `domain.Hooks` — `StageStart` возвращает `context.Context`. Обновить `hookStart` в `internal/application/hooks.go`. Обновить `otel/hooks.go` (минимально — возвращать ctx). Обновить `mockHooks` в `pipeline_constructors_test.go`, `recordHooks` в `observability_hooks_test.go`, `countingHooks` в `stream_backpressure_test.go`, `MockHooks` в `resilience/embedder_test.go`, тестовые ожидания в `resilience/llm_test.go`. Обновить `produceChunks` в `pipeline.go` для использования traceCtx из hookStart. Touches: `internal/domain/hooks.go`, `internal/application/hooks.go`, `internal/application/pipeline.go`, `pkg/draftrag/otel/hooks.go`, `internal/application/pipeline_constructors_test.go`, `internal/application/observability_hooks_test.go`, `internal/application/stream_backpressure_test.go`, `internal/infrastructure/resilience/embedder_test.go`, `internal/infrastructure/resilience/llm_test.go`

## Фаза 2: MVP Slice

Цель: убрать panics из конструкторов Pipeline — заменить на error return (AC-002, AC-003).

- [x] T2.1 Заменить panics на error в `internal/application/pipeline.go`: `NewPipeline` (panics на nil остаются), `NewPipelineWithConfig` (StreamBufferSize < 0 → error). Сигнатура: `(*Pipeline, error)`. Touches: `internal/application/pipeline.go`

- [x] T2.2 Заменить panics на error в `pkg/draftrag/draftrag.go`: `NewPipelineWithOptions` (DefaultTopK < 0, MaxContextChars < 0, MaxContextChunks < 0, MMRCandidatePool < 0, MMRLambda вне [0,1] → error). `NewPipeline`, `NewPipelineWithChunker` — сигнатура `(*Pipeline, error)`. Никаких panics в production-коде не остаётся (кроме nil store/llm/embedder). Touches: `pkg/draftrag/draftrag.go`

- [x] T2.3 Обновить все call sites под новые сигнатуры `(*Pipeline, error)`:
  - 8 examples: `examples/*/main.go` — добавить `, err` и `if err != nil { ... }`
  - 18 тестовых файлов `internal/application` + `pkg/draftrag` — добавить `, err` и `require.NoError(t, err)`
  - panic recovery-тесты заменены на error-проверки (AC-002)
  Touches: 10 examples + 18 test files + 3 nil-arg panic test changes

- [x] T2.4 Подтвердить MVP: `go build ./...` и `go test ./... -count=1` успешны. Ни одного panic-вызова в конструкторах (кроме nil store/llm/embedder). Проверено grep. Touches: CI (локальный запуск)

## Фаза 3: Основная реализация

Цель: Hooks StageStart создаёт span (AC-001, AC-005) + единый struct конфигурации (AC-004).

- [x] T3.1 Реализовать OTel `StageStart` с созданием span:
  - В `pkg/draftrag/otel/hooks.go`: `StageStart` создаёт span через `h.tracer.Start`, кладёт его в возвращаемый `context.Context`
  - В `pkg/draftrag/otel/hooks.go`: `StageEnd` извлекает span из context, завершает его с duration и error-статусом (убрать ретроспективный расчёт startTime = now - duration)
  - В `internal/application/hooks.go`: `hookStart` принимает context и возвращает context (передаёт возвращённый из Hooks)
  - В `internal/application/pipeline.go`: `processDocumentOp`/`produceChunks` использует context из `hookStart` для embed/Upsert; передаёт его же в `hookEnd`
  - Обновить `mockHooks` — возвращать ctx (уже сделано в T1.2)
  Touches: `pkg/draftrag/otel/hooks.go`, `internal/application/hooks.go`, `internal/application/pipeline.go`

- [x] T3.2 Удалить `PipelineConfig` struct, удалить type alias, нормализовать имя поля:
  - Удалить `type PipelineConfig struct { ... }` из `internal/application/pipeline.go`
  - Удалить временный alias `type PipelineConfig = PipelineOptions`
  - Убедиться, что `NewPipelineWithConfig` принимает `PipelineOptions` (уже должно быть из T1.1)
  - Переименовать `DedupSourcesByParentID` → `DedupByParentID` в `pkg/draftrag.PipelineOptions`; обновить все ссылки в `pkg/draftrag/draftrag.go` и `internal/application/pipeline.go`
  Touches: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`

- [x] T3.3 Мигрировать 13 test files с `application.PipelineConfig` на `draftrag.PipelineOptions`:
  - Заменить `application.PipelineConfig{...}` на `draftrag.PipelineOptions{...}` во всех internal-тестах
  - Обновить импорты (добавить `draftrag "github.com/bzdvdn/draftrag/pkg/draftrag"` где нужно)
  - Normalize `DedupByParentID` → используется в тестах (уже правильное имя после T3.2)
  Touches: `internal/application/batch_test.go`, `internal/application/pipeline_index_concurrency_test.go`, `internal/application/stream_backpressure_test.go`, `internal/application/t4_1_coverage_test.go`, `internal/application/prompt_context_limit_test.go`, `internal/application/pipeline_methods_test.go`, `internal/application/answer_with_citations_test.go`, `internal/application/answer_inline_citations_test.go`, `internal/application/retrieval_deduplication_test.go`, `internal/application/retrieval_reranker_mmr_test.go`, `internal/application/pipeline_options_test.go`, `internal/application/pipeline_constructors_test.go`, `internal/application/observability_hooks_test.go`

## Фаза 4: Проверка

Цель: automated coverage для всех AC + финальные проверки.

- [x] T4.1 Добавить тесты на error вместо panic для всех невалидных параметров:
  - `TestNewPipelineWithOptions_DefaultTopK_Invalid` — DefaultTopK < 0
  - `TestNewPipelineWithOptions_MaxContextChars_Invalid` — MaxContextChars < 0
  - `TestNewPipelineWithOptions_MaxContextChunks_Invalid` — MaxContextChunks < 0
  - `TestNewPipelineWithOptions_MMRCandidatePool_Invalid` — MMRCandidatePool < 0
  - `TestNewPipelineWithOptions_MMRLambda_Invalid` — MMRLambda < 0 и > 1
  - `TestNewPipelineWithOptions_StreamBufferSize_Invalid` — StreamBufferSize < 0
  - `TestNewPipelineWithOptions_ValidZeroConfig` — все поля zero, возвращает (p, nil) с дефолтами (AC-003)
  Touches: `pkg/draftrag/pipeline_coverage_test.go` (или `pkg/draftrag/pipeline_options_test.go`)

- [x] T4.2 Добавить тест на Hooks StageStart со span в контексте:
  - `TestHooks_StageStart_CreatesSpan` в `pkg/draftrag/otel/hooks_trace_test.go`: `StageStart` возвращает context; проверить, что в context есть активный span через `trace.SpanFromContext`; `StageEnd` завершает span; проверить `StartTime` ≈ момент вызова StageStart (AC-005)
  Touches: `pkg/draftrag/otel/hooks_trace_test.go`

- [x] T4.3 Финальные проверки:
  - `go build ./...` — без ошибок
  - `go vet ./...` — без предупреждений
  - `go test ./... -count=1` — все тесты зелёные
  - `grep -r 'panic(' internal/application/pipeline.go pkg/draftrag/draftrag.go` — только nil-проверки (store, llm, embedder)
  - `grep -r 'PipelineConfig' internal/` — 0 результатов
  - `grep -r 'DedupSourcesByParentID' .` — 0 результатов
  - `grep -r 'NewPipeline\b' examples/` — все вызовы с `, err` и обработкой ошибки
  Touches: CI, локальный запуск

## Покрытие критериев приемки

- AC-001 Hooks StageStart возвращает context → T1.2, T3.1, T4.2
- AC-002 Конструкторы возвращают error вместо panic → T2.1, T2.2, T2.3, T4.1
- AC-003 Обратная совместимость для валидной конфигурации → T2.3, T4.1
- AC-004 Единый struct конфигурации → T1.1, T3.2, T3.3
- AC-005 StageStart в OTel создаёт span → T1.2, T3.1, T4.2

## Заметки

- Фазы выполняются строго последовательно (DEC-001): Фаза 1 → Фаза 2 → Фаза 3 → Фаза 4. Каждый шаг оставляет код в компилируемом состоянии.
- Все изменения публичной сигнатуры — breaking changes для pre-1.0. CHANGELOG обновляется в T4.3.
- Panics на nil store/llm/embedder остаются — не трогать.
- Не добавлять новые файлы — все изменения в существующих.
