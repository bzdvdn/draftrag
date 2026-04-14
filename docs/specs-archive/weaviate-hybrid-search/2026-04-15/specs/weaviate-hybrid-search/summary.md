---
slug: weaviate-hybrid-search
generated_at: 2026-04-14
---

## Goal

Разработчики получают возможность использовать гибридный поиск (BM25 + semantic fusion) в Weaviate через интерфейс HybridSearcher.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Реализация HybridSearcher интерфейса | WeaviateStore реализует метод SearchHybrid |
| AC-002 | Использование GraphQL API с BM25 и nearVector | Код содержит GraphQL запрос с bm25 и nearVector |
| AC-003 | Использование fusion-стратегии | Код содержит fusion с типом rrf или weighted |
| AC-004 | Реализация HybridSearcherWithFilters | WeaviateStore реализует методы с фильтрами |
| AC-005 | Валидация HybridConfig | SearchHybrid вызывает config.Validate() |
| AC-006 | Обработка ошибок GraphQL API | Код обрабатывает GraphQL и HTTP ошибки |

## Out of Scope

- Поддержка других vectorstores (chromadb, milvus, memory)
- Reranking с late interaction models (ColBERT, SPLADE)
- Matryoshka embeddings и multi-step retrieval
- Другие fusion стратегии кроме RRF и weighted
