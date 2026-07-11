# Sub-query decomposition Задачи

## Phase Contract

Inputs: spec, plan (pass), data-model (no-change).
Outputs: tasks с фазами, Surface Map, покрытие AC.
Stop if: — нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/domain/models.go` | T3.3 |
| `internal/infrastructure/decomposer/llm.go` | T2.1 |
| `internal/infrastructure/decomposer/rule.go` | T3.1 |
| `internal/infrastructure/decomposer/composite.go` | T3.1 |
| `internal/application/subdecompose.go` | T1.3, T3.3 |
| `pkg/draftrag/draftrag.go` | T1.2 |
| `pkg/draftrag/search.go` | T1.2 |
| `pkg/draftrag/search_routing.go` | T1.2, T2.2, T3.2 |
| `pkg/draftrag/errors.go` | T1.1 |
| `pkg/draftrag/reranker_llm.go` | — |
| `internal/infrastructure/decomposer/*_test.go` | T2.3, T4.1 |
| `internal/application/subdecompose_test.go` | T2.3, T4.1 |
| Makefile / CI | T4.2 |

## Implementation Context

- Цель MVP: `Search("...").SubDecompose().Retrieve/Answer(ctx)` — LLM decomposes → parallel retrieve → merge → answer.
- Инварианты/семантика:
  - `QueryDecomposer` возвращает `[]string` (nil/пустой = fallback to single-query)
  - Merge: dedup по `Chunk.ID`, max score per chunk, сортировка по score desc
  - Sub-query topK = исходный topK, не настраивается индивидуально
  - Parallel execution: `errgroup` с concurrency limit (по умолчанию 4)
- Ошибки/коды:
  - `ErrSubDecomposeNotSupported` — nil decomposer + SubDecompose()
  - Graceful degradation: ошибка декомпозиции → логируем → single-query
- Контракты/протокол:
  - LLM decomposer input: `"Разбей запрос на независимые под-вопросы. Ответь JSON-массивом строк: [\"...\", \"...\"]"`
  - LLM output: JSON-массив строк (толерантный парсинг: regex extraction если JSON невалиден)
  - `Decompose(ctx, query string) ([]string, error)` — interface
- Границы scope:
  - Не делаем weighted sub-questions, DAG execution, multi-turn decomposition
  - Не меняем data model (Chunk, Document, RetrievalResult)
- Proof signals:
  - mockLLM возвращает 2+ sub-questions → mockStore.Search вызван 2+ раз с разными query
  - merged результат: unique Chunk.ID count < raw count (при дубликатах)
  - AC-007: goroutine start/end timestamps перекрываются
- References: DEC-001 (отдельный интерфейс), DEC-002 (errgroup), DEC-003 (merge max score), DEC-004 (composite), DEC-005 (новый route)

## Фаза 1: Основа

Цель: Domain interface, PipelineOptions, SearchBuilder, routing entry.

- [x] T1.1 Добавить `QueryDecomposer` interface в `internal/domain/interfaces.go` и sentinel `ErrSubDecomposeNotSupported` в `pkg/draftrag/errors.go`.
  Touches: `internal/domain/interfaces.go`, `pkg/draftrag/errors.go`
  AC: AC-001, AC-006

- [x] T1.2 Добавить `QueryDecomposer` поле в `PipelineOptions` + `Pipeline` (`pkg/draftrag/draftrag.go`); метод `SubDecompose()` и поле `subDecompose bool` на `SearchBuilder` (`pkg/draftrag/search.go`); новый route `routeSubDecompose` + handler-записи в `retrieveHandlers`/`answerHandlers` (`pkg/draftrag/search_routing.go`).
  Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/search.go`, `pkg/draftrag/search_routing.go`
  AC: AC-001, AC-006

- [x] T1.3 Реализовать `QuerySubDecompose` в `internal/application/subdecompose.go`: параллельная (errgroup) embed+search по под-вопросам, merge (dedup by Chunk.ID + max score), hooks per sub-query. Базовая версия без Answer (только retrieve).
  Touches: `internal/application/subdecompose.go`
  AC: AC-002, AC-004, AC-007

## Фаза 2: MVP Slice

Цель: LLM-based decomposer + routing handlers + Answer.

- [x] T2.1 Реализовать `LLMQueryDecomposer` в `internal/infrastructure/decomposer/llm.go`: системный prompt + LLMProvider.Generate + толерантный JSON-парсинг (regex fallback).
  Touches: `internal/infrastructure/decomposer/llm.go`
  AC: AC-002

- [x] T2.2 Реализовать handler-функции `subDecomposeRetrieve`/`subDecomposeAnswer` в `search_routing.go`: decompose → parallel retrieve → merge → (for Answer) LLM generate. Добавить handler-записи в `citeHandlers`/`inlineCiteHandlers`/`streamHandlers`/`streamSourcesHandlers`/`streamCiteHandlers`.
  Touches: `pkg/draftrag/search_routing.go`, `internal/application/subdecompose.go`
  AC: AC-001, AC-008, AC-009

- [x] T2.3 Написать unit-тесты для MVP path: `TestPipeline_QuerySubDecompose*` (mock LLM returning JSON, mock Store), `TestLLMQueryDecomposer*` (valid/invalid JSON, error), проверка parallel execution.
  Touches: `internal/application/subdecompose_test.go`, `internal/infrastructure/decomposer/llm_test.go`
  AC: AC-001, AC-002, AC-007

## Фаза 3: Основная реализация

Цель: Rule-based decomposer, composite, edge cases, оставшиеся routing handlers.

- [x] T3.1 Реализовать `RuleQueryDecomposer` (разбиение по союзам/разделителям "и","или",",") и `CompositeDecomposer` (LLM→Rule→single fallback) в `internal/infrastructure/decomposer/`.
  Touches: `internal/infrastructure/decomposer/rule.go`, `internal/infrastructure/decomposer/composite.go`
  AC: AC-003, AC-005

- [x] T3.2 Добавить handler-записи `subDecomposeCite`/`subDecomposeInlineCite`/`subDecomposeStream`/`subDecomposeStreamSources`/`subDecomposeStreamCite` во все router maps.
  Touches: `pkg/draftrag/search_routing.go`, `internal/application/subdecompose.go`
  AC: AC-009

- [x] T3.3 Обработать edge cases: nil decomposer → `ErrSubDecomposeNotSupported`; context cancellation → cancel all sub-queries; один под-вопрос → skip merge overhead; zero результатов → empty context (как обычный Answer).
  Touches: `internal/application/subdecompose.go`, `internal/domain/models.go` (если нужен sentinel), `pkg/draftrag/draftrag.go`
  AC: AC-005, AC-006

## Фаза 4: Проверка

Цель: Полное тестовое покрытие, lint, vet.

- [x] T4.1 Написать тесты для P2-AC: AC-003 (rule-based), AC-004 (merge dedup), AC-005 (composite fallback chain), AC-006 (per-request override), AC-008 (Answer integration), AC-009 (Cite/Stream).
  Touches: `internal/application/subdecompose_test.go`, `internal/infrastructure/decomposer/rule_test.go`, `internal/infrastructure/decomposer/composite_test.go`, `pkg/draftrag/search_builder_test.go`
  AC: AC-003, AC-004, AC-005, AC-006, AC-008, AC-009

- [x] T4.2 Запустить `go vet`, `golangci-lint`, `go test ./...` — без ошибок.
  AC: all

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T2.2, T2.3
- AC-002 -> T1.3, T2.1, T2.3
- AC-003 -> T3.1, T4.1
- AC-004 -> T1.3, T4.1
- AC-005 -> T3.1, T3.3, T4.1
- AC-006 -> T1.1, T1.2, T3.3, T4.1
- AC-007 -> T1.3, T2.3
- AC-008 -> T2.2, T4.1
- AC-009 -> T2.2, T3.2, T4.1

## Заметки

- T1.1 → T1.2 → T1.3 — жёсткая последовательность (interface → SearchBuilder → application)
- T2.1 (decomposer) независим от T2.2 (routing handlers) — можно параллелить
- T3.1 (rule+composite) можно реализовать после MVP, не дожидаясь T3.2/T3.3
- Все новые файлы (decomposer/) нужно создать в bootstrapping phase (implicit в T2.1)
