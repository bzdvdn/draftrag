---
slug: milvus-hybrid-search
generated_at: 2026-04-15
---

## Goal

Разработчики получают возможность использовать гибридный поиск (BM25 + semantic fusion) в Milvus через интерфейс HybridSearcher.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | MilvusStore реализует HybridSearcher интерфейс | Compile-time assertion без ошибок |
| AC-002 | SearchHybrid использует BM25 и dense векторы через AnnSearchRequest | Код создаёт AnnSearchRequest для text_sparse и text_dense |
| AC-003 | SearchHybrid поддерживает fusion-стратегии RRF и weighted | Код передаёт rerank strategy в hybrid_search() |
| AC-004 | MilvusStore реализует HybridSearcherWithFilters с фильтрацией | Код добавляет expr параметр в AnnSearchRequest |
| AC-005 | SearchHybrid валидирует HybridConfig перед выполнением | Код вызывает config.Validate() и возвращает ошибку |
| AC-006 | Код обрабатывает ошибки Milvus API информативно | Код оборачивает ошибки от Milvus в информативные messages |

## Out of Scope

- Поддержка других vectorstores (chromadb, memory)
- Reranking с custom strategies или late interaction models
- Другие fusion стратегии кроме RRF и weighted
- Изменение существующих методов MilvusStore (Upsert, Search, Delete)
- Миграция существующих данных или schema changes
