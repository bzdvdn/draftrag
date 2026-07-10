# RAGAS-style evaluation metrics — Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для этой фичи.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `pkg/draftrag/eval/models.go` | T1.1 |
| `pkg/draftrag/eval/ragas.go` | T2.1, T2.2, T2.3 |
| `pkg/draftrag/eval/ragas_test.go` | T2.4, T3.2 |
| `pkg/draftrag/eval/harness.go` | T3.1 |
| `pkg/draftrag/eval/metrics.go` | T3.1 |

## Implementation Context

- **Цель MVP:** три standalone-функции `ComputeFaithfulness`, `ComputeAnswerRelevance`, `ComputeContextRelevance` в `pkg/draftrag/eval/` (покрытие AC-001, AC-002, AC-003, AC-005, AC-006) + интеграция в `Metrics` + `RunWithAnswer` (AC-004).
- **Инварианты:**
  - Zero value bool = false (метрики отключены по умолчанию)
  - Zero value `Case.ExpectedAnswer` = "" (метрики, требующие ответа, дают 0)
  - Размерность эмбеддингов должна совпадать в рамках одного вызова
  - LLMProvider/Embedder — публичные type aliases из `pkg/draftrag` (`= domain.LLMProvider`, `= domain.Embedder`)
- **Ошибки:**
  - LLM-вызов Faithfulness: timeout/rate-limit → error (не 0)
  - Embedder-вызов: error → пробрасывается наверх
  - nil LLMProvider/Embedder при включённой метрике → score 0, nil error
- **Proof signals:**
  - `go test ./pkg/draftrag/eval/` проходит
  - `ComputeFaithfulness` с mock LLM возвращает 1.0 для полностью подтверждённого ответа
  - `RunWithAnswer` возвращает Metrics с ненулевыми RAGAS-полями
- **Вне scope:** изменение `Run()`, визуализация, export, кастомные prompt-шаблоны
- **References:** DEC-001 (standalone functions), DEC-002 (один LLM-вызов Faithfulness), DEC-003 (Embedder-based), DEC-004 (RunWithAnswer — новый экспорт)

## Фаза 1: Data model

Цель: добавить новые поля в существующие структуры данных eval-пакета.

- [x] T1.1 Добавить поля в `Case` (`ExpectedAnswer string`), `Metrics` (`Faithfulness`, `AnswerRelevance`, `ContextRelevance float64`), `Options` (`EnableFaithfulness`, `EnableAnswerRelevance`, `EnableContextRelevance bool`). Touches: `pkg/draftrag/eval/models.go`

## Фаза 2: MVP — standalone функции

Цель: реализовать три независимые RAGAS-метрики как экспортируемые функции + тесты.

- [x] T2.1 Реализовать `ComputeFaithfulness(ctx, answer, context, llmProvider)` — один LLM-вызов с prompt на декомпозицию ответа в claims и верификацию каждого claim против контекста, парсинг JSON-ответа (поле `faithfulness_score`). Graceful degradation: пустой answer → 0, nil llmProvider → 0. Touches: `pkg/draftrag/eval/ragas.go`
- [x] T2.2 Реализовать `ComputeContextRelevance(ctx, question, contextChunks, embedder)` — для каждого чанка вычисляется embedding, усредняется косинусная близость с embedding вопроса. Internal helper: `cosineSimilarity(a, b []float64) float64`. Graceful degradation: пустой contextChunks → 0, nil embedder → 0. Touches: `pkg/draftrag/eval/ragas.go`
- [x] T2.3 Реализовать `ComputeAnswerRelevance(ctx, question, answer, embedder)` — LLM-вызов генерирует N=3 вопроса из ответа, каждый эмбеддится, усредняется косинусная близость с embedding исходного вопроса. Graceful degradation: пустой answer → 0, nil embedder → 0. Touches: `pkg/draftrag/eval/ragas.go`
- [x] T2.4 Тесты для T2.1–T2.3: mock LLMProvider (возвращает JSON с score), mock Embedder (возвращает векторы с известным расстоянием). Проверка AC-001, AC-002, AC-003, AC-005, AC-006. Touches: `pkg/draftrag/eval/ragas_test.go`

## Фаза 3: Интеграция

Цель: связать standalone-метрики с eval-отчётом через `RunWithAnswer` и `computeMetrics`.

- [x] T3.1 Добавить `RunWithAnswer(ctx, runner, llm, embedder, cases, opts)` в harness.go — итерация по кейсам, вызов `runner.Retrieve` + генерация ответа через `llm.Generate`, затем вызов трёх RAGAS-функций для каждого кейса. Расширить `computeMetrics` в metrics.go для агрегации новых метрик. Touches: `pkg/draftrag/eval/harness.go`, `pkg/draftrag/eval/metrics.go`
- [x] T3.2 Интеграционный тест для `RunWithAnswer` с mock retrieval runner, mock LLM и mock embedder. Проверка AC-004: поля `Metrics.Faithfulness`, `Metrics.AnswerRelevance`, `Metrics.ContextRelevance` заполнены. Touches: `pkg/draftrag/eval/ragas_test.go`

## Фаза 4: Проверка

Цель: доказать, что фича работает, и оставить пакет в reviewable состоянии.

- [x] T4.1 Запустить `go test ./pkg/draftrag/eval/` и `go vet ./pkg/draftrag/eval/`. Touches: `pkg/draftrag/eval/`

## Покрытие критериев приемки

- AC-001 -> T2.1, T2.4
- AC-002 -> T2.3, T2.4
- AC-003 -> T2.2, T2.4
- AC-004 -> T3.1, T3.2
- AC-005 -> T2.1, T2.2, T2.3, T2.4
- AC-006 -> T2.1, T2.4
