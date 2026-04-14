# Retrieval Strategies — План

## Цель

Добавить HyDE, Multi-Query и Reranker как независимые, opt-in расширения retrieval. Все три работают поверх существующего `Query` infrastructure без изменения vector store или chunker.

## Scope

- `internal/domain/interfaces.go` — Reranker интерфейс
- `internal/application/pipeline.go` — QueryHyDE, AnswerHyDE, QueryMulti, AnswerMulti, maybeRerank, rrfMergeMultiple
- `pkg/draftrag/draftrag.go` — тип-алиас Reranker, поле в PipelineOptions
- `pkg/draftrag/search.go` — builder методы HyDE(), MultiQuery(n)
- Нетронутыми остаются vector stores

## Implementation Surfaces

- `internal/domain/interfaces.go` — новый интерфейс `Reranker`; существующая поверхность
- `internal/application/pipeline.go` — новые методы; существующая поверхность
- `pkg/draftrag/` — публичный экспорт; существующая поверхность

## Влияние на архитектуру

- `PipelineConfig` получает поле `Reranker` — additive, без breaking change.
- `maybeRerank` вызывается в каждом Query-методе после `maybeDedup`; не меняет контракт.
- HyDE и MultiQuery — новые методы, не меняют существующие.

## Acceptance Approach

- AC-001 → `QueryHyDE` в application; routing в `SearchBuilder.Retrieve`
- AC-002 → `QueryMulti` + `rrfMergeMultiple` в application; routing в `SearchBuilder.Retrieve`
- AC-003 → `maybeRerank` вызывается во всех Query-методах
- AC-004 → nil reranker → `maybeRerank` возвращает result без изменений

## Стратегия реализации

- DEC-001 RRF реализован в application, не в vectorstore
  Why: multi-query — это application-level операция; vectorstore не знает о нескольких запросах
  Tradeoff: дублирует RRF-логику из hybrid search; но dependency от vectorstore была бы неправильной
  Affects: internal/application/pipeline.go
  Validation: RRF объединяет результаты корректно (unit test с детерминированными вводами)

- DEC-002 HyDE системный промпт — константа
  Why: пользователи редко нуждаются в кастомизации; конфигурируемость — premature abstraction
  Tradeoff: нельзя изменить без патча
  Affects: internal/application/pipeline.go
  Validation: HyDE тест проходит с mockLLM

## Порядок реализации

1. `domain.Reranker` интерфейс
2. `maybeRerank` + `PipelineConfig.Reranker`
3. `QueryHyDE` / `AnswerHyDE`
4. `rrfMergeMultiple` + `QueryMulti` / `AnswerMulti`
5. `pkg/draftrag` публичный экспорт
6. builder методы в search.go
7. Тесты

## Риски

- Риск: multi-query делает N+1 LLM вызовов (1 для парафраз + N поисков) — latency растёт
  Mitigation: задокументировано; пользователь выбирает N осознанно (рекомендуется 2-4)

## Rollout и compatibility

- Additive; нет breaking changes.
- Reranker nil по умолчанию → без изменений для существующих пользователей.

## Проверка

- `go test ./pkg/draftrag/...` — HyDE, MultiQuery, Reranker тесты pass
- `go test ./internal/application/...` — application тесты pass
