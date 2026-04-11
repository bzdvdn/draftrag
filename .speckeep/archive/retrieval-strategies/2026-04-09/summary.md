# Сводка архива

## Спецификация

- snapshot: добавлены три opt-in retrieval стратегии — HyDE, Multi-Query с RRF и pluggable Reranker
- slug: retrieval-strategies
- archived_at: 2026-04-09
- status: completed

## Причина

Базовый семантический поиск ограничен: вопрос и релевантный документ могут иметь разные формулировки (HyDE), embedding чувствителен к формулировке вопроса (Multi-Query), precision@K можно улучшить cross-encoder'ом (Reranker). Каждая стратегия ортогональна и opt-in.

## Результат

- `domain.Reranker` интерфейс; `PipelineOptions.Reranker`; `maybeRerank` во всех Query-методах.
- `QueryHyDE` / `AnswerHyDE` — гипотетический документ → поиск.
- `QueryMulti` / `AnswerMulti` + `rrfMergeMultiple` (k=60).
- Builder-методы `HyDE()` и `MultiQuery(n)` в SearchBuilder.
- Тесты: `reranker_test.go` (2 теста) + HyDE/MultiQuery в `search_builder_test.go` (4 теста).
- Документация в `docs/advanced.md`: секции HyDE, Multi-Query, Reranker.

## Продолжение

- Параллельные multi-query запросы через горутины для снижения latency.
- Несколько гипотетических документов в HyDE (ensemble embedding).
- Конкретные реализации Reranker (cross-encoder обёртки для Cohere, Jina и т.д.).
