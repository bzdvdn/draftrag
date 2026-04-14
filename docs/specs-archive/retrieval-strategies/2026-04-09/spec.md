# Retrieval Strategies (HyDE, Multi-Query, Reranker)

## Scope Snapshot

- In scope: три независимые стратегии улучшения retrieval — HyDE, Multi-Query с RRF и pluggable Reranker интерфейс.
- Out of scope: конкретные реализации cross-encoder reranker, изменения chunker/embedder моделей.

## Цель

Разработчики получают три ортогональных инструмента улучшения качества RAG-поиска без изменения основного pipeline: HyDE для смысловой дистанции вопрос/ответ, Multi-Query для robustness к формулировке, Reranker для post-retrieval scoring.

## Основной сценарий

1. **HyDE**: LLM генерирует гипотетический ответ → его embedding используется для поиска вместо embedding вопроса.
2. **Multi-Query**: LLM генерирует N перефразировок → N параллельных поисков → RRF-объединение топ-K.
3. **Reranker**: после retrieval вызывается `Reranker.Rerank(ctx, query, chunks)` → переупорядоченный список.

## Scope

- `internal/domain/interfaces.go`: добавить `Reranker` интерфейс
- `internal/application/pipeline.go`: `QueryHyDE`, `AnswerHyDE`, `QueryMulti`, `AnswerMulti`, `maybeRerank`
- `pkg/draftrag/draftrag.go`: `type Reranker = domain.Reranker`, поле в `PipelineOptions`
- `pkg/draftrag/search.go`: builder-методы `HyDE()` и `MultiQuery(n)`

## Контекст

- HyDE (Gao et al. 2022): гипотетические документы имеют меньшее embedding-расстояние до релевантных chunks, чем сам вопрос.
- Multi-Query: paraphrasing компенсирует чувствительность embedding моделей к формулировке.
- RRF (k=60): стандартный параметр из IR-литературы; устойчив к разным масштабам scores.
- Reranker: cross-encoders (напр., ms-marco-MiniLM) значительно улучшают precision@K; library-level интерфейс позволяет подключить любую реализацию.

## Требования

- **RQ-001** `domain.Reranker` интерфейс: `Rerank(ctx, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)`.
- **RQ-002** `PipelineOptions.Reranker` — опциональный; если nil, шаг пропускается.
- **RQ-003** `Reranker` вызывается после `maybeDedup` во всех Query-методах.
- **RQ-004** `QueryHyDE`: `llm.Generate(hydeSystemPrompt, question)` → embed result → store.Search.
- **RQ-005** `QueryMulti(n)`: LLM генерирует n парафраз → n поисков → `rrfMergeMultiple`.
- **RQ-006** RRF формула: `score += 1.0 / (60 + rank + 1)` для каждого списка; sort desc; trim topK.
- **RQ-007** SearchBuilder: `HyDE()` и `MultiQuery(n)` — builder-методы, приоритет HyDE > MultiQuery в routing.

## Вне scope

- Конкретная реализация cross-encoder reranker (пользователь подключает своё).
- Async/concurrent multi-query поиск (реализовано последовательно).
- HyDE с несколькими гипотетическими документами (только один).
- Кэширование LLM вызовов для парафраз.

## Критерии приемки

### AC-001 HyDE Retrieve

- **Почему важно**: расширяет recall на вопросы с нестандартными формулировками.
- **Given** pipeline с `fixedEmbedder` и заполненным store
- **When** `Search("q").TopK(2).HyDE().Retrieve(ctx)`
- **Then** возвращаются chunks без ошибки
- **Evidence**: `TestSearchBuilder_HyDE` pass

### AC-002 Multi-Query Retrieve

- **Почему важно**: robustness к формулировке вопроса.
- **Given** pipeline с mockLLM (возвращает "ok" на любой запрос)
- **When** `Search("q").TopK(2).MultiQuery(2).Retrieve(ctx)`
- **Then** возвращаются deduplicated chunks
- **Evidence**: `TestSearchBuilder_MultiQuery` pass

### AC-003 Reranker вызывается

- **Почему важно**: pluggable post-retrieval scoring должен срабатывать.
- **Given** pipeline с `reverseReranker` (mock, устанавливает `called=true`)
- **When** `Search("q").TopK(2).Retrieve(ctx)`
- **Then** `rr.called == true` и chunks непустые
- **Evidence**: `TestPipeline_Reranker_IsCalled` pass

### AC-004 Без Reranker работает

- **Почему важно**: нет регрессии для существующих pipeline.
- **Given** pipeline без Reranker (nil)
- **When** `Search("q").TopK(1).Retrieve(ctx)`
- **Then** возвращается результат без ошибки
- **Evidence**: `TestPipeline_NoReranker_Works` pass

## Допущения

- HyDE системный промпт фиксирован: "Write a short factual passage that would directly answer the question."
- Multi-Query парсинг: split by `\n`, trim empty lines.
- RRF k=60 — константа, не конфигурируется (по аналогии с hybrid search).

## Открытые вопросы

- none
