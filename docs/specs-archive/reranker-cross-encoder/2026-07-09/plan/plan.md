# Reranker: cross-encoder и Cohere Rerank — План

## Phase Contract

Inputs: `spec.md`, `inspect.md` (pass).
Outputs: `plan.md`, `data-model.md`.
Stop if: spec расплывчата — нет, spec конкретна.

## Цель

Добавить production-ready reranker реализации Cohere Rerank API (P1/MVP) и LLM-based (P2) в новый пакет `pkg/draftrag/reranker/`. Интерфейс `domain.Reranker` уже существует — реализации нет. Batch-режим через опциональный `BatchReranker` для MultiQuery оптимизации.

## MVP Slice

Cohere Rerank API + `BatchReranker` интерфейс + pipeline-интеграция. LLM-reranker — P2.

Покрывает: AC-001, AC-002, AC-003, AC-006, AC-007, AC-008, AC-009, AC-010.
Отложено: AC-004, AC-005 (LLM-reranker, P2).

## First Validation Path

```go
p, _ := NewPipeline(store, llm, embed)
cr, _ := reranker.NewCohereRerank(reranker.CohereRerankOptions{
    APIKey:  os.Getenv("COHERE_API_KEY"),
    Timeout: 30 * time.Second,
})
p, _ = NewPipelineWithOptions(store, llm, embed, PipelineOptions{Reranker: cr})
result, _ := p.Search("query").TopK(10).Retrieve(ctx)
// result.Chunks порядок изменён относительно embedding similarity
```

## Scope

- Новый пакет: `pkg/draftrag/reranker/`
- Новая опциональная capability: `domain.BatchReranker` в `internal/domain/interfaces.go`
- `CohereReranker` — HTTP-клиент к `POST /v2/rerank`
- Pipeline-интеграция: `maybeRerank` → `maybeRerankBatch` для MultiQuery
- Тесты: unit + mock HTTP-сервер для Cohere
- Документация: `docs/en/reranker.md`, `docs/ru/reranker.md`
- **Не меняется**: существующие store-реализации, `SearchBuilder`, `PipelineOptions` (поле `Reranker` уже есть)

## Performance Budget

- P95 Cohere Rerank для 20 чанков: ≤ 500ms (при сетевой задержке ≤ 100ms)
- Batch (5 query, 20 чанков): P95 ≤ 600ms (concurrent fan-out, не последовательно)
- Без reranker'а: zero overhead (nil-guard, одна проверка)

## Implementation Surfaces

### Новые

| Surface | Почему новая |
|---|---|
| `pkg/draftrag/reranker/reranker.go` | Публичный пакет для reranker'ов. Изоляция HTTP-клиента Cohere |
| `pkg/draftrag/reranker/cohere.go` | `CohereReranker` + `CohereRerankOptions` |
| `pkg/draftrag/reranker/errors.go` | `ErrInvalidRerankerConfig` (и reuse через `fmt.Errorf("%w: detail")`) |
| `docs/en/reranker.md` | English документация |
| `docs/ru/reranker.md` | Russian документация |

### Существующие (модификация)

| Surface | Изменение |
|---|---|
| `internal/domain/interfaces.go:86-88` | Добавить `BatchReranker` интерфейс |
| `internal/application/retrieval.go:12-22` | Добавить `maybeRerankBatch` |
| `internal/application/query.go:110-117` | В `QueryMulti`: type-assert на `BatchReranker` → `RerankBatch` |
| `pkg/draftrag/draftrag.go` | Re-export `BatchReranker` |
| `pkg/draftrag/reranker_test.go` | Уже есть reverseReranker mock — обновить для batch |

## Bootstrapping Surfaces

```
pkg/draftrag/reranker/
├── reranker.go    — re-exports, public API
├── cohere.go      — CohereReranker implementation
└── errors.go      — sentinel errors
```

## Влияние на архитектуру

- Локальное: новый пакет не создаёт циклических зависимостей (зависит от `domain` и `net/http`)
- Совместимость: `PipelineOptions.Reranker` остаётся `nil`-совместимым. Все существующие pipeline без reranker'а работают без изменений
- BatchReranker — опциональная capability: type-assert, не ломает существующие реализации Reranker
- `pkg/draftrag/reranker/` — публичный пакет без внутренних под-пакетов

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|---|---|---|---|
| AC-001 | Cohere HTTP-клиент + integration test с mock-сервером | `cohere.go` | Порядок чанков изменён |
| AC-002 | Guard: empty chunks → no-op | `cohere.go` | `len(out)==0, err==nil` |
| AC-003 | Constructor validation: пустой ключ → sentinel | `cohere.go`, `errors.go` | `errors.Is(err, ErrInvalidRerankerConfig)` |
| AC-006 | HTTP 401 → error | `cohere.go` | `err != nil` |
| AC-007 | Invariant: len(out) == len(in) | `cohere.go`, `llm.go` | Unit test циклом |
| AC-008 | errgroup fan-out в BatchReranker | `cohere.go`, `retrieval.go` | Время < 150ms для 5×100ms |
| AC-009 | Fallback: нет BatchReranker → последовательно | `query.go`, `retrieval.go` | Mock считает вызовы `Rerank` |
| AC-010 | Документация с кодом | `docs/*/reranker.md` | Файлы существуют, содержат код |

## Данные и контракты

- Data model: минимальные изменения (см. `data-model.md`)
- Новый sentinel `ErrInvalidRerankerConfig` — стандартный паттерн `fmt.Errorf("%w: detail", sentinel)`
- `BatchReranker` — новый интерфейс, не ломает существующие контракты
- Cohere API ответ: `results[{index, relevance_score}]` — маппинг на `RetrievedChunk` по `index`

## Стратегия реализации

### DEC-001: Пакет `pkg/draftrag/reranker/`

**Why**: Cohere требует HTTP-клиент с API-ключом. Вынос в отдельный пакет изолирует эту зависимость от основного API. Все остальные store/LLM/embedder фабрики — в `pkg/draftrag/`, но они facade-обёртки. Reranker — первый компонент с собственным HTTP-клиентом.

**Tradeoff**: пользователь импортирует `pkg/draftrag/reranker/` отдельно. Небольшое UX-трение против чистой изоляции.

**Affects**: `pkg/draftrag/reranker/`

**Validation**: `go build ./pkg/draftrag/...` без ошибок

### DEC-002: Batch через опциональный интерфейс, не изменение `Reranker`

**Why**: Сохраняет backward compatibility. Существующие кастомные `Reranker` реализации продолжают работать. Паттерн уже используется (`StreamingLLMProvider`, `VectorStoreWithFilters`).

**Tradeoff**: Type assertion в hotspot (QueryMulti). На практике — одна проверка на вызов, overhead нулевой.

**Affects**: `internal/domain/interfaces.go`, `internal/application/retrieval.go`, `internal/application/query.go`

**Validation**: AC-009 (fallback)

### DEC-003: Concurrent fan-out для Cohere batch

**Why**: Cohere v2 API не поддерживает массив query — только одиночную строку. Concurrent fan-out (errgroup + N параллельных HTTP-запросов) даёт выигрыш в latency без изменения API.

**Tradeoff**: N одновременных соединений к Cohere. На практике N ≤ 5, rate limit на стороне Cohere.

**Affects**: `pkg/draftrag/reranker/cohere.go`

**Validation**: AC-008 (latency < 150ms)

## Incremental Delivery

### MVP — Cohere Rerank + batch

Задачи:
1. `BatchReranker` интерфейс + `maybeRerankBatch` в pipeline
2. `CohereReranker` (single-query) + валидация + тесты
3. Batch-режим для Cohere (concurrent fan-out) + тесты
4. Pipeline интеграция: QueryMulti → `RerankBatch`
5. Документация + пример

Критерий: AC-001, 002, 003, 006, 007, 008, 009, 010.

### P2 — LLM-reranker

После MVP:
1. `LLMReranker` — zero-shot scoring через `LLMProvider.Generate`
2. Prompt engineering для скоринга релевантности
3. Тесты с mock LLM

Критерий: AC-004, AC-005.

## Порядок реализации

1. **BatchReranker интерфейс** (domain) — без него нельзя писать реализации
2. **Cohere single-query** — основа, от неё пляшет batch
3. **Cohere batch** — errgroup fan-out
4. **Pipeline интеграция** — QueryMulti type-assert
5. **Тесты всех AC** — от простого к сложному
6. **Документация** — после стабильного API

Параллельно: errors, re-exports.

## Риски

| Риск | Mitigation |
|---|---|
| Cohere API rate limit (429) | Exponential backoff через существующий `resilience.RetryConfig`. Добавить `MaxRetries` в `CohereRerankOptions` |
| Cohere API изменит v2 | `BaseURL` кастомизируемый. Версионирование через URL |
| Batch fan-out создаёт N одновременных соединений | N ≤ number of query variants ≤ 5. Контролируемо |
| LLM-reranker P2 может затянуться | Вынесен из MVP. Не блокирует релиз |

## Rollout и compatibility

- Никаких migration/feature flag не требуется
- Reranker — опциональная capability, nil по умолчанию
- Все существующие тесты проходят без изменений
- Добавить `TestPipeline_NoReranker_Works` (уже есть) и `TestPipeline_Reranker_WithBatch`

## Проверка

- `go test ./pkg/draftrag/reranker/...` — unit + mock HTTP
- `go test ./pkg/draftrag/...` — integration с reverseReranker mock
- `go vet ./...`, `golangci-lint run ./...`
- Ручная проверка: `examples/reranker/main.go` с реальным Cohere API ключом

## Соответствие конституции

Нет конфликтов. Чистая архитектура (domain → application → infrastructure → new reranker), context.Context во всех публичных операциях, Go 1.23, никаких встроенных HTTP-серверов.
