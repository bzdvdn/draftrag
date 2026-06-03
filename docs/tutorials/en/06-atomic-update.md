---
title: Atomic Document Updates
related_examples:
  - examples/pgvector/
prerequisites:
  - Go 1.23+
  - Docker
---

# Atomic Document Updates

In production, data changes: documents are edited, become stale, or get deleted. draftRAG supports document updates for stores implementing `DocumentStore`.

## 1. Start PostgreSQL with pgvector

```yaml
services:
  pgvector:
    image: pgvector/pgvector:pg16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: draftrag
```

## 2. Prepare the store

```go
dsn := "postgres://postgres:postgres@localhost:5432/draftrag?sslmode=disable"
db, _ := sql.Open("postgres", dsn)

migrateOpts := draftrag.PGVectorMigrateOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "my_docs",
        EmbeddingDimension: 768,
        CreateExtension:    true,
    },
}
draftrag.MigratePGVector(ctx, db, migrateOpts)

store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
    TableName: "my_docs", EmbeddingDimension: 768,
})
```

## 3. Index initial documents

```go
pipeline.Index(ctx, []draftrag.Document{
    {ID: "ug1", Content: "pgvector is a PostgreSQL extension for vector search."},
    {ID: "ug2", Content: "Document updates recreate all chunks."},
})
```

## 4. Update a document

```go
err := pipeline.UpdateDocument(ctx, draftrag.Document{
    ID:      "ug1",
    Content: "pgvector adds IVFFlat and HNSW indexes to PostgreSQL.",
})
```

## 5. Delete a document

```go
pipeline.DeleteDocument(ctx, "ug2")
```

## 6. Transactional updates

PGVector supports transactional document stores, so `UpdateDocument` runs in a single transaction.

## Next

Proceed to [07-citations.md](07-citations.md) — source citations in answers.
