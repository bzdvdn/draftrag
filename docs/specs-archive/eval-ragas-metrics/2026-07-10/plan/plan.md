# RAGAS-style evaluation metrics — План

## Phase Contract

Inputs: spec и минимальный repo-контекст.
Outputs: plan, data model.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Добавить три RAGAS-style метрики (Faithfulness, Answer Relevance, Context Relevance) в eval-пакет draftRAG как набор независимых функций + интеграция в отчёт `Metrics`. Без изменения существующего `Run()` — новый `RunWithAnswer()`. Все метрики опциональны, graceful degradation при nil-провайдерах.

## MVP Slice

- **Инкремент 1**: три standalone-функции `ComputeFaithfulness`, `ComputeAnswerRelevance`, `ComputeContextRelevance` в `pkg/draftrag/eval/` — покрывают AC-001, AC-002, AC-003, AC-005, AC-006.
- **Инкремент 2**: интеграция в `Metrics` + `RunWithAnswer` — покрывает AC-004.
- Расширение `Case` полем `ExpectedAnswer` (опционально).

## First Validation Path

`go test ./pkg/draftrag/eval/` с mock LLMProvider/Embedder:
- Faithfulness: mock возвращает JSON с подтверждёнными/неподтверждёнными claims
- Answer Relevance: mock Embedder возвращает vectors с известным косинусным расстоянием
- Context Relevance: аналогично Answer Relevance

## Scope

- Новый файл `pkg/draftrag/eval/ragas.go` — три функции-метрики + internal helpers.
- Новый файл `pkg/draftrag/eval/ragas_test.go` — unit-тесты.
- Изменение `pkg/draftrag/eval/models.go` — новые поля в `Metrics`, `Options`, `Case`.
- Изменение `pkg/draftrag/eval/harness.go` — новая функция `RunWithAnswer`.
- Изменение `INSPECT.md` — не меняется, артефакт inspect.
- `internal/domain/` — не меняется (используем существующие `LLMProvider` и `Embedder`).

## Performance Budget

- Faithfulness: 1 LLM-вызов на кейс.
- Answer Relevance: 1 LLM-вызов (генерация N вопросов) + N+1 Embedder-вызовов.
- Context Relevance: N+1 Embedder-вызовов (вопрос + каждый чанк).
- **none** — eval офлайн, performance не критичен.

## Implementation Surfaces

1. `pkg/draftrag/eval/ragas.go` (новая) — `ComputeFaithfulness`, `ComputeAnswerRelevance`, `ComputeContextRelevance`, prompt templates, internal `cosineSimilarity`, `generateQuestions`.
2. `pkg/draftrag/eval/ragas_test.go` (новая) — тесты всех трёх метрик с mock.
3. `pkg/draftrag/eval/models.go` (существующая) — новые поля в `Metrics`, `Options`, `Case`.
4. `pkg/draftrag/eval/harness.go` (существующая) — `RunWithAnswer(ctx, runner, llm, embedder, cases, opts)`.
5. `pkg/draftrag/eval/metrics.go` (существующая) — расширение `computeMetrics`.

## Bootstrapping Surfaces

- `none` — нужная структура (`pkg/draftrag/eval/`) уже существует.

## Влияние на архитектуру

- Новые типы не вводятся (кроме полей структур).
- `LLMProvider` и `Embedder` используются через публичные type aliases из `pkg/draftrag`.
- Eval-пакет остаётся в `pkg/draftrag/eval/` без новых зависимостей.
- Обратная совместимость: `Run` не меняется; `Options` получает новые bool-поля (zero value = false, семантика preserve).

## Acceptance Approach

- AC-001: `ComputeFaithfulness` с mock LLM, возвращающим structured JSON score. Проверка score = 1.0 / < 1.0.
- AC-002: `ComputeAnswerRelevance` с mock Embedder. Проверка score выше для прямого ответа.
- AC-003: `ComputeContextRelevance` с mock Embedder. Проверка score = 1.0 для всех релевантных чанков.
- AC-004: `RunWithAnswer` + mock Runner/LLM/Embedder. Проверка `Metrics.Faithfulness != 0`.
- AC-005: Вызов с nil LLMProvider/Embedder. Проверка score = 0, err = nil.
- AC-006: `ComputeFaithfulness` с пустым answer. Проверка score = 0, err = nil.

## Данные и контракты

- `data-model.md` — описывает новые поля в `Metrics`, `Options`, `Case`. Никаких новых domain-типов.

## Стратегия реализации

### DEC-001 Standalone functions вместо RAGASEvaluator struct

- Why: три метрики имеют разные сигнатуры (разные зависимости), объединение в struct добавит лишнюю косвенность без пользы. Go-идиоматичнее экспортировать функции.
- Tradeoff: пользователь вызывает три функции вручную при независимом использовании; `RunWithAnswer` — это orchestration wrapper.
- Affects: `pkg/draftrag/eval/ragas.go`
- Validation: тесты вызывают функции напрямую.

### DEC-002 Faithfulness через один LLM-вызов с JSON prompt

- Why: spec допускает chain-of-thought на одном вызове. Два вызова (декомпозиция + верификация) удваивает latency и cost без существенного выигрыша в точности для Go-библиотеки.
- Tradeoff: меньше контроля над промежуточным шагом декомпозиции; более сложный prompt.
- Affects: `pkg/draftrag/eval/ragas.go` — prompt template
- Validation: mock LLM возвращает корректный JSON → score корректный.

### DEC-003 Answer Relevance через Embedder (не LLM)

- Why: spec допускает Embedder. LLM-based оценка релевантности дороже (N LLM-вызовов) и нестабильнее. Embedder + cosine similarity детерминирован и значительно дешевле.
- Tradeoff: требуется Embedder; семантическая близость не всегда совпадает с человеческой оценкой релевантности.
- Affects: `pkg/draftrag/eval/ragas.go` — `ComputeAnswerRelevance`
- Validation: тест с известными эмбеддингами.

### DEC-004 RunWithAnswer — новая функция, не расширение Run

- Why: `Run` принимает только `RetrievalRunner` (без LLM/Embedder). Добавление опциональных LLM/Embedder в `Run` сломает сигнатуру и усложнит код. `RunWithAnswer` — новый экспорт, сохраняющий полную обратную совместимость.
- Tradeoff: дублирование логики итерации по кейсам. Альтернатива — internal `runInternal` — не оправдана для 20 строк.
- Affects: `pkg/draftrag/eval/harness.go`
- Validation: `RunWithAnswer` возвращает Report с RAGAS-метриками.

## Incremental Delivery

### MVP (Инкремент 1: standalone функции)

- `ComputeFaithfulness(ctx, answer, context, llmProvider)` → (score, error)
- `ComputeAnswerRelevance(ctx, question, answer, embedder)` → (score, error)
- `ComputeContextRelevance(ctx, question, contextChunks, embedder)` → (score, error)
- Покрытие: AC-001, AC-002, AC-003, AC-005, AC-006

### Инкремент 2: интеграция

- `Case.ExpectedAnswer string` — опциональный эталонный ответ
- `Metrics.{Faithfulness,AnswerRelevance,ContextRelevance} float64`
- `Options.{EnableFaithfulness,EnableAnswerRelevance,EnableContextRelevance} bool`
- `RunWithAnswer(ctx, runner RetrievalRunner, llm LLMProvider, embedder Embedder, cases []Case, opts Options) (Report, error)`
- Покрытие: AC-004

## Порядок реализации

1. `ragas.go` + `ragas_test.go` — три standalone-функции (MVP, 3 AC).
2. `models.go` — новые поля в Metrics/Options/Case.
3. `harness.go` — `RunWithAnswer`.
4. `metrics.go` — расширение `computeMetrics`.

Пункты 2-4 можно безопасно параллелить после готовности ragas.go (зависимость: функции существуют).

## Риски

- **Риск: JSON-парсинг ответа LLM для Faithfulness**. LLM может вернуть невалидный JSON.
  Mitigation: fallback на повторный запрос с более строгим instruction; если повторно невалид — возвращаем 0 и error.
- **Риск: Embedder размерность не известна заранее**.
  Mitigation: Embedder.Embed возвращает `[]float64` фиксированной размерности; `cosineSimilarity` паникует при mismatch — документируем и проверяем в тестах.

## Rollout and compatibility

- `Run` полностью неизменён.
- `Options` zero-value для новых bool-полей = false → метрики не активны по умолчанию.
- `Case.ExpectedAnswer` опционально, zero value — пустая строка → метрики, требующие ответа, дают 0.
- `RunWithAnswer` — новый экспорт, не ломает существующий код.
- Специальных rollout-действий не требуется.

## Проверка

- `go test ./pkg/draftrag/eval/` — unit-тесты (mock LLM + mock Embedder).
- `go vet ./pkg/draftrag/eval/` — статический анализ.
- AC-001–AC-006 покрыты тестами.
- DEC-001–DEC-004 валидируются через тесты и код-ревью.

## Соответствие конституции

- нет конфликтов
