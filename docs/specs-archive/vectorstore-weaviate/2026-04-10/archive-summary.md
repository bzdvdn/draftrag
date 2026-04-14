---
slug: vectorstore-weaviate
status: completed
archived_at: 2026-04-10
---

# Archive Summary: vectorstore-weaviate

## Status

completed

## Reason

Weaviate VectorStore реализован: добавлена поддержка Weaviate как полноценного `VectorStore` + `VectorStoreWithFilters` по аналогии с Qdrant и pgvector. Все 5 acceptance criteria закрыты.

## Completed Scope

- `internal/infrastructure/vectorstore/weaviate.go` — `WeaviateStore` с методами `Upsert`, `Delete`, `Search`, `SearchWithFilter`, `SearchWithMetadataFilter`; вспомогательные функции `uuidFromID` (UUID v5 через stdlib), `searchWithWhere`, `parseGraphQLResponse`, `whereParentIDs`, `whereMetadataFields`
- `internal/infrastructure/vectorstore/weaviate_test.go` — 6 тестов с mock HTTP server: `TestWeaviateUpsertSearch`, `TestWeaviateSearchWithFilter`, `TestWeaviateSearchWithMetadataFilter`, `TestWeaviateDeleteIdempotent`, `TestWeaviateSearchEmpty`, `TestWeaviateUuidFromID`
- `pkg/draftrag/weaviate.go` — публичный API: `WeaviateOptions`, `NewWeaviateStore`, `CreateWeaviateCollection`, `DeleteWeaviateCollection`, `WeaviateCollectionExists`
- `pkg/draftrag/weaviate_test.go` — 3 теста публичного API: `TestNewWeaviateStore_InvalidConfig`, `TestCreateWeaviateCollection`, `TestWeaviateCollectionExists`

## Acceptance

- AC-001: `TestWeaviateUpsertSearch` PASS — round-trip, Score>0, все поля Chunk корректны
- AC-002: `TestWeaviateSearchWithFilter` PASS — WHERE содержит "parentId"
- AC-003: `TestWeaviateSearchWithMetadataFilter` PASS — WHERE содержит "meta_category"
- AC-004: `TestWeaviateDeleteIdempotent` PASS — 404 и 204 оба возвращают nil
- AC-005: `TestNewWeaviateStore_InvalidConfig` PASS — `ErrInvalidVectorStoreConfig` при пустом host; `go build ./...` ok

## Notable Deviations

- Использован raw HTTP вместо официального Weaviate Go client v4 (DEC-001) — для совместимости с `httptest.Server` и паритета с Qdrant/ChromaDB.
- UUID v5 реализован через `crypto/sha1` без новых зависимостей в `go.mod` (DEC-002).
- Metadata хранится дважды: JSON-строка `chunkMetadata` + flat `meta_{key}` (DEC-003) — для server-side WHERE-фильтра без GraphQL introspection.
- Добавлен `pkg/draftrag/weaviate_test.go` как дополнительная поверхность сверх Surface Map — требуется для покрытия AC-005 на уровне публичного API.
