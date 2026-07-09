---
title: Quickstart
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Quickstart with draftRAG

Build your first RAG system in 5 minutes. Uses in-memory vector store and mock LLM — no external dependencies required.

## 1. Create a project

```bash
mkdir my-first-rag && cd my-first-rag
go mod init my-first-rag
go get github.com/bzdvdn/draftrag@latest
```

## 2. Write the code

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
    ctx := context.Background()

    store := draftrag.NewInMemoryStore()

    embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
        Model: "nomic-embed-text",
    })
    llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
        Model: "llama3.2",
    })

    pipeline, err := draftrag.NewPipelineWithChunker(
        store, llm, embedder,
        draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
            ChunkSize: 1000,
            Overlap:   100,
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    docs := []draftrag.Document{
        {ID: "doc1", Content: "Go is a statically typed, compiled programming language."},
        {ID: "doc2", Content: "Goroutines are lightweight threads started with the go keyword."},
        {ID: "doc3", Content: "Channels in Go are used for goroutine synchronization."},
    }
    if err := pipeline.Index(ctx, docs); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Indexing complete")

    answer, result, err := pipeline.Search("What is a goroutine?").TopK(3).Cite(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Question: What is a goroutine?\nAnswer: %s\nSources: %d\n",
        answer, len(result.Chunks))
}
```

## 3. Run

```bash
go run .
```

## Next

Try a real LLM — install [Ollama](https://ollama.ai) and set environment variables:

```bash
export LLM_PROVIDER=ollama
export OLLAMA_LLM_MODEL=llama3.2
export OLLAMA_EMBED_MODEL=nomic-embed-text
```

Then proceed to [02-basic-rag.md](02-basic-rag.md) — connect Qdrant for persistent storage.
