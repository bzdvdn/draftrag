# Pipeline API

## Constructors

### NewPipeline

```go
func NewPipeline(store VectorStore, llm LLMProvider, embedder Embedder) (*Pipeline, error)
```

Minimal configuration: `DefaultTopK = 5`, no chunking (1 document = 1 chunk).

### NewPipelineWithChunker

```go
func NewPipelineWithChunker(store VectorStore, llm LLMProvider, embedder Embedder, chunker Chunker) (*Pipeline, error)
```

### NewPipelineWithOptions

```go
func NewPipelineWithOptions(store VectorStore, llm LLMProvider, embedder Embedder, opts PipelineOptions) (*Pipeline, error)
```

## PipelineOptions

```go
type PipelineOptions struct {
    // DefaultTopK — default number of chunks to retrieve. 0 → 5. <0 → error.
    DefaultTopK int

    // SystemPrompt — override system prompt for Answer*.
    // Empty string — built-in v1 prompt.
    SystemPrompt string

    // Chunker — if set, Index splits documents into chunks before indexing.
    Chunker Chunker

    // Hooks — observability hooks. nil → no-op.
    Hooks Hooks

    // MaxContextChars — limit for the "Context" section in the prompt (characters). 0 → no limit.
    MaxContextChars int
    // MaxContextChunks — limit on the number of chunks in context. 0 → no limit.
    MaxContextChunks int

    // DedupByParentID — deduplicate chunks by ParentID in RetrievalResult.
    DedupByParentID bool

    // MMREnabled — enable MMR reranking (context diversification).
    MMREnabled bool
    // MMRLambda — relevance/diversity balance [0..1]. 0 → 0.5.
    MMRLambda float64
    // MMRCandidatePool — how many candidates to fetch before MMR selection. 0 → topK.
    MMRCandidatePool int

    // IndexConcurrency — workers for IndexBatch. 0 → 4.
    IndexConcurrency int
    // IndexBatchRateLimit — max Embed calls/sec in IndexBatch. 0 → no limit.
    IndexBatchRateLimit int
}
```

## Indexing Methods

### Index

```go
func (p *Pipeline) Index(ctx context.Context, docs []Document) error
```

Sequential indexing. If `Chunker` is set, each document is split into chunks. An error on any one document aborts the entire operation.

### IndexBatch

```go
func (p *Pipeline) IndexBatch(ctx context.Context, docs []Document, batchSize int) (*IndexBatchResult, error)
```

Parallel indexing with `batchSize` workers. Errors on individual documents do not halt processing of the remaining documents.

```go
type IndexBatchResult struct {
    Successful     []Document       // successfully indexed
    Errors         []IndexBatchError // per-document errors
    ProcessedCount int               // total processed
}

type IndexBatchError struct {
    DocumentID string
    Error      error
}
```

```go
result, err := pipeline.IndexBatch(ctx, docs, 8)
if err != nil {
    return err // system error (context cancelled, etc.)
}
if len(result.Errors) > 0 {
    for _, e := range result.Errors {
        log.Printf("doc %s failed: %v", e.DocumentID, e.Error)
    }
}
fmt.Printf("indexed %d/%d docs\n", len(result.Successful), result.ProcessedCount)
```

## Search Builder

The primary API for search and answer generation — a fluent builder. Created via `pipeline.Search(question)`, parameters are set via method chaining, and the query is executed by a terminal method.

```go
// Create a builder
b := pipeline.Search("question")

// Parameters (all optional)
b.TopK(5)                                        // number of chunks (default: PipelineOptions.DefaultTopK)
b.ParentIDs("doc-1", "doc-2")                    // search within these documents only
b.Filter(draftrag.MetadataFilter{                // metadata filter (AND)
    Fields: map[string]string{"lang": "en"},
})
b.Hybrid(draftrag.DefaultHybridConfig())         // hybrid BM25 + semantic
```

### Terminal Methods

| Method | Returns | Description |
|---|---|---|
| `Retrieve(ctx)` | `(RetrievalResult, error)` | Search only, no generation |
| `Answer(ctx)` | `(string, error)` | RAG answer |
| `Cite(ctx)` | `(string, RetrievalResult, error)` | Answer + sources with scores |
| `InlineCite(ctx)` | `(string, RetrievalResult, []InlineCitation, error)` | Answer with `[n]` citations in text |
| `Stream(ctx)` | `(<-chan string, error)` | Streaming response token by token |
| `StreamCite(ctx)` | `(<-chan string, RetrievalResult, []InlineCitation, error)` | Streaming with inline citations |

### Examples

```go
// Simple search
result, err := pipeline.Search("question").TopK(5).Retrieve(ctx)

// RAG answer with hybrid search
answer, err := pipeline.Search("question").TopK(5).Hybrid(draftrag.DefaultHybridConfig()).Answer(ctx)

// Answer with citations, scoped to a single document
answer, sources, citations, err := pipeline.Search("question").
    TopK(5).
    ParentIDs("doc-1").
    InlineCite(ctx)

// Streaming for HTTP SSE
tokens, err := pipeline.Search("question").TopK(5).Stream(ctx)

// Streaming with citations (sources/citations available immediately)
tokens, sources, citations, err := pipeline.Search("question").TopK(5).StreamCite(ctx)
```

## Simple Methods (no parameters)

For basic usage with `DefaultTopK`:

```go
func (p *Pipeline) Query(ctx context.Context, question string) (RetrievalResult, error)
func (p *Pipeline) Answer(ctx context.Context, question string) (string, error)
func (p *Pipeline) Retrieve(ctx context.Context, question string, topK int) (RetrievalResult, error)
```

`Retrieve` implements `eval.RetrievalRunner` and is used directly in the eval harness.

## DeleteDocument

Deletes a document and all its chunks from the store by ParentID.

```go
err := pipeline.DeleteDocument(ctx, "doc-id")
if errors.Is(err, draftrag.ErrDeleteNotSupported) {
    // store does not support deletion by ParentID
}
```

Supported by all stores: **InMemoryStore**, **pgvector**, **Qdrant**, **ChromaDB**.

Each uses a native batch-delete on the `parent_id` field:
- InMemoryStore — map iteration
- pgvector — `DELETE FROM ... WHERE parent_id = $1`
- Qdrant — filter API (`must: key=parent_id`)
- ChromaDB — where filter (`{"parent_id": "..."}`)

`ErrDeleteNotSupported` is only returned for custom `VectorStore` implementations that do not implement `DocumentStore`.
