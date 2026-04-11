# Document Lifecycle — Задачи

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/domain/interfaces.go | T1.1 |
| internal/infrastructure/vectorstore/memory.go | T2.1 |
| internal/infrastructure/vectorstore/pgvector.go | T2.2 |
| internal/infrastructure/vectorstore/qdrant.go | T2.3 |
| internal/infrastructure/vectorstore/chromadb.go | T2.4 |
| internal/application/pipeline.go | T2.5 |
| pkg/draftrag/draftrag.go | T2.6 |
| internal/infrastructure/vectorstore/qdrant_delete_test.go | T3.1 |
| internal/infrastructure/vectorstore/chromadb_delete_test.go | T3.2 |

## Фаза 1: Основа

- [x] T1.1 Добавить `domain.DocumentStore` интерфейс (`VectorStore` + `DeleteByParentID`). Touches: internal/domain/interfaces.go

## Фаза 2: Основная реализация

- [x] T2.1 Реализовать `InMemoryStore.DeleteByParentID`: итерация по map. Touches: internal/infrastructure/vectorstore/memory.go
- [x] T2.2 Реализовать `PGVectorStore.DeleteByParentID`: `DELETE WHERE parent_id = $1`. Touches: internal/infrastructure/vectorstore/pgvector.go
- [x] T2.3 Реализовать `QdrantStore.DeleteByParentID`: POST delete с filter body; добавить compile-time assertion. Touches: internal/infrastructure/vectorstore/qdrant.go
- [x] T2.4 Реализовать `ChromaStore.DeleteByParentID`: POST delete с where body; добавить compile-time assertion. Touches: internal/infrastructure/vectorstore/chromadb.go
- [x] T2.5 Реализовать `Pipeline.DeleteDocument` и `Pipeline.UpdateDocument` с capability check. Touches: internal/application/pipeline.go
- [x] T2.6 Добавить публичные `DeleteDocument`, `UpdateDocument`, `ErrDeleteNotSupported`, `ErrEmptyDocumentID`. Touches: pkg/draftrag/draftrag.go

## Фаза 3: Проверка

- [x] T3.1 Написать `qdrant_delete_test.go`: SendsFilter, ServerError, ContextCancelled, NilContextPanics. Touches: internal/infrastructure/vectorstore/qdrant_delete_test.go
- [x] T3.2 Написать `chromadb_delete_test.go`: SendsWhereFilter, ServerError, ContextCancelled, NilContextPanics. Touches: internal/infrastructure/vectorstore/chromadb_delete_test.go
- [x] T3.3 Убедиться что `go test ./...` проходит без ошибок.

## Покрытие критериев приемки

- AC-001 → T2.3, T3.1
- AC-002 → T2.4, T3.2
- AC-003 → T3.1, T3.2
- AC-004 → T3.1, T3.2
- AC-005 → T2.5
