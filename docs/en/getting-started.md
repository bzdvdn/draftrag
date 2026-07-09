# Getting Started

## Installation

```bash
go get github.com/bzdvdn/draftrag
```

For pgvector:

```bash
go get github.com/jackc/pgx/v5
```

Minimum Go version: **1.23**.

## Minimal Example

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag"

embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "text-embedding-ada-002",
})
llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "gpt-4o-mini",
})

pipeline, err := draftrag.NewPipeline(draftrag.NewInMemoryStore(), llm, embedder)
if err != nil {
    log.Fatal(err)
}

pipeline.Index(ctx, []draftrag.Document{
    {ID: "doc1", Content: "..."},
})

answer, err := pipeline.Answer(ctx, "Question?")
```

## Typical Configurations

### With Chunking

By default `Index` indexes each document as a single chunk. For automatic splitting — pass a `Chunker` in options:

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DefaultTopK: 5,
    Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
        ChunkSize: 500,  // runes
        Overlap:   60,
    }),
})
if err != nil {
    log.Fatal(err)
}
```

### With pgvector

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, _ := sql.Open("pgx", "postgres://user:pass@localhost/mydb?sslmode=disable")

// Create schema (idempotent)
draftrag.MigratePGVector(ctx, db, draftrag.PGVectorMigrateOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        CreateExtension:    true,
    },
})

store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
    TableName:          "rag_chunks",
    EmbeddingDimension: 1536,
})
```

### With Qdrant

```go
store, _ := draftrag.NewQdrantStore(draftrag.QdrantOptions{
    URL:        "http://localhost:6333",
    Collection: "my_collection",
    Dimension:  1536,
})
// Create collection in advance:
draftrag.CreateCollection(ctx, draftrag.QdrantOptions{...})
```

### With Ollama (local, no API keys)

```go
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
    Model: "nomic-embed-text",  // ollama pull nomic-embed-text
})
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
    Model: "llama3.2",  // ollama pull llama3.2
})
// Default BaseURL: http://localhost:11434
```

### With Anthropic Claude

```go
llm := draftrag.NewAnthropicLLM(draftrag.AnthropicLLMOptions{
    BaseURL: "https://api.anthropic.com",
    APIKey:  "sk-ant-...",
    Model:   "claude-3-haiku-20240307",
})
```

## Next Steps

- [Concepts](../en/concepts.md) — understand the key abstractions
- [Pipeline API](../en/pipeline.md) — full method reference
- [Advanced Features](../en/advanced.md) — streaming, citations, hybrid search
