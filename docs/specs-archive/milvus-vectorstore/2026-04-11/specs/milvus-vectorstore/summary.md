---
slug: milvus-vectorstore
generated_at: 2026-04-11
---

## Goal

Реализовать `MilvusStore` через Milvus REST API v2 (без SDK), удовлетворяющий `domain.VectorStore`, `VectorStoreWithFilters` и `DocumentStore`.

## Acceptance Criteria

| ID     | Summary                              | Proof Signal                                      |
|--------|--------------------------------------|---------------------------------------------------|
| AC-001 | Upsert отправляет POST upsert        | unit-тест: мок-сервер получает корректное тело    |
| AC-002 | Delete отправляет фильтр по id       | unit-тест: фильтр-выражение `id == "<id>"`        |
| AC-003 | Search возвращает ≤topK чанков       | unit-тест: десериализация ответа мок-сервера      |
| AC-004 | SearchWithFilter фильтрует ParentID  | unit-тест: фильтр `parent_id in [...]` в запросе  |
| AC-005 | SearchWithMetadataFilter по полям    | unit-тест: фильтр по metadata-полю в запросе      |
| AC-006 | DeleteByParentID удаляет документ    | unit-тест: фильтр `parent_id == "..."` в запросе  |
| AC-007 | Compile-time assertions              | `go build ./...` без ошибок                       |
| AC-008 | HTTP/API ошибки возвращают error     | unit-тест: error-path, паники нет                 |

## Out of Scope

- Создание и управление коллекциями Milvus
- Поддержка Milvus < 2.3 (нет REST API v2)
- gRPC-транспорт и официальный milvus-sdk-go
- Гибридный поиск (HybridSearcher)
- Интеграция с Zilliz Cloud
