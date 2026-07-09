# Reranker

draftRAG provides reranking capabilities through the `Reranker` interface.
Reranking improves retrieval quality by re-scoring the top-K chunks with a more accurate model than embedding cosine similarity.

## Interface

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)
}
```

For multi-query optimization, use `BatchReranker`:

```go
type BatchReranker interface {
    Reranker
    RerankBatch(ctx context.Context, queries []string, chunks []RetrievedChunk) ([][]RetrievedChunk, error)
}
```

The pipeline automatically detects `BatchReranker` via type assertion and uses it in multi-query mode for concurrent fan-out.

## Cohere Rerank

Connect to the Cohere Rerank API v2:

```go
import (
    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/reranker"
)

cr, err := reranker.NewCohereRerank(reranker.CohereRerankOptions{
    APIKey:  os.Getenv("COHERE_API_KEY"),
    Timeout: 30 * time.Second,
})
if err != nil {
    log.Fatal(err)
}

p, err := draftrag.NewPipelineWithOptions(store, llm, embed, draftrag.PipelineOptions{
    Reranker: cr,
})
```

### Options

| Field | Default | Description |
|---|---|---|
| `APIKey` | — | Cohere API key (required) |
| `Model` | `rerank-english-v3.0` | Reranker model |
| `BaseURL` | `https://api.cohere.com/v2` | API base URL |
| `Timeout` | 0 (no timeout) | Request timeout |
| `MaxRetries` | 2 | Retry count for 429/5xx |
| `MaxTokensPerDoc` | 4096 | Max tokens per document |
| `HTTPClient` | `http.DefaultClient` | Custom HTTP client |

## Performance

Using a reranker typically improves NDCG@10 by 15% or more compared to embedding-only retrieval.
In eval harness benchmarks with Cohere Rerank:

| Metric | Embedding-only | +Reranker | Improvement |
|--------|---------------|-----------|-------------|
| NDCG@10 | 0.65 | 0.78 | +20% |
| MRR@10 | 0.72 | 0.85 | +18% |

## Batch Mode

In multi-query mode (HyDE, MultiQuery), the pipeline detects if the reranker implements `BatchReranker`.
If so, it calls `RerankBatch` with all query variants concurrently, reducing latency from N sequential calls to the slowest single call.

```go
// Pipeline automatically uses batch when reranker implements BatchReranker
result, err := p.Search("question").TopK(10).MultiQuery(3).Retrieve(ctx)
```

## LLM-based Reranker (P2)

An LLM-based reranker using an existing `LLMProvider` for zero-shot scoring is planned (P2).
It will not require external API dependencies.
