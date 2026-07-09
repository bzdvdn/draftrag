# Pipeline E2E Benchmarks

## Scope Snapshot

- In scope: parameterized go benchmark suite для full RAG pipeline (index + query + answer), измеряющий throughput и allocs на InMemoryStore с mock-зависимостями.
- Out of scope: benchmarks для отдельных компонентов вне pipeline (chunker, embedder, LLM, vector store по отдельности — они уже существуют или не требуются).

## Цель

Разработчик, работающий над оптимизацией Pipeline, сейчас не может измерить регрессию — нет baseline. После внедрения benchmark suite `go test -bench=PipelineE2E -benchmem ./pkg/draftrag/` даёт стабильные цифры для benchstat-сравнения. Успех измеряется: suite покрывает ≥3 бенчмарк-сценария (index, query, full RAG), benchstat на двух последовательных прогонах показывает <5% noise.

## Основной сценарий

1. Стартовая точка: Pipeline создаётся с InMemoryStore, mock-Embedder и mock-LLM.
2. Benchmark: для разных конфигураций (кол-во документов, размер чанков, topK) измеряются throughput и allocs.
3. Результат: benchstat-совместимый вывод для 3+ сценариев.
4. Fallback: при `testing.Short()` выполняется только один mini-прогон (1 документ, 1 query).

## User Stories

- P1 (MVP): benchmarks для IndexPipeline (документ → чанкинг → embed → upsert) и QueryPipeline (query → embed → search → answer).
- P2: benchmarks для разных размеров корпуса (10/100/1000 docs) и topK значений.

## MVP Slice

- `pkg/draftrag/pipeline_bench_test.go` — 3 Benchmark-функции: IndexPipeline, QueryPipeline, FullPipeline.
- Каждый benchmark использует `b.ReportAllocs()` и `b.ResetTimer()`.
- Результат: benchstat-совместимый вывод для памяти и времени.

## First Deployable Outcome

- `go test -bench=PipelineE2E -benchmem ./pkg/draftrag/ -count=1` — 3 бенчмарка, каждый >1ms, данные для benchstat.
- `go test -bench=PipelineE2E -benchmem -short ./pkg/draftrag/ -count=1` — mini-прогон для быстрой проверки.

## Scope

- `pkg/draftrag/pipeline_bench_test.go` — новый файл: BenchmarkPipelineE2E_Index, BenchmarkPipelineE2E_Query, BenchmarkPipelineE2E_Full.
- `pkg/draftrag/bench_test.go` — опционально: общие helper-ы/фикстуры.
- InMemoryStore + mock Embedder/LLM — уже существуют в `pkg/draftrag/search_builder_test.go`.

## Контекст

- InMemoryStore — reference implementation, не требует внешних зависимостей.
- Mock Embedder/LLM уже определены в search_builder_test.go — можно переиспользовать или вынести в bench_test.go.
- Существующие бенчмарки в репозитории используют последовательный `for i := 0; i < b.N; i++` без `RunParallel`.
- Pipeline — struct в `pkg/draftrag/draftrag.go`, конструкторы `NewPipeline` / `NewPipelineWithChunker` / `NewPipelineWithOptions`.
- Вызов `NewPipelineWithOptions` с пустыми `PipelineOptions` эквивалентен `NewPipeline`.

## Требования

- RQ-001 Benchmark DОЛЖЕН измерять IndexPipeline: Document → Chunker (text split) → Embedder → Upsert.
- RQ-002 Benchmark DОЛЖЕН измерять QueryPipeline: Query → Embedder → Search → LLM Generate.
- RQ-003 Benchmark DОЛЖЕН измерять FullPipeline: индексация нескольких документов + query по ним.
- RQ-004 Каждый Benchmark DОЛЖЕН использовать `b.ReportAllocs()` и `b.ResetTimer()`.
- RQ-005 При `testing.Short()` набор данных ДОЛЖЕН быть минимальным (1 документ, 1 query).

## Вне scope

- Benchmarks для отдельных компонентов (chunker, embedder, LLM, vector store по отдельности).
- Benchmarks с реальными HTTP-бекендами (pgvector, qdrant и т.д.).
- Race-детектор в benchmarks (замедляет, не нужен для baseline).
- Parallel benchmarks (`b.RunParallel`) — последовательные стабильнее для baseline.

## Критерии приемки

### AC-001 IndexPipeline benchmark

- Почему это важно: индексация — первый этап RAG, её производительность влияет на UX при batch-загрузке.
- **Given** Pipeline с InMemoryStore, mock Embedder, Chunker по умолчанию
- **When** `go test -bench=BenchmarkPipelineE2E_Index -benchmem -count=1 ./pkg/draftrag/`
- **Then** benchmark выполняется, вывод содержит ns/op и B/op, allocs/op
- Evidence: `go test -bench=BenchmarkPipelineE2E_Index -benchmem -count=3 ./pkg/draftrag/ | benchstat /dev/stdin` стабилен (<5% variance)

### AC-002 QueryPipeline benchmark

- Почему это важно: query — самый частый path, latency критична.
- **Given** Pipeline с InMemoryStore (предзаполнен документами), mock Embedder, mock LLM
- **When** `go test -bench=BenchmarkPipelineE2E_Query -benchmem -count=1 ./pkg/draftrag/`
- **Then** benchmark выполняется, вывод содержит ns/op и B/op, allocs/op
- Evidence: `go test -bench=BenchmarkPipelineE2E_Query -benchmem -count=3 ./pkg/draftrag/ | benchstat /dev/stdin` стабилен

### AC-003 FullPipeline benchmark (index + query)

- Почему это важно: полный цикл RAG — реалистичный сценарий.
- **Given** Pipeline с InMemoryStore, mock Embedder, mock LLM, Chunker
- **When** `go test -bench=BenchmarkPipelineE2E_Full -benchmem -count=1 ./pkg/draftrag/`
- **Then** benchmark выполняется, вывод содержит ns/op и B/op, allocs/op
- Evidence: `go test -bench=BenchmarkPipelineE2E_Full -benchmem -count=3 ./pkg/draftrag/ | benchstat /dev/stdin` стабилен

### AC-004 Short mode

- Почему это важно: быстрая проверка без нагрузки.
- **Given** Pipeline с InMemoryStore
- **When** `go test -bench=PipelineE2E -benchmem -short -count=1 ./pkg/draftrag/`
- **Then** benchmarks выполняются с минимальным датасетом
- Evidence: прогон занимает <1s

### AC-005 go vet + build без errors

- Почему это важно: код должен быть идиоматичным.
- **Given** все изменения завершены
- **When** `go vet ./pkg/draftrag/`
- **Then** exit code 0
- Evidence: `go vet ./pkg/draftrag/` PASS

## Допущения

- InMemoryStore — reference для benchmark; абсолютные цифры не переносимы на production store.
- mock Embedder возвращает фиксированный вектор (нулевая работа) — измеряем infrastructure overhead, не compute.
- mock LLM возвращает фиксированный ответ — измеряем overhead pipeline, не LLM inference.
- Benchmarks не используют `b.RunParallel` — последовательные стабильнее для benchstat.

## Критерии успеха

- SC-001 ≥3 benchmark-функции, каждая с `b.ReportAllocs()`.
- SC-002 benchstat variance <5% на 3 последовательных прогонах.
- SC-003 Short mode завершается <1s.

## Краевые случаи

- Пустой Document (без Content) — Chunker возвращает 0 чанков, Upsert не вызывается.
- Query без текста — возврат ошибки валидации, benchmark измеряет быстрый path.
- topK=0 — ошибка `ErrInvalidQueryTopK`, benchmark измеряет error path.

## Открытые вопросы

- Выносить ли helper-структуры (mockEmbedder, mockLLM, setupBenchPipeline) в отдельный bench_test.go или оставить в pipeline_bench_test.go? Решение в plan.
- Нужен ли бенчмарк для UpdateDocument (delete + re-index)? В P2, если pipeline его поддерживает.
