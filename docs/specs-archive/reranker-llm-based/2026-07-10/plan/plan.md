# LLM-as-judge Reranker — План

## Phase Contract

Inputs: spec, inspect (pass), repo-контекст (domain.Reranker, domain.BatchReranker, pipeline wiring).
Outputs: plan.md, data-model.md.
Stop if: spec расплывчата — нет, spec стабильна.

## Цель

Реализовать LLMReranker как новый компонент `internal/infrastructure/reranker/` с публичным конструктором в `pkg/draftrag/reranker_llm.go`. Reranker использует существующий `domain.LLMProvider` для zero-shot скоринга чанков батчем, сортирует по убыванию score, поддерживает retry и graceful degradation. Подключается через существующий `PipelineOptions.Reranker`.

## MVP Slice

Базовый LLMReranker: скоринг всех чанков одним LLM-вызовом, сортировка по score, graceful degradation при ошибке. Обязательные AC: AC-001, AC-002, AC-004, AC-005.

## First Validation Path

Создать in-memory Pipeline с mock LLMProvider, передать 3 чанка с разной релевантностью, проверить что после Query чанки отсортированы по LLM-score убыванию.

## Scope

- `internal/infrastructure/reranker/llm_reranker.go` — реализация Reranker + BatchReranker
- `pkg/draftrag/reranker_llm.go` — `NewLLMReranker`, `LLMReranker`, `LLMRerankerOptions`
- `internal/infrastructure/reranker/llm_reranker_test.go` — unit-тесты
- Pipeline wiring — не меняется, `PipelineOptions.Reranker` уже существует

## Performance Budget

- SC-001: <500ms дополнительной задержки на batch из 10 чанков (быстрая локальная модель). Для remote API latency определяется LLMProvider.
- `alloc/op`: не более 1 KB на чанк сверх входных данных (парсинг JSON, временные структуры).

## Implementation Surfaces

| Surface | Статус | Почему |
|---------|--------|--------|
| `internal/infrastructure/reranker/llm_reranker.go` | новая | Основная реализация, следует precedent `internal/infrastructure/rewriter/` |
| `pkg/draftrag/reranker_llm.go` | новая | Публичный API: конструктор + опции |
| `pkg/draftrag/draftrag.go` | не меняется | Reranker уже в PipelineOptions |
| `internal/domain/interfaces.go` | не меняется | Reranker + BatchReranker уже есть |
| `internal/application/retrieval.go` | не меняется | maybeRerank/maybeRerankBatch уже работают |

## Bootstrapping Surfaces

- `internal/infrastructure/reranker/` — создать директорию
- `pkg/draftrag/reranker_llm.go` — файл уже не существует, создать

## Влияние на архитектуру

- Локальное: новый пакет `internal/infrastructure/reranker/`, изолирован, следует Clean Architecture.
- Интеграции: не требует изменений в существующих интерфейсах или store/LLM реализациях.
- Migration/rollout: не требуется — opt-in через `PipelineOptions.Reranker`.

## Acceptance Approach

- AC-001: mock LLMProvider, проверить что Score у чанков после Rerank установлен.
- AC-002: mock LLMProvider с предсказуемым выводом, проверить порядок.
- AC-003: mock LLMProvider, захватить systemPrompt из Generate, проверить кастомный текст.
- AC-004: mock LLMProvider с ошибкой, проверить что вернулись исходные чанки без ошибки.
- AC-005: mock LLMProvider со счётчиком, batchSize=10, N=5 → 1 вызов.
- AC-006: type assertion + RerankBatch с 2 query + mock.
- AC-007: mock LLMProvider с ошибкой 2 раза, затем success, maxRetries=2.

## Данные и контракты

- Data model не меняется: `domain.RetrievedChunk`, `domain.Reranker`, `domain.BatchReranker` — существующие.
- API/event contracts не меняются: новый компонент не добавляет публичных методов на Pipeline.
- `data-model.md`: stub no-change.

## Стратегия реализации

- DEC-001 Новый пакет `internal/infrastructure/reranker/`
  Why: следует precedent `rewriter/` — infrastructure-реализация интерфейса domain.Reranker.
  Tradeoff: ещё одна директория, но чистая изоляция по Clean Architecture.
  Affects: `internal/infrastructure/reranker/llm_reranker.go`, `pkg/draftrag/reranker_llm.go`.
  Validation: package компилируется, тесты проходят.

- DEC-002 Batch-скоринг: все чанки одним LLM-вызовом, JSON-массив [0..10]
  Why: минимум LLM-вызовов, JSON парсится штатно, шкала 0–10 integer надёжнее float.
  Tradeoff: размер промпта растёт с batchSize; при превышении контекстного окна chunk обрезается.
  Affects: `llm_reranker.go` — формирование промпта, парсинг ответа.
  Validation: AC-005 (1 вызов на batch), AC-001 (score присвоен).

- DEC-003 Retry → graceful degradation
  Why: временные ошибки не должны терять ранжирование; исчерпание retry → исходный порядок.
  Tradeoff: retry увеличивает latency при сбоях; graceful degradation скрывает ошибку от caller.
  Affects: `llm_reranker.go` — retry loop, fallback path.
  Validation: AC-007 (retry success), AC-004 (graceful после исчерпания).

- DEC-004 BatchReranker — отдельный метод с общей логикой скоринга
  Why: переиспользовать скоринг, не дублировать; pipeline проверяет capability через type assertion.
  Tradeoff: чуть больше кода, чем fallback на single-query в maybeRerankBatch.
  Affects: `llm_reranker.go` — scoreChunks helper, Rerank и RerankBatch поверх него.
  Validation: AC-006 (type assertion + batch rerank).

## Incremental Delivery

### MVP (Первая ценность)

- `internal/infrastructure/reranker/llm_reranker.go`: Reranker + scoreChunks + retry + graceful degradation.
- `pkg/draftrag/reranker_llm.go`: NewLLMReranker, опции (BatchSize, PromptTemplate, MaxRetries).
- Tests: AC-001, AC-002, AC-004, AC-005, AC-007.
- Критерий: `pipeline.Search("q").TopK(5).Retrieve(ctx)` возвращает чанки с LLM-score.

### Итеративное расширение

- Шаг 2: AC-003 (кастомный prompt template) + AC-006 (BatchReranker).
- Шаг 3: Дополнительные краевые случаи (пустой список, непарсимый ответ, score=0).

## Порядок реализации

1. Создать `internal/infrastructure/reranker/llm_reranker.go` — core scoring.
2. Создать `pkg/draftrag/reranker_llm.go` — public API.
3. Написать тесты (AC-001–AC-007).
4. Проверить интеграцию через example или smoke test.

Параллельно: нет зависимостей между файлами одного пакета.

## Риски

- Риск: LLM не следует формату JSON-массива в ответе
  Mitigation: fallback на score=0 с логированием; тест AC-001 покрывает корректный парсинг.

- Риск: LLM latency > 500ms (SC-001)
  Mitigation: target для быстрой локальной модели (Ollama, small model); SC не является hard requirement.

- Риск: Контекстное окно LLM переполнено при большом batchSize
  Mitigation: batchSize конфигурируем (default 10); ответственность пользователя.

## Rollout и compatibility

- Специальных rollout-действий не требуется: компонент opt-in, не ломает существующий API.
- Monitoring/logging: ошибки LLM логируются через стандартный логгер (если передан).

## Проверка

- Unit-тесты: `internal/infrastructure/reranker/llm_reranker_test.go` — mock LLMProvider покрывает все AC.
- Проверка конституции: Clean Architecture, Go 1.21+, context.Context, интерфейсы — все соблюдены.
- `go vet`, `go test ./internal/infrastructure/reranker/...` — без ошибок.

## Соответствие конституции

- нет конфликтов
