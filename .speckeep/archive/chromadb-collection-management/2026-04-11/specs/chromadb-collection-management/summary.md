---
slug: chromadb-collection-management
generated_at: 2026-04-11
---

## Goal

Добавить публичный domain-интерфейс `CollectionManager` и реализовать его в `ChromaStore` для programmatic управления жизненным циклом коллекции ChromaDB.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | CreateCollection idempotent | nil при существующей и несуществующей коллекции |
| AC-002 | DeleteCollection отправляет DELETE | mock захватывает DELETE /api/v1/collections/{name} → nil |
| AC-003 | DeleteCollection: 404→nil, 5xx→error | mock 404 → nil; mock 500 → err со статусом |
| AC-004 | CollectionExists → true при 200 | возвращает (true, nil) |
| AC-005 | CollectionExists → false при 404 | возвращает (false, nil), не ошибку |
| AC-006 | Compile-time assertion | go build ./... без ошибок |

## Out of Scope

- Гибридный поиск (BM25) для ChromaDB
- Управление коллекциями других хранилищ (Qdrant, Milvus, pgvector)
- Настройка параметров коллекции через CollectionManager
- ListCollections
