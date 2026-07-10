# Query Rewriting — План

## Phase Contract

Inputs: spec docs/specs/query-rewriting/spec.md, inspect docs/specs/query-rewriting/inspect.md (concerns — Warnings исправлены).
Outputs: plan.md, data-model.md.
Stop if: нет — spec достаточна.

## Цель

Добавить плагируемый `QueryRewriter` в domain layer, интегрировать в PipelineOptions и SearchBuilder, реализовать LLMRewriter как встроенную стратегию. Архитектурный подход следует паттерну `Reranker` — optional component через интерфейс в domain, подключение через опции.

## MVP Slice

- Интерфейс `domain.QueryRewriter` + модели `RewrittenQuery` / `QueryHistory`
- Интеграция в `PipelineOptions` и `SearchBuilder.Rewriter(r)`
- Routing: новый маршрут `routeRewriter`, отключающий HyDE/MultiQuery при установке
- Fallback при ошибке rewriter (логирование + исходный запрос)
- LLMRewriter: базовая реализация через LLMProvider
- AC-001, AC-002, AC-003, AC-004, AC-005, AC-006, AC-007

## First Validation Path

1. Написать тест с mock-rewriter, возвращающим одну и три переформулировки.
2. Вызвать `pipeline.Search("q").Rewriter(mock).Retrieve(ctx)` — убедиться, что retrieval использует переписанные запросы.
3. Для LLMRewriter — интеграционный тест с Ollama LLM: проверить, что генерация переформулировок работает.
4. `go test ./...` без регрессий.

## Scope

- `internal/domain/interfaces.go` — новый интерфейс
- `internal/domain/models.go` — новые value objects
- `internal/infrastructure/rewriter/` — новая директория, реализация LLMRewriter
- `pkg/draftrag/draftrag.go` — PipelineOptions
- `pkg/draftrag/search.go` — SearchBuilder.Rewriter(r)
- `pkg/draftrag/search_routing.go` — новый route
- `internal/application/` — переиспользование RRF без изменений
- VectorStore, Embedder, LLMProvider, Chunker — НЕ меняются

## Performance Budget

`none` — rewriting добавляет LLM-вызов (latency = latency LLM), pipeline не вводит дополнительных аллокаций на критическом пути кроме самих переформулировок. P99 для путей без rewriter не меняется.

## Implementation Surfaces

| Surface | Статус | Зачем |
|---|---|---|
| `internal/domain/interfaces.go` | изменяется | добавить `QueryRewriter` |
| `internal/domain/models.go` | изменяется | добавить `RewrittenQuery`, `QueryHistory` |
| `internal/infrastructure/rewriter/llm_rewriter.go` | новая | LLMRewriter реализация |
| `internal/infrastructure/rewriter/llm_rewriter_test.go` | новая | тесты LLMRewriter |
| `pkg/draftrag/draftrag.go` | изменяется | `PipelineOptions.QueryRewriter` |
| `pkg/draftrag/search.go` | изменяется | `SearchBuilder.Rewriter(r)` |
| `pkg/draftrag/search_routing.go` | изменяется | новый маршрут + handler |
| `internal/application/` | не меняется | RRF fusion переиспользуется как есть |

## Bootstrapping Surfaces

- `internal/infrastructure/rewriter/` — создать директорию с пакетом

## Влияние на архитектуру

- Новый optional component в pipeline, аналогичный `Reranker`.
- Никаких изменений в ядре pipeline (VectorStore/LLM/Embedder).
- Caller управляет историей — pipeline не хранит состояние.
- Существующие HyDE/MultiQuery остаются как built-in флаги, но при установленном Rewriter игнорируются с предупреждением.

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|---|---|---|---|
| AC-001 | Unit-тест: type-assert кастомной структуры в `domain.QueryRewriter` | `interfaces.go` | `go build ./...` + assert |
| AC-002 | Unit-тест: два rewriter'а через PipelineOptions и per-request | `search.go`, `draftrag.go` | разные retrieval результаты |
| AC-003 | Unit-тест: mock возвращает 3 переформулировки, RRF fusion | `search_routing.go` | объединённые результаты, без дублей |
| AC-004 | Unit-тест: LLMRewriter с history проверяет использование контекста | `llm_rewriter.go` | переформулировка включает контекст |
| AC-005 | Unit-тест: error-rewriter, fallback на исходный запрос | `search_routing.go` | успешный retrieval + лог ошибки |
| AC-006 | Integration-тест: LLMRewriter с Ollama | `llm_rewriter.go` | непустые переформулировки |
| AC-007 | Unit-тест: Rewriter + HyDE/MultiQuery флаги — warning, игнор флагов | `search_routing.go` | warning в hooks + retrieval от rewriter |

## Данные и контракты

- Изменения data model: `RewrittenQuery` (Query string + Weight float64), `QueryHistory` ([]Message{Role, Content}).
- Никаких изменений API-контрактов (публичные типы draftrag).
- `data-model.md` — статус `changed`.

## Стратегия реализации

### DEC-001 QueryRewriter в domain, а не в pkg/draftrag

Why: все внешние интерфейсы (VectorStore, Embedder, LLMProvider, Chunker, Reranker) определены в domain. QueryRewriter следует тому же паттерну — единый контракт для всех стратегий.
Tradeoff: кастомные реализации должны импортировать `internal/domain` или использовать пакет draftrag (который реэкспортирует).
Affects: `internal/domain/interfaces.go`
Validation: go build проходит; пример кастомного rewriter'a в тесте работает.

### DEC-002 RewrittenQuery.Weight зарезервирован, но не используется в MVP

Why: future-proofing для weighted fusion (например, доверять LLM-based переформулировке больше, чем rule-based). Пока все weights = 1.0.
Tradeoff: небольшой оверхед поля в структуре.
Affects: `internal/domain/models.go`
Validation: Weight не влияет на fusion в MVP; тест проверяет, что все weights = 1.0 по умолчанию.

### DEC-003 Caller управляет QueryHistory

Why: pipeline не должен управлять состоянием диалога — это ответственность вызывающей стороны. История не хранится, не кэшируется.
Tradeoff: caller должен собирать и обрезать историю; pipeline не может автоматически выводить контекст.
Affects: никаких изменений в pipeline state.
Validation: AC-004.

### DEC-004 RouteRewriter выше HyDE/MultiQuery в pickRoute

Why: если rewriter установлен, built-in флаги должны игнорироваться — иначе двойное переписывание. Приоритет: проверка Rewriter первой.
Tradeoff: HyDE/MultiQuery не работают одновременно с кастомным rewriter'ом.
Affects: `search_routing.go::pickRoute`
Validation: AC-007.

### DEC-005 LLMRewriter в отдельном infrastructure-пакете

Why: следует паттерну `internal/infrastructure/llm/`, `internal/infrastructure/embedder/` — реализация отделена от интерфейса.
Tradeoff: дополнительная директория.
Affects: `internal/infrastructure/rewriter/`
Validation: AC-006.

## Incremental Delivery

### MVP (Первая ценность)

- Интерфейс + модели (`domain/interfaces.go`, `domain/models.go`, `data-model.md`)
- PipelineOptions.QueryRewriter + SearchBuilder.Rewriter(r)
- routeRewriter + fallback + warning при HyDE/MultiQuery
- AC-001, AC-002, AC-005, AC-007

### Итеративное расширение

1. **1:N fusion**: поддержка нескольких переформулировок через RRF — AC-003
2. **LLMRewriter**: базовая реализация через LLMProvider — AC-006
3. **Multi-turn**: передача QueryHistory в Rewrite — AC-004

## Порядок реализации

1. Domain: интерфейс + модели (основа для всего)
2. PipelineOptions + SearchBuilder: API для подключения
3. Routing + fallback: search_routing.go
4. LLMRewriter: infrastructure реализация
5. Тесты: unit + integration

Параллельно: 1+2 можно делать вместе; 4 может начинаться после 1.

## Риски

- **Риск 1**: LLMRewriter зависит от LLM — при ошибке LLM падает производительность retrieval (fallback спасает, но recall снижается).
  Mitigation: fallback на исходный запрос + логирование (AC-005).
- **Риск 2**: Конфликт между Rewriter и существующими HyDE/MultiQuery — пользователь может случайно включить оба.
  Mitigation: явный приоритет Rewriter + warning (AC-007).
- **Риск 3**: QueryHistory может расти бесконтрольно.
  Mitigation: caller отвечает за обрезку; документация рекомендует лимит.

## Rollout и compatibility

- Полностью обратно совместимо: по умолчанию QueryRewriter = nil, поведение не меняется.
- Никаких feature flags или миграций не требуется.
- Старые HyDE/MultiQuery продолжают работать без изменений.

## Проверка

- Unit-тесты для каждого AC (7 тестов, минимум).
- Integration-тест для LLMRewriter с Ollama (AC-006).
- `go vet ./...`, `golangci-lint`, `go test ./...` без ошибок.
- Ручная проверка: пример в `examples/` с кастомным rewriter'ом (опционально).

## Соответствие конституции

- Нет конфликтов.
- Clean Architecture соблюдена: интерфейс в domain, реализация в infrastructure, интеграция в application (PipelineOptions) и public API (SearchBuilder).
- Context.Context во всех публичных операциях — соблюдено.
- Language policy: docs/agent/comments — русский, соблюдено.
