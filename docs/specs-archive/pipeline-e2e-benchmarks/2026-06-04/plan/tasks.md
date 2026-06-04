# Pipeline E2E Benchmarks — Задачи

## Phase Contract

Inputs: plan.md, spec.md, pkg/draftrag/draftrag.go, pkg/draftrag/search_builder_test.go
Outputs: tasks.md
Stop if: нет — plan детален, AC привязаны.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/pipeline_bench_test.go` | T1.1, T2.1, T2.2, T2.3, T3.1 |

## Implementation Context

- **Цель MVP**: 3 Benchmark-функции (Index, Query, Full) с sub-benchmarks для 10/100/1000 docs, benchmem, short mode.
- **Инварианты/семантика**:
  - Pipeline создаётся через `NewPipelineWithChunker(store, llm, embedder, chunker)` для Index/Full, `NewPipeline(store, llm, embedder)` для Query
  - `benchEmbedder` возвращает фиксированный вектор `[]float64{1, 0, 0}` (нулевая работа)
  - `benchLLM` возвращает `"answer"` (нулевая работа)
  - Chunker по умолчанию — `domain.NewBasicChunker(1000, 200)` (из `pkg/draftrag/basic_chunker.go`)
- **Ошибки/коды**: не тестируются (только happy path в benchmarks)
- **Контракты/протокол**: `genDocs(n, contentSize) []Document` — генератор; `setupBenchPipeline` возвращает `*Pipeline`
- **Границы scope**: не трогаем production-код, не добавляем новые типы в public API
- **Proof signals**: `go test -bench=PipelineE2E -benchmem -count=1 ./pkg/draftrag/` — 3+ бенчмарка, benchstat-совместимый вывод
- **References**: DEC-001 (всё в одном файле), DEC-002 (genDocs), DEC-003 (sub-benchmarks), DEC-004 (предзаполненный store)

## Фаза 1: Основа

Цель: helper-типы, генератор документов, setup-функция.

- [x] T1.1 Создать `pipeline_bench_test.go`: benchEmbedder, benchLLM, genDocs, setupBenchPipeline. Helper-типы — копия mockLLM/fixedEmbedder из search_builder_test.go. genDocs принимает count и contentSize, возвращает `[]domain.Document`. setupBenchPipeline создаёт Pipeline + InMemoryStore. Touches: `pkg/draftrag/pipeline_bench_test.go`

## Фаза 2: MVP Benchmarks

Цель: 3 Benchmark-функции с sub-benchmarks для разных размеров корпуса.

- [x] T2.1 Реализовать BenchmarkPipelineE2E_Index: genDocs → Pipeline.Index. Sub-benchmarks: docs10/docs100/docs1000. short mode → docs10 только. `b.ReportAllocs()`, `b.ResetTimer()`. Touches: `pkg/draftrag/pipeline_bench_test.go`
- [x] T2.2 Реализовать BenchmarkPipelineE2E_Query: setupBenchPipeline с предзаполненным store → Query. Sub-benchmarks: docs10/docs100/docs1000. Время setup не измеряется. Touches: `pkg/draftrag/pipeline_bench_test.go`
- [x] T2.3 Реализовать BenchmarkPipelineE2E_Full: Index + Query в одном b.N цикле. Sub-benchmarks: docs10/docs100/docs1000. short → docs10. Touches: `pkg/draftrag/pipeline_bench_test.go`

## Фаза 3: Проверка

Цель: доказать, что benchmarks работают и дают стабильный baseline.

- [x] T3.1 Финальная проверка: `go test -bench=PipelineE2E -benchmem -count=1 ./pkg/draftrag/` — 3+ бенчмарка PASS; `go test -bench=PipelineE2E -benchmem -short -count=1 ./pkg/draftrag/` <1s; `go vet ./pkg/draftrag/` exit 0. Touches: `pkg/draftrag/pipeline_bench_test.go`

## Покрытие критериев приемки

- AC-001 -> T2.1, T3.1
- AC-002 -> T2.2, T3.1
- AC-003 -> T2.3, T3.1
- AC-004 -> T2.1, T2.2, T2.3 (short guard), T3.1
- AC-005 -> T3.1

## Заметки

- T3.1 — единственная задача валидации (прогон инструментов + bench output).
- Short mode реализуется через `if testing.Short() { n = 10 }` в genDocs или в sub-bench выборе.
- Все helper-типы в одном файле (DEC-001).
