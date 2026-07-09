# Reranker: cross-encoder и Cohere Rerank — Задачи

## Phase Contract

Inputs: `plan.md`, `data-model.md`, `spec.md`.
Outputs: упорядоченные задачи с поверхностями и покрытием AC.
Stop if: AC нельзя привязать к задачам — нет, все AC покрыты.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `pkg/draftrag/reranker/reranker.go` | T1.1 |
| `pkg/draftrag/reranker/errors.go` | T1.2 |
| `pkg/draftrag/draftrag.go` | T1.1 |
| `pkg/draftrag/reranker/cohere.go` | T2.1, T3.1, T3.2 |
| `internal/application/retrieval.go` | T3.2 |
| `internal/application/query.go` | T3.2 |
| `pkg/draftrag/reranker/cohere_test.go` | T4.1 |
| `pkg/draftrag/reranker_test.go` | T4.1 |
| `docs/en/reranker.md` | T4.2 |
| `docs/ru/reranker.md` | T4.2 |

## Implementation Context

- **Цель MVP**: Cohere Rerank API (single + batch) через `pkg/draftrag/reranker/`. LLM-reranker отложен (P2).
- **Инварианты/семантика**:
  - `BatchReranker` extends `Reranker` — опциональная capability, type-assert
  - Reranker **не фильтрует** чанки: `len(out) == len(in)`
  - Nil reranker = no-op (guard в `maybeRerank`)
  - Cohere batch = concurrent fan-out (errgroup), не один HTTP-запрос
- **Ошибки/коды**:
  - `ErrInvalidRerankerConfig` — sentinel для валидации опций
  - API-ключ логируется через `RedactSecrets` (не в открытом виде)
- **Контракты/протокол**:
  - `POST https://api.cohere.com/v2/rerank` — model, query (string), documents ([]string)
  - Response: `results[{index, relevance_score}]` — сортировка по `relevance_score` desc
- **Границы scope**:
  - Не реализуем LLM-reranker (P2)
  - Не меняем store-реализации, SearchBuilder, PipelineOptions
- **Proof signals**:
  - `go test ./pkg/draftrag/reranker/...` passes
  - Mock HTTP-сервер подтверждает batch fan-out (N параллельных запросов)
  - AC-001–AC-003, AC-006–AC-010 покрыты тестами
- **References**: DEC-001 (пакет), DEC-002 (BatchReranker interface), DEC-003 (concurrent fan-out), DM (data model)

## Фаза 0: Scope

- [x] T0.1 AC-004, AC-005 (LLM-reranker, P2) — отложены. Не входят в MVP.
  Touches: `docs/specs/reranker-cross-encoder/spec.md`

## Фаза 1: Foundation

Цель: подготовить интерфейсы, пакет и sentinel-ошибки.

- [x] T1.1 Добавить `BatchReranker` интерфейс в `internal/domain/interfaces.go`, создать `pkg/draftrag/reranker/reranker.go` с re-export'ами, добавить re-export `BatchReranker` в `pkg/draftrag/draftrag.go`.
  Touches: `internal/domain/interfaces.go`, `pkg/draftrag/reranker/reranker.go`, `pkg/draftrag/draftrag.go`

- [x] T1.2 Добавить `ErrInvalidRerankerConfig` sentinel в `pkg/draftrag/reranker/errors.go`. Паттерн: `fmt.Errorf("%w: detail", ErrInvalidRerankerConfig)`.
  Touches: `pkg/draftrag/reranker/errors.go`

## Фаза 2: Cohere single-query

Цель: реализовать базовый `CohereReranker` (один query).

- [x] T2.1 Реализовать `CohereReranker` с `Rerank(ctx, query, chunks)`:
  - `CohereRerankOptions` с `APIKey` (обязательно), `Model` (default `rerank-english-v3.0`), `BaseURL` (default `https://api.cohere.com/v2`), `Timeout`, `MaxRetries` (default 2), `MaxTokensPerDoc` (default 4096)
  - Constructor validation: пустой ключ → `ErrInvalidRerankerConfig`
  - HTTP-запрос к `POST {BaseURL}/rerank`
  - Маппинг ответа: `results[].index` → исходный chunk, сортировка по `relevance_score` desc
  - `RedactSecrets` для API-ключа в логах/ошибках
  - No-filter invariant: всегда возвращает len(in) чанков
  - Пустой вход → пустой выход без ошибки
  Touches: `pkg/draftrag/reranker/cohere.go`

## Фаза 3: Batch + pipeline integration

Цель: batch-режим для Cohere + интеграция в pipeline MultiQuery.

- [x] T3.1 Добавить обработку ошибок Cohere API:
  - 401/403 → error с сообщением
  - 429/5xx → retry через `MaxRetries` (backoff через контекст или простой sleep)
  - Таймаут → context.DeadlineExceeded
  Touches: `pkg/draftrag/reranker/cohere.go`

- [x] T3.2 Реализовать `RerankBatch` в `CohereReranker` и pipeline-интеграцию:
  - `RerankBatch(ctx, queries, chunks)` → errgroup с N goroutines, каждая вызывает `Rerank` для одного query
  - В `internal/application/retrieval.go`: `maybeRerankBatch(ctx, queries, result)`
  - В `internal/application/query.go` `QueryMulti`: после RRF merge → type-assert `BatchReranker` → `RerankBatch` если имплементирует, иначе `Rerank` (fallback)
  Touches: `pkg/draftrag/reranker/cohere.go`, `internal/application/retrieval.go`, `internal/application/query.go`

## Фаза 4: Проверка

Цель: тесты, документация, verify.

- [x] T4.1 Написать тесты для всех AC MVP:
  - `TestCohereRerank_Success` — mock HTTP-сервер, проверка порядка (AC-001)
  - `TestCohereRerank_EmptyChunks` — no-op (AC-002)
  - `TestCohereRerank_InvalidKey` — constructor error (AC-003)
  - `TestCohereRerank_Unauthorized` — 401 error (AC-006)
  - `TestCohereRerank_NoFilter` — len(out) == len(in) (AC-007)
  - `TestCohereRerank_BatchFanOut` — errgroup, latency < 150ms для 5×100ms (AC-008)
  - `TestCohereRerank_BatchFallback` — последовательный вызов при отсутствии BatchReranker (AC-009)
  - Обновить `pkg/draftrag/reranker_test.go`: `TestPipeline_Reranker_IsCalled` с CohereReranker
  Touches: `pkg/draftrag/reranker/cohere_test.go`, `pkg/draftrag/reranker_test.go`

- [x] T4.2 Документация и cleanup:
  - `docs/en/reranker.md` — пример с Cohere, batch-режим, сравнение производительности
  - `docs/ru/reranker.md` — то же на русском
  - Обновить `ROADMAP.md`: перенести `reranker-cross-encoder` в ✅
  Touches: `docs/en/reranker.md`, `docs/ru/reranker.md`, `ROADMAP.md`

## Покрытие критериев приемки

- AC-001 -> T2.1, T4.1
- AC-002 -> T2.1, T4.1
- AC-003 -> T1.2, T2.1, T4.1
- AC-004 -> T0.1
- AC-005 -> T0.1
- AC-006 -> T3.1, T4.1
- AC-007 -> T2.1, T4.1
- AC-008 -> T3.2, T4.1
- AC-009 -> T3.2, T4.1
- AC-010 -> T4.2

## Заметки
- Фазы идут sequentially: Foundation → Cohere → Batch → Verify. Параллельно: только T4.2 (docs) частично overlap с T4.1.
- После verify → `speckeep archive reranker-cross-encoder .`
