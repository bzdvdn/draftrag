# Advanced Features

## Citations

### Cite

Returns the answer + a list of used chunks with scores:

```go
answer, sources, err := pipeline.Search("question").TopK(5).Cite(ctx)
for i, r := range sources.Chunks {
    fmt.Printf("[%d] %s (score=%.3f)\n", i+1, r.Chunk.ParentID, r.Score)
    fmt.Printf("    %s\n", r.Chunk.Content[:100])
}
```

### InlineCite

The LLM receives a numbered context `[1] text... [2] text...` and inserts references in the response:

```go
answer, sources, citations, err := pipeline.Search("question").TopK(5).InlineCite(ctx)
// answer: "Goroutines are created with the go keyword [1]. They are multiplexed onto OS threads [1][2]."

for _, c := range citations {
    fmt.Printf("[%d] ParentID=%s Score=%.3f\n",
        c.Number,
        c.Chunk.Chunk.ParentID,
        c.Chunk.Score,
    )
}
```

`citations` contains only the chunks actually referenced in the response (the LLM may not use all sources).

---

## Streaming

Requires an LLM implementing `StreamingLLMProvider` (OpenAI-compatible, Anthropic):

```go
tokenChan, err := pipeline.Search("question").TopK(5).Stream(ctx)
if errors.Is(err, draftrag.ErrStreamingNotSupported) {
    // fallback to regular Answer
}
for token := range tokenChan {
    fmt.Print(token)
    // flush for SSE/http.Flusher in web applications
}
```

### Streaming with inline citations

```go
tokenChan, sources, citations, err := pipeline.Search("question").TopK(5).StreamCite(ctx)
// sources and citations are ready immediately (search + context building are synchronous)
// tokens are generated asynchronously
for token := range tokenChan {
    fmt.Print(token)
}
```

### HTTP Server-Sent Events

```go
func handleChat(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    flusher := w.(http.Flusher)

    tokenChan, err := pipeline.Search(r.URL.Query().Get("q")).TopK(5).Stream(r.Context())
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    for token := range tokenChan {
        fmt.Fprintf(w, "data: %s\n\n", token)
        flusher.Flush()
    }
    fmt.Fprint(w, "data: [DONE]\n\n")
    flusher.Flush()
}
```

---

## Hybrid Search (BM25 + Semantic)

Combines semantic search with keyword search. Improves recall for precise terms and abbreviations.

Requires `HybridSearcher` (pgvector or in-memory).

```go
config := draftrag.DefaultHybridConfig()
// or custom:
config := draftrag.HybridConfig{
    UseRRF:         true,  // Reciprocal Rank Fusion (recommended)
    SemanticWeight: 0.7,   // ignored when UseRRF=true
    RRFK:           60,    // RRF constant
}

result, err := pipeline.Search("question").TopK(5).Hybrid(config).Retrieve(ctx)
answer, err := pipeline.Search("question").TopK(5).Hybrid(config).Answer(ctx)
```

If the store does not support hybrid search: `errors.Is(err, draftrag.ErrHybridNotSupported)`.

### HybridConfig

| Field | Default | Description |
|---|---|---|
| `UseRRF` | `true` | Reciprocal Rank Fusion |
| `SemanticWeight` | `0.7` | Semantic weight for weighted fusion |
| `RRFK` | `60` | RRF constant: `score = 1/(k + rank)` |

---

## MMR Reranking

Maximal Marginal Relevance — selects diverse sources, balancing relevance and novelty of information:

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    MMREnabled:       true,
    MMRLambda:        0.6,   // 0 = only diversity, 1 = only relevance
    MMRCandidatePool: 20,    // request 20 candidates, select topK via MMR
    DefaultTopK:      5,
})
if err != nil {
    log.Fatal(err)
}
```

When `MMRCandidatePool > 0`, the pipeline requests `MMRCandidatePool` chunks from the store and then selects `topK` via MMR.

---

## Source Deduplication

Removes multiple chunks from the same document, keeping the most relevant one:

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DedupByParentID: true,
    DefaultTopK:     10, // request more, some will be removed by deduplication
})
if err != nil {
    log.Fatal(err)
}
```

Useful when a single document produces many chunks: without deduplication the response may reference 5 chunks from the same document.

---

## Metadata Filtering

### By ParentID (documents)

```go
// Search only in documents from a specific source
result, err := pipeline.Search("question").TopK(5).ParentIDs("doc-1", "doc-2", "doc-3").Retrieve(ctx)
answer, err := pipeline.Search("question").TopK(5).ParentIDs("doc-1", "doc-2").Answer(ctx)
```

### By Metadata

AND-filter on arbitrary document metadata fields:

```go
filter := draftrag.MetadataFilter{
    Fields: map[string]string{
        "category": "legal",
        "lang":     "en",
        "year":     "2024",
    },
}

result, err := pipeline.Search("question").TopK(5).Filter(filter).Retrieve(ctx)
answer, err := pipeline.Search("question").TopK(5).Filter(filter).Answer(ctx)
```

Requires `VectorStoreWithFilters`. If the store does not support it: `errors.Is(err, draftrag.ErrFiltersNotSupported)`.

---

## Observability Hooks

Hooks are called at each pipeline stage. Use for metrics, logging, tracing:

```go
type Hooks struct {
    OnChunkingStart  func(ctx context.Context, op string)
    OnChunkingEnd    func(ctx context.Context, op string, duration time.Duration, err error)
    OnEmbedStart     func(ctx context.Context, op string)
    OnEmbedEnd       func(ctx context.Context, op string, duration time.Duration, err error)
    OnSearchStart    func(ctx context.Context, op string)
    OnSearchEnd      func(ctx context.Context, op string, duration time.Duration, err error)
    OnGenerateStart  func(ctx context.Context, op string)
    OnGenerateEnd    func(ctx context.Context, op string, duration time.Duration, err error)
}
```

```go
hooks := draftrag.Hooks{
    OnEmbedEnd: func(ctx context.Context, op string, d time.Duration, err error) {
        metrics.EmbedDuration.Observe(d.Seconds())
        if err != nil {
            metrics.EmbedErrors.Inc()
        }
    },
    OnGenerateEnd: func(ctx context.Context, op string, d time.Duration, err error) {
        slog.InfoContext(ctx, "llm generate", "op", op, "duration", d, "err", err)
    },
}

pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Hooks: hooks,
})
if err != nil {
    log.Fatal(err)
}
```

---

## Eval Harness

Evaluates retrieval quality on a set of questions with expected sources:

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag/eval"

cases := []eval.Case{
    {
        Question:        "How do goroutines work?",
        ExpectedParents: []string{"go-goroutines", "go-concurrency"},
    },
    {
        Question:        "What are channels?",
        ExpectedParents: []string{"go-channels"},
    },
}

results, err := eval.Run(ctx, pipeline, cases, eval.Options{DefaultTopK: 5})

fmt.Printf("Hit@5: %.3f\n", results.HitAtK)
fmt.Printf("MRR:   %.3f\n", results.MRR)
```

### Metrics

| Metric | Description |
|---|---|
| `Hit@K` | Fraction of questions where at least one expected source appears in the top-K |
| `MRR` | Mean Reciprocal Rank — average reciprocal rank of the first hit |

---

## Context Limits

Limits the size of the context passed to the LLM:

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    MaxContextChars:  4000,  // no more than 4000 characters in the context section
    MaxContextChunks: 8,     // no more than 8 chunks
})
if err != nil {
    log.Fatal(err)
}
```

When the limit is exceeded, chunks are truncated by score priority (the most relevant remain).

---

## HyDE (Hypothetical Document Embeddings)

Improves recall for complex questions: the LLM first generates a hypothetical answer, then its embedding is used for retrieval (instead of the question embedding).

```go
result, err := pipeline.Search("How does Go's garbage collector work?").TopK(5).HyDE().Retrieve(ctx)
answer, err := pipeline.Search("How does Go's garbage collector work?").TopK(5).HyDE().Answer(ctx)
```

When to use: technical or highly specialized questions where the question phrasing is far from the answer phrasing in the documents.

HyDE is compatible with Cite, InlineCite, Stream, and other methods:

```go
answer, sources, err := pipeline.Search("question").TopK(5).HyDE().Cite(ctx)
```

---

## Multi-Query Retrieval

Generates N rephrasings of the question, performs retrieval for each, and merges results via Reciprocal Rank Fusion (RRF). Reduces the impact of specific phrasing on search quality.

```go
// MultiQuery(n) — number of rephrasings (recommended 2-4)
result, err := pipeline.Search("goroutines in Go").TopK(5).MultiQuery(3).Retrieve(ctx)
answer, err := pipeline.Search("goroutines in Go").TopK(5).MultiQuery(3).Answer(ctx)
```

With `n=3`, the pipeline performs 4 searches (original + 3 paraphrases) and merges via RRF (k=60):

```
score(chunk) = Σ  1 / (60 + rank_i)
              per list i
```

Chunks are sorted by descending total RRF score, top-K are selected.

Compatible with HyDE (HyDE is applied first, then MultiQuery):

```go
result, err := pipeline.Search("question").TopK(5).HyDE().MultiQuery(2).Retrieve(ctx)
```

---

## Reranker

Post-retrieval reranking — allows plugging in a cross-encoder or any other relevance scoring model. Called after vector store retrieval.

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)
}
```

Connection:

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Reranker: myReranker,
})
if err != nil {
    log.Fatal(err)
}
```

The Reranker is called automatically in all retrieval methods (Retrieve, Answer, Cite, HyDE, MultiQuery, etc.).

Example implementation (stub for testing):

```go
type scoreReranker struct{}

func (r *scoreReranker) Rerank(_ context.Context, _ string, chunks []draftrag.RetrievedChunk) ([]draftrag.RetrievedChunk, error) {
    // reorder chunks by custom logic
    sort.Slice(chunks, func(i, j int) bool {
        return chunks[i].Score > chunks[j].Score
    })
    return chunks, nil
}
```

---

## Resilience (Retry + Circuit Breaker)

Wrappers with retry logic and circuit breaker for Embedder and LLM. Protect against transient failures and cascading outages.

```go
// Defaults: MaxRetries=3, CBThreshold=5, CBTimeout=30s
embedder := draftrag.NewRetryEmbedder(
    draftrag.NewOpenAICompatibleEmbedder(...),
    draftrag.RetryOptions{},
)

llm := draftrag.NewRetryLLMProvider(
    draftrag.NewAnthropicLLM(...),
    draftrag.RetryOptions{},
)

pipeline, err := draftrag.NewPipeline(store, llm, embedder)
if err != nil {
    log.Fatal(err)
}
```

### RetryOptions

| Field | Default | Description |
|---|---|---|
| `MaxRetries` | `3` | Maximum retry attempts |
| `BaseDelay` | `100ms` | Initial delay |
| `MaxDelay` | `10s` | Maximum delay |
| `Multiplier` | `2.0` | Exponential backoff multiplier |
| `JitterFactor` | `0.25` | Random jitter fraction |
| `CBThreshold` | `5` | Error threshold to open CB |
| `CBTimeout` | `30s` | CB recovery timeout |

### Custom parameters

```go
embedder := draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{
    MaxRetries:   5,
    BaseDelay:    200 * time.Millisecond,
    CBThreshold:  10,
    CBTimeout:    60 * time.Second,
})
```

### Circuit Breaker State

```go
re := draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{})

stats := re.CircuitBreakerStats()
fmt.Printf("state=%s failures=%d\n", re.CircuitBreakerState(), stats.FailureCount)

// errors.Is(err, draftrag.ErrCircuitOpen) — CB blocked the request
```

### Error Classification

By default all errors (except context.Canceled/DeadlineExceeded) are considered retryable. Explicit marking:

```go
// Mark an error as non-retryable (will not be retried)
return draftrag.WrapNonRetryable(fmt.Errorf("invalid api key"))

// Explicitly mark as retryable
return draftrag.WrapRetryable(fmt.Errorf("service unavailable"))

// Check
if draftrag.IsRetryable(err) {
    // ...
}
```
