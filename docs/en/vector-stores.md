# Vector Stores

## In-Memory

```go
store := draftrag.NewInMemoryStore()
```

Stores data in the process memory. No external dependencies required. Suitable for prototyping, tests, and small document sets (up to ~10k chunks).

**Supports**: `VectorStore`, `VectorStoreWithFilters`, `HybridSearcher`  
**Does not persist data** between restarts.

---

## pgvector (PostgreSQL)

Production-ready store with full SQL filtering and hybrid search support.

### Dependencies

```bash
go get github.com/jackc/pgx/v5
```

PostgreSQL 15+ with pgvector extension.

### Migrations

Before first use, create the schema:

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/bzdvdn/draftrag/pkg/draftrag"
)

db, err := sql.Open("pgx", "postgres://user:pass@localhost/db?sslmode=disable")

err = draftrag.MigratePGVector(ctx, db, draftrag.PGVectorMigrateOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        CreateExtension:    true, // CREATE EXTENSION IF NOT EXISTS vector
    },
})
```

`MigratePGVector` is idempotent — safe to call on every startup. For production, apply SQL files from `pkg/draftrag/migrations/pgvector/` as a separate deployment step.

### Creating a store

```go
store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
    TableName:          "rag_chunks",
    EmbeddingDimension: 1536,
})

// Or with runtime limits:
store := draftrag.NewPGVectorStoreWithOptions(db, draftrag.PGVectorStoreOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        IndexMethod:        "hnsw",  // "ivfflat" (default) or "hnsw"
        Lists:              100,     // ivfflat parameter
    },
    Runtime: draftrag.PGVectorRuntimeOptions{
        SearchTimeout:   2 * time.Second,
        UpsertTimeout:   5 * time.Second,
        MaxTopK:         50,
        MaxContentBytes: 64 * 1024,
    },
})
```

Migration:

- was: `NewPGVectorStoreWithRuntimeOptions(db, opts, runtime)` (deprecated)
- now: `NewPGVectorStoreWithOptions(db, PGVectorStoreOptions{PGVectorOptions: opts, Runtime: runtime})`

### PGVectorOptions

| Field | Default | Description |
|---|---|---|
| `TableName` | `draftrag_chunks` | Table name |
| `EmbeddingDimension` | — | **Required.** Vector dimension |
| `CreateExtension` | `false` | Create pgvector extension |
| `IndexMethod` | `ivfflat` | Index method: `ivfflat` or `hnsw` |
| `Lists` | `100` | ivfflat parameter |

### Docker Compose (quick start)

```yaml
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypass
      POSTGRES_DB: mydb
    ports:
      - "5432:5432"
```

**Supports**: `VectorStore`, `VectorStoreWithFilters`, `HybridSearcher`

---

## Qdrant

```go
opts := draftrag.QdrantOptions{
    URL:        "http://localhost:6333",  // default
    Collection: "my_collection",          // required
    Dimension:  1536,                     // required
    Timeout:    10 * time.Second,         // for collection operations
}

store, err := draftrag.NewQdrantStore(opts)
```

### Collection management

```go
// Check existence
exists, err := draftrag.CollectionExists(ctx, opts)

// Create
err = draftrag.CreateCollection(ctx, opts)

// Delete
err = draftrag.DeleteCollection(ctx, opts)
```

### Docker

```bash
docker run -d -p 6333:6333 qdrant/qdrant
```

### Qdrant Cloud

Pass the cloud instance URL in `QdrantOptions.URL`.

**Supports**: `VectorStore`, `VectorStoreWithFilters`

---

## Weaviate

Weaviate-backed store via the public `pkg/draftrag` API.

Status: **experimental** (see `../en/compatibility.md`). Detailed setup and troubleshooting: `../en/weaviate.md`.

```go
opts := draftrag.WeaviateOptions{
    Host:       "localhost:8080", // required
    Scheme:     "http",           // "" → http
    Collection: "MyChunks",       // required (Weaviate class)
    APIKey:     "",               // optional (Weaviate Cloud)
    Timeout:    10 * time.Second, // HTTP timeout for schema operations
}

store, err := draftrag.NewWeaviateStore(opts)
```

### Collection management (production recommendation)

In production, schema/collection creation is typically done as a separate deployment step (deploy job/init container), not at service startup:

```go
exists, err := draftrag.WeaviateCollectionExists(ctx, opts)
if !exists {
    err = draftrag.CreateWeaviateCollection(ctx, opts)
}
```

**Supports**: `VectorStore`, `VectorStoreWithFilters`

---

## ChromaDB

```go
store, err := draftrag.NewChromaDBStore(draftrag.ChromaDBOptions{
    BaseURL:    "http://localhost:8000",  // default
    Collection: "my_collection",          // required
    Dimension:  1536,                     // required
})
```

The collection must be created in advance via the ChromaDB API or client.

### Docker

```bash
docker run -d -p 8000:8000 chromadb/chroma
```

**Supports**: `VectorStore`, `VectorStoreWithFilters`

---

<!-- @sk-task api-consistency-pass#T3.5: docs sync — Milvus section (DEC-008, RQ-008, AC-013) -->

## Milvus / Zilliz

High-performance distributed vector search (REST API).

**Status**: ⚠️ public API in development (`pkg/draftrag.NewMilvusStore` not yet available, see `internal/infrastructure/vectorstore/milvus.go`).

Detailed setup and troubleshooting: `../en/milvus.md` (planned).

```go
// Internal API (not recommended for direct use):
store, err := vectorstore.NewMilvusStore(baseURL, collection, token)
```

The collection must be created in advance via the Milvus API or client. Supports basic retrieval, metadata filters, and ParentID. **Hybrid search is not supported** in the public model.

### Docker (quick start)

```bash
docker run -d -p 19530:19530 -p 9091:9091 milvusdb/milvus:latest
```

**Supports** (internal API): `VectorStore`, `VectorStoreWithFilters`

---

## Capability table

<!-- @sk-task docs-and-examples#T3.6: links to examples/ in the first column of capability table (AC-015) -->
<!-- @sk-task api-consistency-pass#T3.5: docs sync — capability table 6×6 = 36 cells (DEC-008, RQ-008, AC-014) -->

| | [In-Memory](examples/memory/) | [pgvector](examples/pgvector/) | [Qdrant](examples/qdrant/) | [ChromaDB](examples/chromadb/) | [Weaviate](examples/weaviate/) | [Milvus](examples/milvus/) |
|---|---|---|---|---|---|---|
| **Basic retrieval** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Metadata filter** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **ParentID filter** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Hybrid (BM25 + semantic)** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌[^hybrid] |
| **DeleteByParentID** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Collection management** | N/A | ✅ (SQL migrations) | ✅ | ✅ | ✅ | ✅ |

**Legend**:
- ✅ — supported
- ❌ — not supported (see footnote for exceptions)
- N/A — not applicable (no collections/schemas)

**Footnotes**:

[^hybrid]: Hybrid search (BM25) is not supported at the public API level for Weaviate and Milvus. Internal implementations in `internal/infrastructure/vectorstore/{weaviate,milvus}.go` contain `SearchHybrid` methods but are not exposed in `pkg/draftrag/`. Use pgvector (production-ready) or In-Memory (prototyping) for hybrid search.

**Incompatible combinations**:

- `Hybrid(cfg)` in SearchBuilder + Weaviate/Milvus/Qdrant/ChromaDB → `draftrag.ErrHybridNotSupported` (at runtime).
- `UpdateDocument` + In-Memory/Qdrant/ChromaDB/Weaviate/Milvus → `draftrag.ErrUpdateNotAtomic` (no transactional store; T3.2 best-effort path). Only pgvector guarantees atomicity.

---

## Production status

| Store | Status |
|---|---|
| In-Memory | for tests and prototypes |
| pgvector | production-ready |
| Qdrant | production-ready |
| ChromaDB | production-ready |
| Weaviate | ⚠️ experimental (see `../en/compatibility.md` and `../en/weaviate.md`) |
| Milvus | ⚠️ public API in development (see `internal/infrastructure/vectorstore/milvus.go`) |
