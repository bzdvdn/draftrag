---
title: Production Hardening
related_examples:
  - examples/memory/
  - examples/pgvector/
  - examples/qdrant/
prerequisites:
  - Go 1.23+
  - Docker
---

# Production Hardening

This tutorial combines all previous topics into a production-ready pipeline. Organized into subsections for navigation.

<!-- Anchor links -->
<a name="sec-10-1"></a>
<a name="sec-10-2"></a>
<a name="sec-10-3"></a>
<a name="sec-10-4"></a>
<a name="sec-10-5"></a>

## 10.1 Retry and Circuit Breaker

```go
retryOpts := draftrag.RetryOptions{
    MaxRetries:   3,
    BaseDelay:    100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    JitterFactor: 0.25,
    CBThreshold:  5,
    CBTimeout:    30 * time.Second,
}

llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{Model: "llama3.2"})
retryLLM := draftrag.NewRetryLLMProvider(llm, retryOpts)

embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{Model: "nomic-embed-text"})
retryEmbedder := draftrag.NewRetryEmbedder(embedder, retryOpts)
```

## 10.2 Observability (OTel)

```go
hooks, _ := otel.NewHooks(otel.HooksOptions{
    TracerProvider: tp,
    MeterProvider:  mp,
})

pipeline := draftrag.NewPipelineWithOptions(store, retryLLM, retryEmbedder,
    draftrag.PipelineOptions{
        Chunker:                chunker,
        Hooks:                  hooks,
        MaxContextChars:        4000,
        DedupSourcesByParentID: true,
    })
```

## 10.3 Embedding Cache

```go
cachedEmbedder, _ := draftrag.NewCachedEmbedder(retryEmbedder, draftrag.CacheOptions{
    MaxSize: 10000,
})

stats := cachedEmbedder.Stats()
fmt.Printf("Hit rate: %d/%d\n", stats.Hits, stats.Misses)
```

## 10.4 Rate Limiting and Concurrency

```go
pipeline := draftrag.NewPipelineWithOptions(store, retryLLM, retryEmbedder,
    draftrag.PipelineOptions{
        Chunker:                    chunker,
        IndexConcurrency:           4,
        IndexBatchRateLimit:        10,
        MMRCandidatePool:           20,
        MMREnabled:                 true,
        MMRLambda:                  0.5,
    })

result, _ := pipeline.IndexBatch(ctx, allDocs, 10)
```

## 10.5 Secret Redaction

```go
import "github.com/bzdvdn/draftrag/internal/domain"

safeMsg := domain.RedactSecret(
    "connection failed with api_key=sk-abc123",
    "sk-abc123",
)
// safeMsg: "connection failed with api_key=***"
```

## Complete production pipeline

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
)

func main() {
    ctx := context.Background()
    store := draftrag.NewInMemoryStore()

    llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{Model: "llama3.2"})
    retryLLM := draftrag.NewRetryLLMProvider(llm, draftrag.RetryOptions{
        MaxRetries: 3, CBThreshold: 5, CBTimeout: 30 * time.Second,
    })

    embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{Model: "nomic-embed-text"})
    retryEmb := draftrag.NewRetryEmbedder(embedder, defaultRetry())
    cachedEmb, _ := draftrag.NewCachedEmbedder(retryEmb, draftrag.CacheOptions{MaxSize: 5000})

    hooks, _ := otel.NewHooks(otel.HooksOptions{})

    pipeline := draftrag.NewPipelineWithOptions(store, retryLLM, cachedEmb,
        draftrag.PipelineOptions{
            Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
                ChunkSize: 1000, Overlap: 100,
            }),
            Hooks:               hooks,
            MaxContextChars:     4000,
            DedupSourcesByParentID: true,
            MMREnabled:          true,
        })

    pipeline.Index(ctx, []draftrag.Document{
        {ID: "prod1", Content: "Production RAG needs monitoring, retry, and caching."},
    })
    answer, _, _ := pipeline.Search("production RAG").TopK(3).Cite(ctx)
    log.Printf("Answer: %s", answer)
}

func defaultRetry() draftrag.RetryOptions {
    return draftrag.RetryOptions{
        MaxRetries: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second,
        Multiplier: 2.0, JitterFactor: 0.25, CBThreshold: 5, CBTimeout: 30 * time.Second,
    }
}
```

## Summary

You've completed the full draftRAG tutorial series:

1. [01-quickstart.md](01-quickstart.md) — Quickstart with mock
2. [02-basic-rag.md](02-basic-rag.md) — Basic RAG with Qdrant
3. [03-hybrid-search.md](03-hybrid-search.md) — Hybrid search with Weaviate
4. [04-metadata-filter.md](04-metadata-filter.md) — Metadata filtering
5. [05-streaming.md](05-streaming.md) — Streaming generation
6. [06-atomic-update.md](06-atomic-update.md) — Document updates
7. [07-citations.md](07-citations.md) — Source citations
8. [08-observability.md](08-observability.md) — Observability
9. [09-evaluation.md](09-evaluation.md) — Evaluation
10. [10-production-hardening.md](10-production-hardening.md) — Production hardening (this file)
