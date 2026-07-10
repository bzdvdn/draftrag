# LLM-as-judge Reranker — Задачи

## Phase Contract

Inputs: plan.md, spec.md, data-model.md (no-change).
Outputs: исполнимые задачи с Touches и покрытием AC.
Stop if: задачи расплывчаты — нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/reranker/llm_reranker.go` | T1.1, T2.1, T2.2, T3.1 |
| `pkg/draftrag/reranker_llm.go` | T1.2 |
| `internal/infrastructure/reranker/llm_reranker_test.go` | T1.3, T2.3, T3.2 |
| `internal/domain/interfaces.go` | не меняется |
| `pkg/draftrag/draftrag.go` | не меняется |

## Implementation Context

- Цель MVP: LLMReranker, реализующий domain.Reranker, с batch-скорингом через LLM, graceful degradation и конфигурируемым промптом.
- Инварианты/семантика:
  - Score: LLM возвращает 0–10 integer JSON → нормализация /10 в RetrievedChunk.Score
  - Batch: все чанки в одном LLM-вызове; LLM-вызовов = ceil(N / BatchSize)
  - Retry (MaxRetries): retry при временной ошибке, затем graceful degradation
  - Graceful degradation: ошибка LLM → исходные чанки (score и порядок не меняются), error не возвращается
  - Пустой список: без LLM-вызова, возвращается как есть
  - Непарсимый ответ LLM: score=0, логируется, не прерывает rerank
- Контракты/протокол:
  - `NewLLMReranker(llm domain.LLMProvider, opts ...LLMRerankerOption) (*LLMReranker, error)`
  - `LLMRerankerOptions`: BatchSize (default 10), PromptTemplate, MaxRetries (default 1)
  - Рекспорт: `LLMReranker`, `NewLLMReranker`, `LLMRerankerOption`, `WithBatchSize`, `WithPromptTemplate`, `WithMaxRetries`
  - UsageAwareLLMProvider: опционально, логировать token usage через Hooks если доступен
- Proof signals: тесты с mock LLMProvider покрывают AC-001–AC-007; ручная проверка через пример
- Вне scope: fusion (LLM-score + исходный score), кэширование, cross-encoder

## Фаза 1: Core LLMReranker (MVP)

Цель: реализовать базовый LLMReranker, покрывающий AC-001, AC-002, AC-004, AC-005.

- [x] T1.1 Реализовать `internal/infrastructure/reranker/llm_reranker.go`:
  - Тип `llmReranker` (privated), имплементирующий `domain.Reranker`
  - `scoreChunks(ctx, query, chunks) ([]float64, error)` — batch LLM-вызов, формирование промпта (system + user), парсинг JSON-массива [0..10], нормализация /10
  - `Rerank(ctx, query, chunks)` — вызывает scoreChunks, сортирует по score убыванию
  - Graceful degradation: при ошибке LLM возвращает исходные чанки без ошибки
  - Empty/single chunk guards
  - Учёт контекстной ошибки (`ctx.Err()`)
  Touches: `internal/infrastructure/reranker/llm_reranker.go`

- [x] T1.2 Реализовать `pkg/draftrag/reranker_llm.go`:
  - `NewLLMReranker(llm domain.LLMProvider, opts ...LLMRerankerOption) (*LLMReranker, error)`
  - `LLMReranker` — публичный тип, обёртка над `llmReranker`
  - `LLMRerankerOption` — functional options: `WithBatchSize(n)`, `WithPromptTemplate(tmpl)`, `WithMaxRetries(n)`
  - Дефолты: BatchSize=10, PromptTemplate=defaultJudgePrompt, MaxRetries=1
  - UsageAwareLLMProvider: type-assert и сохранить для опционального usage-логирования
  Touches: `pkg/draftrag/reranker_llm.go`

- [x] T1.3 Написать unit-тесты для MVP ACs:
  - AC-001: mock LLMProvider возвращает JSON scores → проверить что Score установлен
  - AC-002: mock LLMProvider возвращает [0.9, 0.3, 0.7] → проверить порядок
  - AC-004: mock LLMProvider с ошибкой → проверить исходные чанки без ошибки
  - AC-005: mock LLMProvider со счётчиком, N=5, BatchSize=10 → 1 вызов
  Touches: `internal/infrastructure/reranker/llm_reranker_test.go`

## Фаза 2: Extensions

Цель: реализовать оставшиеся AC (AC-003, AC-006, AC-007).

- [x] T2.1 Добавить поддержку кастомного PromptTemplate:
  - `WithPromptTemplate(template string)` — переопределяет defaultJudgePrompt
  - При формировании промпта использовать шаблон пользователя вместо дефолтного
  - AC-003: mock LLMProvider захватывает systemPrompt → проверить кастомный текст
  Touches: `internal/infrastructure/reranker/llm_reranker.go`, `internal/infrastructure/reranker/llm_reranker_test.go`

- [x] T2.2 Реализовать BatchReranker capability:
  - `llmReranker` имплементирует `domain.BatchReranker` (RerankBatch)
  - RerankBatch вызывает scoreChunks для каждого query, переиспользует логику скоринга
  - AC-006: type assertion успешен; тест с 2 query + mock
  Touches: `internal/infrastructure/reranker/llm_reranker.go`, `internal/infrastructure/reranker/llm_reranker_test.go`

- [x] T2.3 Добавить retry logic + тесты:
  - Retry loop в scoreChunks: при ошибке повторять до MaxRetries раз
  - После исчерпания retry — graceful degradation (AC-004)
  - AC-007: mock LLMProvider ошибается 2 раза, success на 3-й → счётчик = 3; исчерпание → graceful degradation
  Touches: `internal/infrastructure/reranker/llm_reranker.go`, `internal/infrastructure/reranker/llm_reranker_test.go`

## Фаза 3: Edge cases + verify readiness

Цель: покрыть краевые случаи и финальная верификация.

- [x] T3.1 Реализовать edge cases в core:
  - Пустой список чанков → без LLM-вызова
  - Один чанк → один LLM-вызов
  - Все score = 0 → исходный порядок
  - Непарсимый ответ LLM → score=0, логируется
  Touches: `internal/infrastructure/reranker/llm_reranker.go`, `internal/infrastructure/reranker/llm_reranker_test.go`

- [x] T3.2 Финальные тесты + verify:
  - go vet, go test ./internal/infrastructure/reranker/...
  - Проверить что public API компилируется (go build ./pkg/draftrag/...)
  - AC-001–AC-007 все покрыты тестами
  Touches: `internal/infrastructure/reranker/llm_reranker_test.go`

## Покрытие критериев приемки

- AC-001 → T1.1, T1.3
- AC-002 → T1.1, T1.3
- AC-003 → T2.1
- AC-004 → T1.1, T1.3
- AC-005 → T1.1, T1.3
- AC-006 → T2.2
- AC-007 → T2.3
