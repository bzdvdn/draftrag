# Retrieval: MMR rerank (диверсификация источников) (v1) — План

## Phase Contract

Inputs: spec + inspect.  
Outputs: план реализации (surfaces, решения, риски) и supporting артефакты при необходимости.  
Stop if: нельзя сделать MMR детерминированным и безопасным без внешних зависимостей.

## Цель

Добавить опциональный шаг MMR selection поверх retrieval кандидатов, чтобы уменьшить дублирование контекста (overlap) и повысить разнообразие источников при сохранении релевантности.

## Scope

- MMR включается только через опцию (по умолчанию выключен).
- Конфигурация v1:
  - `Lambda` в диапазоне `[0..1]`
  - `CandidatePool` (сколько кандидатов запрашивать у VectorStore до отбора)
- Используем только уже имеющиеся embeddings: `queryEmbedding` + `chunk.Embedding`.
- Применяем MMR в Answer* путях (формирование контекста/prompt).

## Implementation Surfaces

- `internal/application/pipeline.go`
  - расширить `PipelineConfig` полями MMR
  - в `Answer*` путях: делать search на `candidateTopK`, затем MMR selection до `topK`
- `internal/application/mmr.go` (новый файл)
  - детерминированная реализация MMR selection + cosine similarity
- `pkg/draftrag/draftrag.go`
  - расширить `PipelineOptions` полями MMR
  - прокинуть в `application.PipelineConfig`
- `internal/application/retrieval_reranker_mmr_test.go` (новый файл)
  - синтетические embeddings с кластерами и проверка AC-001/AC-002

## Влияние на архитектуру

- Domain не меняется (MMR — application concern).
- Public API расширяется аддитивно через `PipelineOptions`.
- Поведение по умолчанию не меняется.

## Acceptance Approach

- AC-001:
  - подготовить retrieval candidates, где “самые релевантные” — из одного кластера (очень похожи друг на друга)
  - включить MMR с `CandidatePool > topK`
  - ожидать, что выбранный набор содержит элементы из разных кластеров (диверсификация)
- AC-002:
  - при выключенном MMR выбранные чанки и порядок совпадают с текущим (score desc, topK)

## Данные и контракты

- Никаких новых persistent данных.
- Контракт v1: MMR разрешено включать только если `Chunk.Embedding` присутствует у кандидатов.
  - Если embeddings отсутствуют, MMR в v1 возвращает ошибку конфигурации/выполнения (решение зафиксировать в задачах).

## Стратегия реализации

- DEC-001 MMR как selection поверх `candidateTopK`
  Why: VectorStore остаётся простым (Search), MMR — чистая post-processing логика в application.
  Tradeoff: нужно запрашивать больше кандидатов (CandidatePool), что может увеличивать latency.
  Affects: `internal/application/pipeline.go`, `internal/application/mmr.go`.
  Validation: unit-тесты на selection.

- DEC-002 Relevance term = исходный score, diversity term = cosine(chunk, selected)
  Why: store score уже отражает релевантность; embeddings нужны только для “похожести между чанками”.
  Tradeoff: score может быть не cosine(query, chunk), но для v1 достаточно практичной эвристики.
  Affects: `internal/application/mmr.go`.
  Validation: синтетические embeddings, где similarity между кластерами контролируемая.

- DEC-003 Детерминизм: tie-breaker по исходному индексу
  Why: тестируемость и стабильность поведения.
  Tradeoff: при равных значениях MMR выбирается “раньше пришедший” кандидат.
  Affects: `internal/application/mmr.go`.
  Validation: тесты с равными score/sim.

## Риски

- Embeddings могут отсутствовать или быть нулевой длины.
  Mitigation: v1 — явная ошибка при включенном MMR и отсутствии embeddings у кандидатов.

- CandidatePool может быть меньше topK или слишком большим.
  Mitigation: нормализовать: `candidateTopK = max(topK, CandidatePool)`; валидация параметров в public API.

## Проверка

- `go test ./...`
- Проверка, что existing Answer* тесты не сломались.

## Соответствие конституции

- Нет конфликтов: интерфейсы не ломаются, поведение по умолчанию сохраняется, тесты детерминированны, контекстная отмена не нарушается.

