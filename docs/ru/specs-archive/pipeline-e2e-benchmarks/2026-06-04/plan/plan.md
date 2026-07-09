# Pipeline E2E Benchmarks План

## Phase Contract

Inputs: spec.md, mocks из search_builder_test.go/pipeline_test.go, Pipeline API
Outputs: plan.md, data-model.md
Stop if: нет — spec детальна.

## MVP Slice

- `pkg/draftrag/pipeline_bench_test.go` — 3 Benchmark-функции: Index, Query, Full
- AC-001, AC-002, AC-003, AC-004, AC-005

## First Validation Path

`go test -bench=PipelineE2E -benchmem -count=1 ./pkg/draftrag/` — 3+ бенчмарка, benchstat-совместимый вывод.

## Scope

- Новый файл `pkg/draftrag/pipeline_bench_test.go` — все benchmark-функции и helper-ы
- Никаких изменений в production-коде
- Никаких изменений в существующих тестовых файлах

## Implementation Surfaces

| Surface | Почему участвует | Новая/сущ. |
|---------|-----------------|------------|
| `pkg/draftrag/pipeline_bench_test.go` | Benchmark-функции + bench helper-ы (setupBenchPipeline, genDocs) | Новая |
| `pkg/draftrag/draftrag.go` | Pipeline — Index/Query/Answer API, не меняется | Сущ. |
| `pkg/draftrag/search_builder_test.go` | mockLLM, fixedEmbedder — prototype для bench mocks | Сущ., без изменений |
| `pkg/draftrag/pipeline_test.go` | testEmbedder, testLLM — альтернативный стиль mocks | Сущ., без изменений |

## Bootstrapping Surfaces

`none` — все нужные структуры в репозитории уже есть.

## Влияние на архитектуру

- Локальное: только pkg/draftrag/ — один новый test-файл
- На интеграции: не влияет
- Migration/compatibility: не требуется

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|----|--------|----------|------------|
| AC-001 | IndexPipeline benchmark: gen N docs → Index() | `pipeline_bench_test.go` | `go test -bench=BenchmarkPipelineE2E_Index -benchmem` PASS |
| AC-002 | QueryPipeline benchmark: store prepopulated → Query() | `pipeline_bench_test.go` | `go test -bench=BenchmarkPipelineE2E_Query -benchmem` PASS |
| AC-003 | FullPipeline benchmark: Index + Query в одном b.N | `pipeline_bench_test.go` | `go test -bench=BenchmarkPipelineE2E_Full -benchmem` PASS |
| AC-004 | Short mode: уменьшенный датасет, <1s | `pipeline_bench_test.go` | `go test -bench=PipelineE2E -short` <1s |
| AC-005 | go vet clean | — | `go vet ./pkg/draftrag/` exit 0 |

## Данные и контракты

- Data model не меняется — см. `data-model.md` (stub: no-change)
- API/event контракты не меняются
- Helper-типы (benchEmbedder, benchLLM) живут только в `pipeline_bench_test.go`

## Стратегия реализации

### DEC-001 Helper-типы и setup в одном файле

Why: все benchmark-функции используют одни и те же mock-типы (benchEmbedder, benchLLM) и setupBenchPipeline. Один файл проще поддерживать, чем два. Mocks в pipeline_bench_test.go — копия существующих mockLLM/fixedEmbedder, чтобы не зависеть от внутренних тестовых типов других файлов.
Tradeoff: небольшая дупликация с search_builder_test.go, но файлы тестовые — дублирование допустимо.
Affects: `pipeline_bench_test.go`
Validation: `go build ./pkg/draftrag/` — OK.

### DEC-002 Data generation через genDocs(N, chunkSize)

Why: генерация документов заданного размера и количества через strings.Repeat. Каждый документ → Chunker (default) → N чанков с embedding. Позволяет параметризовать benchmark.
Tradeoff: docs создаются на каждый b.N, что добавляет overhead в измерение. Принято: overhead минимален (<1%).
Affects: `pipeline_bench_test.go`
Validation: genDocs(10, 100) → 10 документов по ~100 символов.

### DEC-003 Sub-benchmarks для разных размеров корпуса

Why: одна Benchmark-функция с `b.Run` для малого (10 docs), среднего (100 docs) и большого (1000 docs) корпуса. Даёт богатый baseline для benchstat.
Tradeoff: 1000 docs могут быть медленными — short mode ограничивает до 10.
Affects: `pipeline_bench_test.go`
Validation: `go test -bench=PipelineE2E_Index -benchmem` → 3 sub-benchmarks.

### DEC-004 QueryPipeline с предзаполненным store

Why: Query не индексирует — нужны данные в store. `setupBenchPipeline` создаёт Pipeline + заполняет store N документами, затем b.ResetTimer().
Tradeoff: setup не измеряется, что корректно.
Affects: `pipeline_bench_test.go`
Validation: Query возвращает непустой результат.

## Incremental Delivery

### MVP (Первая ценность)

1. `pipeline_bench_test.go` — benchEmbedder, benchLLM, setupBenchPipeline, genDocs
2. BenchmarkPipelineE2E_Index — sub-benchmarks (10, 100, 1000 docs)
3. BenchmarkPipelineE2E_Query — sub-benchmarks (10, 100, 1000 docs)
4. BenchmarkPipelineE2E_Full — index → query в одном benchmark
5. Short mode guard
6. `go vet` + прогон

## Порядок реализации

1. Helper-типы + genDocs + setupBenchPipeline
2. BenchmarkPipelineE2E_Index — базовая индексация
3. BenchmarkPipelineE2E_Query — query с предзаполненным store
4. BenchmarkPipelineE2E_Full — полный цикл
5. Short mode
6. Итоговый прогон + vet

Параллелить: шаги 2-4 можно писать последовательно (зависят от helper-ов из шага 1), но логика каждого независима.

## Риски

- **Риск 1**: Большой корпус (1000 docs) → benchmark >10s на прогон.
  Mitigation: large sub-benchmark только без -short; short mode ≈10 docs.
- **Риск 2**: benchstat noise >5% из-за GC.
  Mitigation: использовать `b.ReportAllocs()` и следить за allocs/op как secondary metric.

## Rollout и compatibility

Специальных rollout-действий не требуется. Новый файл только в test package.

## Проверка

| Шаг | Check | AC |
|-----|-------|----|
| Build | `go build ./pkg/draftrag/` | — |
| Index bench | `go test -bench=BenchmarkPipelineE2E_Index -benchmem -count=1` PASS | AC-001 |
| Query bench | `go test -bench=BenchmarkPipelineE2E_Query -benchmem -count=1` PASS | AC-002 |
| Full bench | `go test -bench=BenchmarkPipelineE2E_Full -benchmem -count=1` PASS | AC-003 |
| Short mode | `go test -bench=PipelineE2E -benchmem -short -count=1` <1s | AC-004 |
| Vet | `go vet ./pkg/draftrag/` exit 0 | AC-005 |

## Соответствие конституции

Нет конфликтов.
