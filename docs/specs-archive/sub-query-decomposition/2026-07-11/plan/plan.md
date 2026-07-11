# Sub-query decomposition План

## Phase Contract

Inputs: spec + inspect (pass) + минимальный repo-контекст.
Outputs: plan, data model, contracts.
Stop if: — нет.

## Цель

Реализовать фичу sub-query decomposition: новый интерфейс `QueryDecomposer`, LLM-based и rule-based реализации, параллельный retrieval по под-вопросам, merge результатов, генерация ответа. Встраивается в существующую routing-архитектуру через `SearchBuilder.SubDecompose()`.

## MVP Slice

LLM-based decomposition + параллельный retrieve + merge + answer. Без rule-based стратегии (P2).

AC-001, AC-002, AC-005 (только LLM→single fallback), AC-007 — обязательны для MVP.

## First Validation Path

Тест: `mockLLM` возвращает `["X requirements?", "Y pricing?"]` → `mockStore` фиксирует вызовы → проверить, что Search вызван 2 раза с разными query и что Answer содержит контекст из обоих retrieve.

## Scope

- Новый интерфейс `QueryDecomposer` в `internal/domain/interfaces.go`
- Composite-декомпозер `NewPipelineWithConfig` в `internal/infrastructure/decomposer/composite.go` (LLM→rule fallback)
- LLM-декомпозер в `internal/infrastructure/decomposer/llm.go`
- Rule-декомпозер в `internal/infrastructure/decomposer/rule.go` (P2, заглушка для MVP)
- Новый файл `internal/application/subdecompose.go` — `QuerySubDecompose` / `AnswerSubDecompose` (и все downstream методы)
- `pkg/draftrag/draftrag.go` — поле `QueryDecomposer` в `PipelineOptions` + поле на `Pipeline`
- `pkg/draftrag/search.go` — `SubDecompose()` + поле `subDecompose bool` на `SearchBuilder`
- `pkg/draftrag/search_routing.go` — `routeSubDecompose`, handler-записи во всех 7 router map
- Новая sentinel `ErrSubDecomposeNotSupported` в `pkg/draftrag/errors.go` (для nil decomposer)
- Existing answer.go — без изменений (использует QueryResult → LLM)

## Performance Budget

- Sub-query retrieval: topK per sub-question = исходный topK, общее latency = max(per-sub-query latency) + overhead декомпозиции + merge. Не медленнее, чем `QueryMulti` для того же количества queries.
- `none` для peak RSS / alloc/op (не критично для этой фичи).

## Implementation Surfaces

| Surface | Почему | Новая/сущ. |
|---|---|---|
| `internal/domain/interfaces.go` | `QueryDecomposer` interface | существующая |
| `internal/infrastructure/decomposer/` | LLM и Rule реализации decomposer'а | новая |
| `internal/application/subdecompose.go` | оркестрация sub-query цикла | новая |
| `pkg/draftrag/draftrag.go` | PipelineOptions + Pipeline поле | существующая |
| `pkg/draftrag/search.go` | `SubDecompose()` method | существующая |
| `pkg/draftrag/search_routing.go` | route + handlers | существующая |
| `pkg/draftrag/errors.go` | sentinel | существующая |

## Bootstrapping Surfaces

- `internal/infrastructure/decomposer/` — создать директорию и базовые файлы

## Влияние на архитектуру

- Локальное: новый optional component (`QueryDecomposer`), новая ветка routing
- Нет миграций/совместимости: все изменения additive (nil = отключено)
- PipelineOptions расширяется на 1 поле (nil-safe)
- `ErrSubDecomposeNotSupported` — новый sentinel

## Acceptance Approach

- **AC-001**: SearchBuilder.SubDecompose() устанавливает флаг → pickRoute возвращает routeSubDecompose → handler вызывает QuerySubDecompose. Наблюдается через mock Store.
- **AC-002**: mockLLM возвращает JSON-массив → handler вызывает embed+search для каждого → проверка call count >= 2.
- **AC-005**: decomposer.LDecompose error → fallback на single-query → store.Search вызван 1 раз с исходным query.
- **AC-007**: goroutine start/end timestamps перекрываются (time overlap) в параллельных Search вызовах.
- **AC-004** (P2): mock возвращает один и тот же chunk по двум sub-queries → merged имеет 1 entry с max score.
- **AC-006** (P2): SubDecompose не вызван → флаг false → routeBasic → single-query.
- **AC-008** (P2): AnswerSubDecompose → LLMProvider получает merged контекст.
- **AC-009** (P2): CiteSubDecompose → возвращает непустые sources.
- **AC-003** (P2): rule-decomposer вызывается без LLM.

## Данные и контракты

- `QueryDecomposer` interface (новый):
  ```go
  type QueryDecomposer interface {
      Decompose(ctx context.Context, query string) ([]string, error)
  }
  ```
- `LLMQueryDecomposer` — impl через LLMProvider + system prompt
- `RuleQueryDecomposer` — impl через rules (союзы/разделители)
- `CompositeDecomposer` — обёртка LLM→Rule→single fallback
- `SearchBuilder.subDecompose bool` — per-request флаг
- `SearchBuilder.SubDecompose() *SearchBuilder` — builder method
- `Pipeline.queryDecomposer QueryDecomposer` — pipeline-level
- `ErrSubDecomposeNotSupported` — новый sentinel
- Изменений data model нет (data-model.md: no-change)

## Стратегия реализации

### DEC-001 Отдельный QueryDecomposer interface (не QueryRewriter)

- Why: sub-query decomposition семантически другой паттерн — под-вопросы не заменяют запрос, а декомпозируют его. QueryRewriter возвращает `[]RewrittenQuery` (может менять запрос), а `QueryDecomposer` возвращает `[]string` (под-вопросы). Разделение интерфейсов проще для понимания и тестирования.
- Tradeoff: больше boilerplate (два похожих, но разных интерфейса), но семантика яснее.
- Affects: `internal/domain/interfaces.go`, `pkg/draftrag/draftrag.go`
- Validation: AC-001, AC-002

### DEC-002 Parallel sub-query retrieval через errgroup

- Why: sub-queries независимы — нет причин выполнять последовательно. `errgroup` с concurrency limit даёт контролируемый параллелизм и корректное распространение cancellation.
- Tradeoff: сложнее, чем последовательный цикл (как в QueryMulti). Но AC-007 требует параллельности.
- Affects: `internal/application/subdecompose.go`
- Validation: AC-007 (time overlap)

### DEC-003 Merge по dedup + max score (не RRF)

- Why: под-вопросы покрывают разные аспекты — применение RRF (как в QueryMulti) снизило бы score разных facet. Dedup по Chunk.ID + max score сохраняет лучший результат для каждого уникального чанка.
- Tradeoff: без RRF порядок merged результатов менее «сглаженный», но sub-query merge не про ранжирование, а про объединение контекста.
- Affects: `internal/application/subdecompose.go`
- Validation: AC-004

### DEC-004 CompositeDecomposer с LLM→Rule fallback

- Why: spec требует graceful degradation. Если LLM недоступен или возвращает ошибку — fallback на rule-based. Rule-based не требует LLM и работает детерминированно.
- Tradeoff: composite добавляет слой косвенности. Для MVP rule-based может быть заглушкой (возвращает nil → single fallback).
- Affects: `internal/infrastructure/decomposer/composite.go`
- Validation: AC-005

### DEC-005 New routeSubDecompose в routing (не как флаг внутри routeRewriter)

- Why: под-вопросы должны эмбеддиться отдельно и искаться отдельно, а затем merge — это другой handler, не rewriter. Собственный route даёт чистую реализацию без усложнения rewriter-пути.
- Tradeoff: новый route = новая запись в 7 router map + 7 handler-функций = дублирование паттерна.
- Affects: `pkg/draftrag/search_routing.go`
- Validation: AC-001, AC-009

## Incremental Delivery

### MVP (Первая ценность)

- AC-001: SubDecompose на SearchBuilder + route + merge
- AC-002: LLM-based decomposer (базовый, JSON-парсинг)
- AC-005: graceful degradation LLM→single-query
- AC-007: параллельный retrieval

Критерий: `Search("сложный запрос").SubDecompose().Answer(ctx)` возвращает ответ, основанный на нескольких retrieve-вызовах.

### Итеративное расширение

- Шаг 2 (P2, rule-based): AC-003, AC-005 полный composite chain
- Шаг 3 (P2, merge): AC-004
- Шаг 4 (P2, per-request override): AC-006
- Шаг 5 (P2, answer + cite): AC-008, AC-009

## Порядок реализации

1. Domain interface `QueryDecomposer` + sentinel `ErrSubDecomposeNotSupported`
2. `SearchBuilder.SubDecompose()` + `PipelineOptions.QueryDecomposer` + routing
3. `internal/application/subdecompose.go` — `QuerySubDecompose` (parallel + merge)
4. `pkg/draftrag/search_routing.go` — routeSubDecompose handlers (Retrieve, Answer, Cite, InlineCite, Stream)
5. `internal/infrastructure/decomposer/llm.go` — LLMQueryDecomposer
6. `internal/infrastructure/decomposer/rule.go` + `composite.go` — P2
7. Answer/Cite/... handler-функции в routing (остальные после MVP)

Шаги 1–5 можно параллелить ограниченно: interface → SearchBuilder → routing → decomposer → application logic. Application logic зависит от interface и routing.

## Риски

- **JSON-парсинг LLM вывода**: LLM может вернуть невалидный JSON → fallback на rule/single. Mitigation: AC-005 уже покрывает graceful degradation. Добавить tolerant parser (regex fallback).
- **Latency**: параллельный поиск по N sub-queries может быть медленнее single-query при низком topK. Mitigation: concurrency limit + reuse worker pool из IndexBatch.
- **Конфликт с MultiQuery/HyDE**: интуитивно sub-decompose комбинируем с rewriter, но не с HyDE/MultiQuery (семантически). Mitigation: `pickRoute` приоритет: если `subDecompose` + conflict → ошибка или игнор sub-decompose.

## Rollout и compatibility

- Новый код полностью additive (nil decomposer = поведение не меняется)
- feature branch → merge → без флагов
- Специальных rollout-действий не требуется

## Проверка

- Unit-тесты: `TestPipeline_QuerySubDecompose*` (mock LLM, mock Store, проверка параллельности через goroutine tracking)
- Unit-тесты: `TestLLMQueryDecomposer*` (валидный/невалидный JSON, ошибка LLM)
- Unit-тесты: `TestCompositeDecomposer*` (LLM→rule fallback chain)
- Lint: `go vet`, `golangci-lint` без ошибок
- Manual: пример в `examples/` (post-MVP)

## Соответствие конституции

- нет конфликтов: все внешние зависимости через Go-интерфейсы, Clean Architecture соблюдена, context.Context во всех публичных операциях, unit-тесты для всех новых функций.
