# Middleware Chain — Задачи

## Phase Contract

Inputs: plan.md, data-model.md, spec.md.
Outputs: исполнимые задачи с покрытием AC-001–AC-005.
Stop if: задачи расплывчаты или coverage не сопоставляется.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/middleware.go` | T1.1 |
| `internal/application/middleware.go` | T1.2 |
| `internal/application/pipeline.go` | T2.1, T2.2 |
| `internal/application/query.go` | T2.3 |
| `internal/application/answer.go` | T2.3 |
| `internal/application/stream.go` | T2.4 |
| `pkg/draftrag/draftrag.go` | T2.5 |
| `internal/application/` (все методы) | T3.1, T3.2 |
| `internal/application/*_test.go` | T4.1, T4.2 |
| `examples/middleware/main.go` | T2.6 |

## Implementation Context

- **Цель MVP:** Middleware interface + цепочка + интеграция в Query/Answer/Index (produceChunks)/AnswerStream + example.
- **Инварианты:** Middleware — функциональный тип `func(next Handler) Handler` (DEC-001). `StageData` — единая структура (DEC-002). Цепочка в application (DEC-003). Streaming — pre-stream + channel wrapper (DEC-004).
- **Ошибки:** `ErrMiddlewareAbort` (sentinel для short-circuit). Любая ошибка из middleware = fail-fast (RQ-004).
- **Контракты:** nil middleware = no-op. Hooks/PIIDetector не меняются.
- **Границы scope:** Не реализуем конкретные middleware (логгер, PII). Не заменяем Hooks/PIIDetector.
- **Proof signals:** `go test ./... -run Middleware` pass. `go run ./examples/middleware` показывает порядок.

## Фаза 1: Основа

Цель: подготовить domain-интерфейс и application-движок цепочки.

- [x] T1.1 Создать `internal/domain/middleware.go` — интерфейс `Middleware` (`func(next Handler) Handler`), тип `Handler`, тип `StageData` (Stage, Operation, Query, Document, Chunks, Answer, Embedding), sentinel `ErrMiddlewareAbort`. Touches: `internal/domain/middleware.go`
- [x] T1.2 Создать `internal/application/middleware.go` — функция `runMiddleware(middlewares []domain.Middleware, stage domain.HookStage, op string, ctx context.Context, data domain.StageData) (domain.StageData, error)`. Nil/пустой срез возвращает data без изменений. Любая ошибка прерывает цепочку. Touches: `internal/application/middleware.go`

## Фаза 2: MVP Slice

Цель: интеграция в ключевые методы pipeline + публичный API + пример.

- [x] T2.1 Добавить `Middleware []domain.Middleware` в `application.PipelineOptions` и поле `middleware` в `application.Pipeline`. Пробросить в `NewPipelineWithConfig`. Touches: `internal/application/pipeline.go`
- [x] T2.2 Интегрировать middleware в `produceChunks`: pre_chunking/post_chunking вокруг `chunker.Chunk`, pre_embed/post_embed вокруг `embedder.Embed`. Touches: `internal/application/pipeline.go`
- [x] T2.3 Интегрировать middleware в `Pipeline.Query` (pre_embed/post_embed, pre_search/post_search) и `Pipeline.Answer` (pre_embed/post_embed, pre_search/post_search, pre_generate/post_generate). Touches: `internal/application/query.go`, `internal/application/answer.go`
- [x] T2.4 Интегрировать middleware в `AnswerStream`: pre_generate (модификация запроса) + обёртка выходного канала для post-generate. Touches: `internal/application/stream.go`
- [x] T2.5 Re-export `Middleware`, `StageData` в `pkg/draftrag/draftrag.go`. Добавить `Middleware []Middleware` в `PipelineOptions`. Пробросить в `application.PipelineOptions` в `NewPipelineWithOptions`. Touches: `pkg/draftrag/draftrag.go`
- [x] T2.6 Создать `examples/middleware/main.go` — pipeline с двумя middleware (логгер + PII-цензор). `go run` показывает порядок вызова. Touches: `examples/middleware/main.go`

## Фаза 3: Основная реализация

Цель: интеграция middleware во все pipeline-методы для полного покрытия стадий.

- [x] T3.1 Интегрировать middleware в методы query-группы: `QueryHyDE`, `QueryMulti`, `QueryWithParentIDs`, `QueryWithMetadataFilter`, `QueryHybrid`, `QueryWithQueries`, `Query` (decompose path). Все стадии embed/search получают pre/post middleware. Touches: `internal/application/query.go`
- [x] T3.2 Интегрировать middleware в методы answer-группы: `AnswerWithCitations`, `AnswerWithInlineCitations`, `AnswerWithParentIDs`, `AnswerWithMetadataFilter`, `AnswerHyDE[WithCitations]`, `AnswerMulti[WithCitations]`, `AnswerHybrid[WithCitations][WithInlineCitations]`, `AnswerWithQueries*`, `generateAnswer`, `generateCitedFromResult`, `generateInlineCitedFromResult`. Все стадии embed/search/generate получают pre/post middleware. Touches: `internal/application/answer.go`

## Фаза 4: Проверка

Цель: краевые случаи, тесты, верификация.

- [x] T4.1 Добавить unit-тесты: AC-001 (порядок через Index/Query/Answer), AC-002 (error short-circuit + spy), AC-005 (identity с nil middleware). Touches: `internal/application/pipeline_middleware_test.go`
- [x] T4.2 Добавить unit-тесты: AC-003 (счётчик стадий для Answer/Index/AnswerStream), AC-004 (модификация вопроса на pre-generate). Panic recovery, context cancel, empty/nil middleware. Touches: `internal/application/pipeline_middleware_test.go`
- [x] T4.3 Проверить: `go vet ./...`, `go fmt ./...`, `golangci-lint run ./...`. Benchmark SC-001 (3 no-op middleware < 5% latency). Touches: `Makefile` (если нужен benchmark target)

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T2.3, T4.1
- AC-002 -> T1.2, T2.1, T2.3, T4.1
- AC-003 -> T2.3, T3.1, T3.2, T4.2
- AC-004 -> T1.1, T2.3, T4.2
- AC-005 -> T1.2, T2.1, T4.1
