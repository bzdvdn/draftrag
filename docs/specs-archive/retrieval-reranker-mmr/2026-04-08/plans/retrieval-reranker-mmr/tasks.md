# Retrieval: MMR rerank (диверсификация источников) (v1) — Задачи

## Phase Contract

Inputs: plan + spec + inspect.  
Outputs: конкретные задачи с покрытием AC, детерминированные тесты, отсутствие сетевых зависимостей.  
Stop if: нельзя получить детерминированный результат на синтетических embeddings.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/application/mmr.go | T1.1, T2.1 |
| internal/application/pipeline.go | T2.2 |
| pkg/draftrag/draftrag.go | T2.3 |
| internal/application/retrieval_reranker_mmr_test.go | T3.1 |

## Фаза 1: Основа

- [x] T1.1 Добавить MMR selection и cosine similarity (детерминированно, tie-breaker по исходному индексу). Touches: `internal/application/mmr.go`. (RQ-003, RQ-005)

## Фаза 2: Основная реализация

- [x] T2.1 Добавить валидируемую конфигурацию MMR в application config (enabled/disabled, `Lambda`, `CandidatePool`). Touches: `internal/application/pipeline.go`. (RQ-001, RQ-002)
- [x] T2.2 Интегрировать MMR в Answer* path: search на `candidateTopK = max(topK, CandidatePool)`, затем selection до `topK`. Touches: `internal/application/pipeline.go`. (AC-001, AC-002)
- [x] T2.3 Добавить публичные опции в `PipelineOptions` и прокинуть в application config. Touches: `pkg/draftrag/draftrag.go`. (RQ-001, RQ-002)

## Фаза 3: Проверка

- [x] T3.1 Добавить unit-тесты: MMR включён диверсифицирует выбор (AC-001), MMR выключен не меняет baseline (AC-002), включённый MMR требует embeddings (guard). Touches: `internal/application/retrieval_reranker_mmr_test.go`. (RQ-004, RQ-005)
- [x] T3.2 Прогнать `go test ./...` и убедиться, что нет регрессий и сеть не используется. Touches: repo.

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.2, T3.1
- AC-002 -> T2.2, T3.1, T3.2
