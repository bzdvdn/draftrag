# Middleware Chain — План

## Phase Contract

Inputs: spec + repo surfaces (domain interfaces, application pipeline, public API).
Outputs: plan, data-model.md.
Stop if: spec недостаточно детализирована — spec прошла inspect pass.

## Цель

Внедрить единую middleware-цепочку, встраиваемую во все стадии pipeline (chunking, embed, search, generate) без изменения существующих контрактов. Middleware получает типизированные данные стадии и вызывает next — шаблон, знакомый по `net/http`. Пустая цепочка (nil) не меняет поведение.

## MVP Slice

Middleware интерфейс + цепочка в `internal/application/` + интеграция в `Query`, `Answer`, `AnswerStream`, `Index`. Без middleware pipeline идентичен текущему.

Покрывает: AC-001 (порядок), AC-002 (ошибка прерывает), AC-005 (без middleware = no-op).

## First Validation Path

`go test ./internal/application/ -run TestMiddlewareChain -v` — unit-тесты с collect-списком и sentinel-ошибкой. После — `go run ./examples/middleware` показывает порядок вызова + модификацию данных.

## Scope

- `internal/domain/middleware.go` — новый файл: `Middleware`, `StageData` (или `Request`/`Response`), sentinel `ErrMiddlewareAbort`.
- `internal/application/middleware.go` — новый файл: цепочка выполнения.
- `internal/application/pipeline.go` — `PipelineOptions.Middleware`, `Pipeline.middleware`.
- `internal/application/query.go`, `answer.go`, `stream.go`, `pipeline.go` — интеграция middleware в методы.
- `pkg/draftrag/draftrag.go` — re-export `Middleware`, `PipelineOptions.Middleware`, проброс.
- `internal/domain/interfaces.go` — не трогать.
- `internal/domain/hooks.go` — не трогать (Hooks живут параллельно).
- `internal/domain/pii.go` — не трогать (PIIDetector мигрируется отдельно, если вообще).

## Performance Budget

- `none` — middleware добавляет 1 вызов интерфейса на stage при пустой цепочке (nil-check) и ≤ N вызовов при N middleware. Дополнительные аллокации: одна `StageData` на stage при наличии middleware, 0 при nil.

## Implementation Surfaces

| Surface | Change | Почему |
|---------|--------|--------|
| `internal/domain/middleware.go` | **Новый** | Интерфейс `Middleware` + тип данных стадии |
| `internal/application/middleware.go` | **Новый** | Цепочка: `runMiddleware(middlewares, stage, ctx, data)` |
| `internal/application/pipeline.go` | Patch | `PipelineOptions.Middleware` + поле `Pipeline.middleware` |
| `internal/application/query.go` | Patch | middleware до/после embed/search |
| `internal/application/answer.go` | Patch | middleware до/после embed/search/generate |
| `internal/application/pipeline.go` | Patch | middleware до/после chunking/embed в `produceChunks` |
| `internal/application/stream.go` | Patch | middleware до generate, обёртка канала |
| `pkg/draftrag/draftrag.go` | Patch | re-export + `PipelineOptions.Middleware` + проброс в core |

## Bootstrapping Surfaces

- `internal/domain/middleware.go` — создаётся первым, т.к. от него зависят все остальные поверхности.
- `internal/application/middleware.go` — создаётся вторым (зависит от domain).
- Остальные поверхности — существующие файлы, только patch.

## Влияние на архитектуру

- **Локальное**: Новый файл в domain и application; добавление опции и поля в Pipeline (оба слоя).
- **Границы**: Hooks и PIIDetector не затрагиваются — параллельные механизмы.
- **Compatibility**: `PipelineOptions` получает новое поле; nil/default = старое поведение.
- **Migration**: не требуется — additive change.

## Acceptance Approach

### AC-001 (порядок выполнения)

- Подход: collect-список в middleware, проверка порядка.
- Surfaces: `internal/domain/middleware.go`, `internal/application/middleware.go`, `internal/application/pipeline.go` (Index).
- Наблюдение: `[]string{"A","B","C"}` в тесте с Index/Query/Answer.

### AC-002 (ошибка прерывает)

- Подход: middleware, возвращающая sentinel-ошибку.
- Surfaces: `internal/application/middleware.go` (fail-fast в цепочке).
- Наблюдение: spy на `store.Search` + `llm.Generate` — не вызывались.

### AC-003 (все стадии)

- Подход: счётчик стадий в middleware.
- Surfaces: query.go, answer.go, stream.go, pipeline.go.
- Наблюдение: для Answer — pre_search, post_search, pre_generate, post_generate.

### AC-004 (модификация данных)

- Подход: middleware заменяет вопрос на pre-generate.
- Surfaces: answer.go (точка pre_generate).
- Наблюдение: mock LLM получает модифицированный вопрос.

### AC-005 (без middleware)

- Подход: identity-тест: nil middleware vs эталон.
- Surfaces: все.
- Наблюдение: идентичные результаты.

## Данные и контракты

- AC-001–AC-005 покрывают все RQ.
- Сущности: `Document`, `Chunk`, `RetrievalResult`, HookStage — не меняются.
- API-контракты (VectorStore, LLMProvider и т.д.) — не меняются.
- `data-model.md` — no-change, т.к. новые типы (Middleware, StageData) не заменяют существующие.

## Стратегия реализации

### DEC-001 Интерфейс Middleware — функциональный (func-based), не интерфейсный

- **Why**: Функциональный Middleware (`type Middleware func(next Handler) Handler`) идиоматичен для Go (net/http, zerolog, chi). Позволяет использовать замыкания для конфигурации без отдельного struct. Интерфейсный подход потребовал бы метод `Wrap` и не дал бы преимуществ, т.к. middleware редко заменяется через DI.
- **Tradeoff**: Функциональный тип нельзя использовать в type-switch для интроспекции. Для данного случая не нужно.
- **Affects**: `internal/domain/middleware.go`, `internal/application/middleware.go`.
- **Validation**: compile-time проверка: `var _ domain.Middleware = func(next domain.Handler) domain.Handler { return next }`.

### DEC-002 StageData — единая структура для всех стадий

- **Why**: Единый `StageData` с опциональными полями (Query, Document, Chunks, Answer, Embedding) проще, чем N отдельных типов под каждую стадию. Middleware может проверять `StageData.Stage` и читать только релевантные поля. Альтернатива (type-specific handlers) потребовала бы N middleware-интерфейсов или unsafe type assertions.
- **Tradeoff**: Неиспользуемые поля — zero value; риск случайного чтения нулевого поля не выше риска ошибочного type assertion.
- **Affects**: `internal/domain/middleware.go`.
- **Validation**: unit-тест проверяет, что middleware для search-стадии читает `StageData.Query`, а для generate — `StageData.Answer`.

### DEC-003 Middleware-цепочка в application-layer

- **Why**: Middleware работает на уровне стадий pipeline, поэтому цепочка живёт в `internal/application/`, а не в `pkg/draftrag/`. Core-pipeline знает о middleware напрямую, публичный слой только пробрасывает опцию. Это повторяет паттерн Hooks и PIIDetector.
- **Tradeoff**: Middleware не видна публичному API (opaque injection). Решение осознанно — middleware не предназначена для runtime-переконфигурации.
- **Affects**: `internal/application/middleware.go`, `pkg/draftrag/draftrag.go`.
- **Validation**: middleware, сконфигурированная через `PipelineOptions`, вызывается внутри application.Pipeline, но не экспортируется из пакета.

### DEC-004 Streaming middleware — обёртка канала

- **Why**: Для `AnswerStream` middleware вызывается pre-stream (может модифицировать запрос) и получает возможность обернуть выходной канал `<-chan string`. Альтернатива (per-token middleware) потребовала бы токенизации на уровне middleware, что нарушает изоляцию concern'ов.
- **Tradeoff**: Middleware не может отфильтровать отдельные токены post-hoc, только целиком ответ (через обёртку канала с аккумуляцией). Для filtering на уровне токенов middleware может накопить полный ответ внутри обёртки и отфильтровать перед закрытием канала.
- **Affects**: `internal/application/stream.go`.
- **Validation**: middleware оборачивает канал, добавляя префикс к первому токену.

## Incremental Delivery

### MVP (Первая ценность)

- Middleware interface in domain
- Middleware chain execution (application)
- Integration in: `Query`, `Answer`, `Index` (produceChunks)
- Integration in: `AnswerStream` (pre-stream only, базовая обёртка)
- PipelineOptions + проброс через public API
- Unit-тесты: AC-001, AC-002, AC-005
- Example: `examples/middleware/main.go` с двумя middleware (логгер + PII-цензор)
- `data-model.md` stub

**Критерий готовности MVP:** `go test ./internal/application/ -run TestMiddleware -v` pass + `go run ./examples/middleware` показывают порядок и модификацию.

### Итеративное расширение

1. **Итерация 2**: Интеграция во все оставшиеся методы (QueryHyDE, QueryMulti, QueryWithParentIDs, QueryWithMetadataFilter, QueryHybrid, AnswerWithCitations, Answer*Filter, AnswerHyDE, AnswerMulti, AnswerHybrid и т.д.). Покрытие AC-003, AC-004.
2. **Итерация 3**: Panic recovery в middleware, тесты краевых случаев (cancel ctx, short-circuit). Обёртка канала для streaming (full post-stream обработка).
3. **Итерация 4**: Performance benchmark (SC-001), покрытие > 85% (SC-002).

## Порядок реализации

1. `internal/domain/middleware.go` — интерфейс + типы (зависимость для всего).
2. `internal/application/middleware.go` — цепочка (зависит от domain).
3. `internal/application/pipeline.go` — опция + поле.
4. `internal/application/hooks.go` или `pipeline.go` — `produceChunks` (chunking/embed middleware).
5. `internal/application/query.go` — Query, QueryWith*, QueryHy*, QueryMulti (search/embed middleware).
6. `internal/application/answer.go` — Answer, AnswerWith* (search/generate middleware).
7. `internal/application/stream.go` — stream middleware.
8. `pkg/draftrag/draftrag.go` — re-export + проброс.
9. Unit-тесты в `internal/application/`.
10. `examples/middleware/main.go`.

**Параллельно**: 1+2 (дизайн), 9+10 (тесты с примером после 8).

## Риски

1. **Риск: Middleware увеличивает сложность pipeline методов**
   - Mitigation: Middleware вызывается через единую функцию `runMiddleware`, не дублируется в каждом методе. Hooks-вызовы служат шаблоном — middleware добавляется рядом.
2. **Риск: StageData будет расти с новыми стадиями**
   - Mitigation: StageData — внутренний тип application-слоя, не экспортируется. Расширение не ломает публичный API.
3. **Риск: Производительность — аллокация StageData на каждый stage**
   - Mitigation: StageData создаётся только при наличии middleware (nil-check). 3 no-op middleware добавит < 5% latency (SC-001 target).

## Rollout and compatibility

- Специальных rollout-действий не требуется — additive change.
- Старое поведение: nil middleware = прежний код. Новое поведение: заданная цепочка выполняется.
- CI: `go vet`, `go fmt`, `golangci-lint` без новых ошибок.

## Проверка

- Unit-тесты:
  - `internal/application/middleware_test.go` — цепочка, порядок, ошибка, empty, short-circuit, panic recovery
  - `internal/application/pipeline_middleware_test.go` — интеграция в каждый метод (список стадий, модификация)
  - `pkg/draftrag/pipeline_test.go` — интеграция через публичный API
- `examples/middleware/main.go` — manual run для визуальной проверки
- AC-001–AC-005 покрываются unit-тестами
- DEC-001–DEC-004 подтверждаются compile-time + unit-тестами

## Соответствие конституции

- нет конфликтов: интерфейсы (Go), контекст, clean architecture (domain → application), простота > расширяемость
