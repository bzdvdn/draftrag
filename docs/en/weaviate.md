# Weaviate

This document describes using Weaviate in draftRAG via the **public API** `pkg/draftrag`.

Important:
- This is **best-effort** documentation (no SLA/SLO guarantees).
- In production, collection preparation (schema/DDL) is typically done as a **separate deployment step** (deploy job / init container), not at service startup.

## Quick Start

Below is a minimal example: idempotent collection preparation → store creation → indexing → retrieval.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
	baseCtx := context.Background()

	// Common budget for schema/collection setup (usually larger in deployment than for queries).
	setupCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	weaviate := draftrag.WeaviateOptions{
		Host:       "localhost:8080",
		Collection: "MyChunks",
		APIKey:     os.Getenv("WEAVIATE_API_KEY"), // optional
		Timeout:    10 * time.Second,             // HTTP timeout for schema operations
	}

	// This is usually performed by a deploy job / init container.
	exists, err := draftrag.WeaviateCollectionExists(setupCtx, weaviate)
	if err != nil {
		panic(err)
	}
	if !exists {
		if err := draftrag.CreateWeaviateCollection(setupCtx, weaviate); err != nil {
			panic(err)
		}
	}

	store, err := draftrag.NewWeaviateStore(weaviate)
	if err != nil {
		// Config errors are comparable via errors.Is(err, draftrag.ErrInvalidVectorStoreConfig)
		panic(err)
	}

	embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "text-embedding-3-small",
		Timeout: 10 * time.Second,
	})
	llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-4o-mini",
		Timeout: 20 * time.Second,
	})

	pipeline, err := draftrag.NewPipeline(store, llm, embedder)
	if err != nil {
		log.Fatal(err)
	}

	indexCtx, cancel := context.WithTimeout(baseCtx, 2*time.Minute)
	defer cancel()
	if err := pipeline.Index(indexCtx, []draftrag.Document{
		{ID: "doc-1", Content: "Go supports concurrency through goroutines and channels."},
		{ID: "doc-2", Content: "Context in Go allows cancelling operations and setting deadlines."},
	}); err != nil {
		panic(err)
	}

	queryCtx, cancel := context.WithTimeout(baseCtx, 20*time.Second)
	defer cancel()

	result, err := pipeline.Search("How to cancel long-running operations in Go?").TopK(5).Retrieve(queryCtx)
	if err != nil {
		panic(err)
	}
	for i, c := range result.Chunks {
		fmt.Printf("[%d] %s (%.3f)\n", i+1, c.Chunk.ParentID, c.Score)
	}
}
```

## Collection Management (Schema)

In Weaviate, draftRAG uses a collection (class) to store chunks. Collection management is available via public functions:

- `WeaviateCollectionExists(ctx, opts)` → `bool`
- `CreateWeaviateCollection(ctx, opts)` → `error` (idempotent: "already exists" is not an error)
- `DeleteWeaviateCollection(ctx, opts)` → `error` (idempotent: 404 is not an error)

Recommendation for production:
- Perform schema/collection creation **separately from runtime** (deploy job / init);
- Use separate timeouts: schema steps should generally be larger than retrieval query timeouts.

## Capabilities and Limitations

Supported:
- basic retrieval (near-vector search) via pipeline;
- filtering by `ParentIDs(...)` (when you want to search only within a group of documents);
- metadata filters via `.Filter(...)` (if you add metadata during indexing).

Limitations:
- **Hybrid search (BM25)** is not supported for Weaviate in draftRAG (unlike pgvector).
  - **Reason:** Weaviate does not have a native BM25 implementation. Implementing it via an external index is excessive and does not align with the draftRAG philosophy (minimalism over extensibility).
  - **Recommendation:** Use pgvector or Qdrant for hybrid search if BM25+semantic is required.

## Production Checklist

### Before Deployment

- [ ] **Schema setup:** Collection created via `CreateWeaviateCollection` in a separate deploy job or init container
- [ ] **Timeouts:** Reasonable timeouts configured for schema operations (recommended 30s) and retrieval (recommended 10s)
- [ ] **Auth:** API key correctly configured for Weaviate Cloud or self-hosted instance
- [ ] **Monitoring:** Metrics configured for latency, error rates, connection pool
- [ ] **Backup:** Weaviate backups configured (if using self-hosted)

### Runtime

- [ ] **Context cancellation:** All operations use `context.Context` for cancellation and deadlines
- [ ] **Retry logic:** Add `RetryEmbedder`/`RetryLLMProvider` if needed for resilience against transient failures
- [ ] **Observability:** Use hooks from `pkg/draftrag/otel` for operation tracing
- [ ] **Resource limits:** Reasonable CPU/memory limits set for the Weaviate container

### After Deployment

- [ ] **Smoke test:** Verify basic retrieval via pipeline
- [ ] **Error handling:** Verify that 401/403/404/500 errors are handled correctly
- [ ] **Performance:** Verify retrieval latency (should be < 1s for typical queries)
- [ ] **Alerts:** Alerts configured for high error rate (> 5%) or high latency (> 5s)

## Performance Guidance

### Batch Size for Indexing

- **Recommendation:** Index documents in batches of 10–100 documents per operation
- **Why:** Batches that are too large (> 100) may cause timeout or memory pressure
- **Why:** Batches that are too small (< 10) increase HTTP request overhead

### Timeouts

| Operation | Recommended Timeout | Notes |
|-----------|-------------------|-------|
| Schema (CreateWeaviateCollection) | 30s | Schema operations are slow, allow more time |
| Retrieval (pipeline.Search) | 10s | Includes embedding + search + generate |
| WeaviateCollectionExists | 5s | Quick check |
| DeleteWeaviateCollection | 10s | Can be slow with large data volumes |

### Indexing and Performance Tuning

- **Vector dimensionality:** Use the dimension matching your embedding model (e.g., 768 for `text-embedding-3-small`)
- **Embedding caching:** Use `CachedEmbedder` to reduce load on the embedding provider
- **Filtering:** Metadata filters may slow down retrieval. Use them judiciously.
- **Connection pooling:** The Weaviate HTTP client does not support connection pooling in draftRAG — keep this in mind under high load

### Monitoring

- **Latency:** Target retrieval latency < 1s (p95)
- **Error rate:** Error rate < 1% in steady state
- **Throughput:** Baseline: 10–100 QPS depending on collection size and Weaviate configuration

## Migration Guide

### Breaking Changes

**No breaking changes in the current version.**

Functions already use consistent names with the `Weaviate*` prefix:
- `WeaviateCollectionExists`
- `CreateWeaviateCollection`
- `DeleteWeaviateCollection`

These names ensure uniqueness in the `draftrag` package (Qdrant uses no prefix).

### If You Are Using an Older Version

If you used draftRAG before v0.x and the function names differed, use the following table for migration:

| Old Name (if any) | New Name |
|------------------|----------|
| `CollectionExists` | `WeaviateCollectionExists` |
| `CreateCollection` | `CreateWeaviateCollection` |
| `DeleteCollection` | `DeleteWeaviateCollection` |

**Note:** Breaking changes are allowed before v1.0. After the v1.0 release, we will follow SemVer.

### Migration Example

```go
// Before (if used)
exists, err := draftrag.CollectionExists(ctx, opts)

// After
exists, err := draftrag.WeaviateCollectionExists(ctx, opts)
```

## Troubleshooting Guide

### Common Issues

#### 1) High latency (> 5s)

**Possible causes:**
- Timeout too large in `WeaviateOptions`
- Weaviate overloaded (high load)
- Network issues between the service and Weaviate

**Debugging steps:**
1. Check `WeaviateOptions.Timeout` — reduce to a reasonable value (10s for retrieval)
2. Check latency to Weaviate via `curl` or `ping`
3. Check Weaviate metrics (CPU, memory, QPS)
4. Reduce batch size for indexing

#### 2) Intermittent 401/403 errors

**Possible causes:**
- API key expired or invalid
- API key not set in `WeaviateOptions`
- Weaviate Cloud changed auth policy

**Debugging steps:**
1. Verify that `WeaviateOptions.APIKey` is set correctly
2. Verify API key validity in the Weaviate Cloud dashboard
3. Verify you are using the `https` scheme for Weaviate Cloud
4. Check draftRAG logs — API key redacted (verify no leak)

#### 3) Connection refused / network errors

**Possible causes:**
- Weaviate unavailable (container not running)
- Incorrect `WeaviateOptions.Host`
- Firewall blocking the connection
- Port mismatch (Weaviate on 8080, you are connecting to a different port)

**Debugging steps:**
1. Check that Weaviate is running: `curl http://localhost:8080/v1/.well-known/ready`
2. Check `WeaviateOptions.Host` — should be `host:port` (without scheme)
3. Check firewall rules
4. Check Weaviate container logs

#### 4) 422 error when creating a collection

**Possible causes:**
- Collection already exists (normal, idempotent)
- Schema conflict (a collection with a different schema already exists)

**Debugging steps:**
1. Check collection existence via `WeaviateCollectionExists`
2. If the collection exists with a different schema — delete it via `DeleteWeaviateCollection` and recreate
3. Check Weaviate logs for error details

#### 5) Empty results from retrieval

**Possible causes:**
- Collection is empty (no indexed documents)
- Filters are too restrictive
- Embedding dimension mismatch

**Debugging steps:**
1. Verify documents are indexed via pipeline.Index
2. Remove filters and try retrieval without them
3. Verify that the embedding dimension in `WeaviateOptions.Dimension` matches the model

### Debugging Tips

#### Enabling verbose logging

Use hooks for operation logging:

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag/otel"

// Add hooks for observability
pipeline.WithHooks(
    draftragotel.NewEmbeddingHook(),
    draftragotel.NewSearchHook(),
    draftragotel.NewGenerationHook(),
)
```

#### Testing locally

Use Docker Compose for a local Weaviate instance:

```bash
docker run -d -p 8080:8080 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  semitechnologies/weaviate:latest
```

#### Checking schema

Check the schema via the Weaviate API:

```bash
curl http://localhost:8080/v1/schema
```

## Common Errors

### 1) 404 / collection missing

**Symptoms**
- Weaviate errors about class/collection not found.

**Checks**
- collection created before service startup (`WeaviateCollectionExists/CreateWeaviateCollection`);
- `WeaviateOptions.Collection` matches the actual class name;
- `Host` points to the correct instance.

### 2) 401/403 / auth

**Symptoms**
- Authorization errors on Weaviate Cloud / a protected instance.

**Checks**
- `WeaviateOptions.APIKey` is valid and being sent (Bearer token);
- you are using `https` (if required by the instance) via `WeaviateOptions.Scheme`.

### 3) `context deadline exceeded` / timeouts

**Symptoms**
- Timeouts during schema operations or retrieval.

**Checks**
- schema steps (`CreateWeaviateCollection`) use a separate `context.WithTimeout` and `WeaviateOptions.Timeout`;
- retrieval path uses a separate `context.WithTimeout` and a reasonable budget for embed/search/generate.

## Links

- Compatibility and support policy: `../en/compatibility.md`
- Vector stores overview: `../en/vector-stores.md`
