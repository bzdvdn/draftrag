---
title: Metadata and Filters
related_examples:
  - examples/chromadb/
prerequisites:
  - Go 1.23+
  - Docker
---

# Metadata and Filters

Real documents carry metadata: author, date, category, tags. draftRAG supports metadata filtering for precise answers.

## 1. Start ChromaDB

```yaml
services:
  chromadb:
    image: chromadb/chroma:0.5.20
    ports:
      - "8000:8000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/api/v1/heartbeat"]
```

```bash
docker compose up -d
```

## 2. Create a collection

```go
opts := draftrag.ChromaDBOptions{
    BaseURL:    "http://localhost:8000",
    Collection: "FilterDemo",
    Dimension:  768,
}
draftrag.CreateChromaCollection(ctx, opts)
store, _ := draftrag.NewChromaDBStore(opts)
```

## 3. Index with metadata

```go
docs := []draftrag.Document{
    {
        ID:      "f1",
        Content: "Go 1.21 introduced improved error handling.",
        Metadata: map[string]string{
            "category": "release",
            "version":  "1.21",
            "year":     "2023",
        },
    },
    {
        ID:      "f2",
        Content: "Generics in Go enable type-safe collections.",
        Metadata: map[string]string{
            "category": "feature",
            "version":  "1.18",
            "year":     "2022",
        },
    },
}
pipeline.Index(ctx, docs)
```

## 4. Filter by metadata

```go
filter := draftrag.MetadataFilter{
    Fields: map[string]string{"category": "feature"},
}
result, _ := pipeline.Search("type-safe collections").
    TopK(5).Filter(filter).Retrieve(ctx)
```

## 5. Combine filters

Multiple `Fields` are joined with AND:

```go
filter := draftrag.MetadataFilter{
    Fields: map[string]string{
        "category": "feature",
        "year":     "2022",
    },
}
```

## Next

Proceed to [05-streaming.md](05-streaming.md) — streaming generation.
