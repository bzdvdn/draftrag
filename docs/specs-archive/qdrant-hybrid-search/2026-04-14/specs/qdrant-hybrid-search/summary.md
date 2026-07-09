---
slug: qdrant-hybrid-search
generated_at: 2026-04-14
---

## Goal

Разработчики получают возможность использовать гибридный поиск (BM25 + semantic fusion) в Qdrant через интерфейс HybridSearcher.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Реализация HybridSearcher интерфейса | QdrantStore реализует метод SearchHybrid |
| AC-002 | Использование Query API с Prefetch | Код содержит Query API вызов с Prefetch |
| AC-003 | Использование Fusion.RRF | Код содержит FusionQuery с Fusion.RRF |
| AC-004 | Реализация HybridSearcherWithFilters | QdrantStore реализует методы с фильтрами |
| AC-005 | Валидация HybridConfig | SearchHybrid вызывает config.Validate() |
| AC-006 | Обработка ошибок Query API | Код обрабатывает HTTP ошибки |

## Out of Scope

- Поддержка других vectorstores (weaviate, chromadb, milvus, memory)
- Reranking с late interaction models (ColBERT, SPLADE)
- Matryoshka embeddings и multi-step retrieval
- Другие fusion стратегии кроме RRF
