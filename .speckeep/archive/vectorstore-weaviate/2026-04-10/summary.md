---
slug: vectorstore-weaviate
generated_at: 2026-04-10
---

## Goal

Добавить поддержку Weaviate как VectorStore с теми же возможностями поиска и фильтрации, что у Qdrant и pgvector.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Upsert → Search round-trip | RetrievalResult с совпадающими полями чанка; Score > 0 |
| AC-002 | SearchWithFilter по ParentID | Все результаты имеют ParentID == "doc-A" |
| AC-003 | SearchWithMetadataFilter | Все результаты имеют Metadata["category"] == "go" |
| AC-004 | Delete идемпотентен | Оба вызова (несуществующий и существующий ID) возвращают nil |
| AC-005 | PublicAPI NewWeaviateStore | go build ./... ok; ErrInvalidConfig при пустом host |

## Out of Scope

- Гибридный BM25+vector поиск через Weaviate
- Использование Weaviate-модулей для генерации эмбеддингов
- Многотенантность Weaviate
- Weaviate Cloud OIDC/OAuth аутентификация
- Автоматическая миграция схемы
