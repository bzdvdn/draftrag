---
title: Hybrid Search with Weaviate
related_examples:
  - examples/weaviate/
prerequisites:
  - Go 1.23+
  - Docker
---

# Hybrid Search with Weaviate

Hybrid search combines semantic (vector) and keyword (BM25) ranking. Weaviate supports this natively.

## 1. Start Weaviate

```yaml
services:
  weaviate:
    image: semitechnologies/weaviate:1.27.5
    ports:
      - "8080:8080"
    environment:
      ENABLE_MODULES: ""
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: "true"
```

```bash
docker compose up -d
```

## 2. Create a collection

```go
opts := draftrag.WeaviateOptions{
    Host:       "localhost:8080",
    Scheme:     "http",
    Collection: "HybridDemo",
}
draftrag.CreateWeaviateCollection(ctx, opts)
store, _ := draftrag.NewWeaviateStore(opts)
```

## 3. Index documents

```go
docs := []draftrag.Document{
    {ID: "h1", Content: "Weaviate is an open-source vector database."},
    {ID: "h2", Content: "BM25 ranks documents based on term frequency."},
    {ID: "h3", Content: "Hybrid search combines vector and keyword relevance."},
}
pipeline.Index(ctx, docs)
```

## 4. Compare vector vs hybrid search

```go
vectorResult, _ := pipeline.Search("ranking algorithm").TopK(3).Retrieve(ctx)
fmt.Printf("Vector search: %d results\n", len(vectorResult.Chunks))

hybridResult, _ := pipeline.Search("ranking algorithm").
    TopK(3).
    Hybrid(draftrag.DefaultHybridConfig()).
    Retrieve(ctx)
fmt.Printf("Hybrid search: %d results\n", len(hybridResult.Chunks))
```

## 5. Tune HybridConfig

```go
cfg := draftrag.HybridConfig{
    SemanticWeight: 0.7,
    UseRRF:         true,
    RRFK:           60,
}
result, _ := pipeline.Search("hybrid search").TopK(5).Hybrid(cfg).Retrieve(ctx)
```

## Next

Proceed to [04-metadata-filter.md](04-metadata-filter.md) — metadata filtering with ChromaDB.
