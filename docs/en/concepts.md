# Concepts

## Architecture Overview

draftRAG consists of four interface components and a Pipeline that connects them:

```
Document → [Chunker] → Chunk → [Embedder] → vector → [VectorStore]
                                                              ↓
Question → [Embedder] → vector → [VectorStore].Search → Chunks → [LLM] → Answer
```

## Document

Indexing unit. Contains text and optional metadata.

```go
type Document struct {
    ID       string            // unique identifier
    Content  string            // text content (required)
    Metadata map[string]string // arbitrary fields for filtering
}
```

`ID` is used as `ParentID` for all chunks created from this document. This allows linking chunks to their source and filtering search by documents.

## Chunk

A document fragment that is actually stored in VectorStore.

```go
type Chunk struct {
    ID        string            // unique chunk ID (typically "docID:position")
    Content   string            // fragment text
    ParentID  string            // parent document ID
    Position  int               // ordinal position in the document
    Embedding []float64         // vector (filled by Pipeline)
    Metadata  map[string]string // metadata (inherited from Document)
}
```

## VectorStore

Stores chunks and performs vector search.

```go
type VectorStore interface {
    Upsert(ctx context.Context, chunk Chunk) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, embedding []float64, topK int) (RetrievalResult, error)
}
```

Extended capabilities are implemented through additional interfaces:

| Interface | What it adds |
|---|---|
| `VectorStoreWithFilters` | `SearchWithFilter` (by ParentID), `SearchWithMetadataFilter` |
| `HybridSearcher` | `SearchHybrid` (BM25 + semantic) |

Pipeline automatically detects supported capabilities via type assertion.

## Embedder

Converts text into a numeric vector.

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float64, error)
}
```

The same Embedder is used for both indexing (document → vector) and search (question → vector). **Important**: use the same model for both operations.

## LLMProvider

Generates a response based on a system prompt and user message.

```go
type LLMProvider interface {
    Generate(ctx context.Context, systemPrompt, userMessage string) (string, error)
}
```

For streaming support:

```go
type StreamingLLMProvider interface {
    LLMProvider
    GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error)
}
```

Pipeline checks whether the LLM implements `StreamingLLMProvider` and returns `ErrStreamingNotSupported` if not.

## Chunker

Splits a Document into Chunks.

```go
type Chunker interface {
    Chunk(ctx context.Context, doc Document) ([]Chunk, error)
}
```

If `Chunker` is not set in `PipelineOptions`, each document is indexed as a single chunk.

## Pipeline

Orchestrates all components. Created once, used repeatedly.

```go
// Minimal configuration
pipeline, err := draftrag.NewPipeline(store, llm, embedder)
if err != nil {
    // error handling
}

// Full configuration
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DefaultTopK:      5,
    Chunker:          myChunker,
    SystemPrompt:     "You are a helpful assistant...",
    MaxContextChars:  4000,
    MaxContextChunks: 10,
    DedupByParentID:  true,
    MMREnabled:       true,
    MMRLambda:        0.6,
    Hooks:            myHooks,
})
if err != nil {
    // error handling
}
```

## RetrievalResult

Returned by `Query*` methods and passed in responses with citations.

```go
type RetrievalResult struct {
    Chunks     []RetrievedChunk
    TotalFound int
}

type RetrievedChunk struct {
    Chunk Chunk
    Score float64  // cosine similarity [0, 1] or RRF score for hybrid
}
```

## Error Handling

All public errors are sentinel values, comparable via `errors.Is`:

```go
var (
    ErrEmptyDocument            // empty document during indexing
    ErrEmptyQuery               // empty question
    ErrInvalidTopK              // topK <= 0
    ErrNilContext               // nil context in public method
    ErrFiltersNotSupported      // store does not support filters
    ErrStreamingNotSupported    // LLM does not support streaming
    ErrHybridNotSupported       // store does not support hybrid search
    ErrEmbeddingDimensionMismatch
    ErrInvalidEmbedderConfig
    ErrInvalidLLMConfig
    ErrInvalidChunkerConfig
    ErrInvalidVectorStoreConfig
)
```
