# Document Lifecycle (Delete + Update)

## Scope Snapshot

- In scope: `Pipeline.DeleteDocument(ctx, docID)` и `Pipeline.UpdateDocument(ctx, doc)` с поддержкой всех 4 vector stores.
- Out of scope: частичное обновление (patch), история версий, транзакционность.

## Цель

Разработчики получают возможность удалять и обновлять проиндексированные документы без необходимости пересоздавать весь store. Это необходимо для систем с изменяемым контентом (wiki, knowledge base).

## Основной сценарий

1. **DeleteDocument**: `pipeline.DeleteDocument(ctx, "doc-id")` → все chunks с `parent_id = "doc-id"` удаляются из store.
2. **UpdateDocument**: `pipeline.UpdateDocument(ctx, doc)` → сначала DeleteDocument, затем Index.
3. **Ошибка**: если store не реализует `DocumentStore` → `ErrDeleteNotSupported`.
4. **Ошибка**: пустой docID → `ErrEmptyDocumentID`.

## Scope

- `internal/domain/interfaces.go`: интерфейс `DocumentStore` (extends `VectorStore` + `DeleteByParentID`)
- `internal/infrastructure/vectorstore/`: `DeleteByParentID` в memory, pgvector, qdrant, chromadb
- `internal/application/pipeline.go`: `DeleteDocument`, `UpdateDocument`
- `pkg/draftrag/draftrag.go`: публичные методы, `ErrDeleteNotSupported`, `ErrEmptyDocumentID`

## Контекст

- Каждый store хранит chunks с полем `parent_id` (docID источника).
- Удаление по `parent_id` — bulk delete; не нужен поиск чанков перед удалением.
- Каждый store имеет нативный bulk-delete API (не поштучное удаление).
- UpdateDocument атомарностью не обладает: если re-index провалится после delete, документ теряется.

## Требования

- **RQ-001** `domain.DocumentStore` интерфейс: `VectorStore` + `DeleteByParentID(ctx, parentID string) error`.
- **RQ-002** `InMemoryStore.DeleteByParentID`: итерация по map, удаление matching chunks.
- **RQ-003** `PGVectorStore.DeleteByParentID`: `DELETE FROM chunks WHERE parent_id = $1`.
- **RQ-004** `QdrantStore.DeleteByParentID`: POST `/collections/{name}/points/delete` с filter body.
- **RQ-005** `ChromaStore.DeleteByParentID`: POST `/api/v1/collections/{name}/delete` с where body.
- **RQ-006** Если store не реализует `DocumentStore` → `ErrDeleteNotSupported`.
- **RQ-007** Nil context → panic; пустой docID → `ErrEmptyDocumentID`.
- **RQ-008** `UpdateDocument` = `DeleteDocument` + `Index`; ошибка Delete прерывает операцию.

## Вне scope

- Транзакционность UpdateDocument.
- Soft delete / tombstoning.
- Batch delete (несколько docID за раз).
- Qdrant/ChromaDB compile-time проверка интерфейса как обязательное требование.

## Критерии приемки

### AC-001 DeleteByParentID отправляет корректный запрос (Qdrant)

- **Given** mock HTTP server для Qdrant
- **When** `store.DeleteByParentID(ctx, "my-doc")`
- **Then** тело запроса содержит `filter.must[0].key="parent_id"` и `match.value="my-doc"`
- **Evidence**: `TestQdrantStore_DeleteByParentID_SendsFilter` pass

### AC-002 DeleteByParentID отправляет корректный запрос (ChromaDB)

- **Given** mock HTTP server для ChromaDB
- **When** `store.DeleteByParentID(ctx, "parent-42")`
- **Then** тело содержит `where.parent_id="parent-42"`
- **Evidence**: `TestChromaStore_DeleteByParentID_SendsWhereFilter` pass

### AC-003 Server error propagation

- **Given** store возвращает HTTP 400/500
- **When** `DeleteByParentID`
- **Then** ошибка не nil
- **Evidence**: ServerError тесты для Qdrant и ChromaDB pass

### AC-004 Context cancellation

- **Given** cancelled context
- **When** `DeleteByParentID`
- **Then** ошибка не nil
- **Evidence**: ContextCancelled тесты pass

### AC-005 ErrDeleteNotSupported

- **Given** store не реализует DocumentStore
- **When** `pipeline.DeleteDocument(ctx, "id")`
- **Then** `errors.Is(err, ErrDeleteNotSupported)`

## Допущения

- `parent_id` хранится в метаданных каждого chunk (уже реализовано в существующих stores).
- UpdateDocument не является атомарным — задокументировано.

## Открытые вопросы

- none
