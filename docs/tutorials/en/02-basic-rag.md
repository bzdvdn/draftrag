---
title: Basic RAG with Qdrant
related_examples:
  - examples/qdrant/
prerequisites:
  - Go 1.23+
  - Docker
  - Ollama (optional)
---

# Basic RAG with Qdrant

Replace the in-memory store with Qdrant — a production-ready vector database. Learn to manage collections and use docker-compose for infrastructure.

## 1. Start Qdrant

```yaml
# docker-compose.yml
services:
  qdrant:
    image: qdrant/qdrant:v1.12.4
    ports:
      - "6333:6333"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:6333/health"]
      interval: 5s
      retries: 10
```

```bash
docker compose up -d
```

## 2. Manage collections

```go
opts := draftrag.QdrantOptions{
    URL:        "http://localhost:6333",
    Collection: "my_rag_docs",
    Dimension:  768,
}

exists, err := draftrag.CollectionExists(ctx, opts)
if !exists {
    draftrag.CreateCollection(ctx, opts)
}

store, err := draftrag.NewQdrantStore(opts)
```

## 3. Index and query

```go
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{Model: "llama3.2"})
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{Model: "nomic-embed-text"})

pipeline := draftrag.NewPipelineWithChunker(store, llm, embedder,
    draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{ChunkSize: 1000, Overlap: 100}))

pipeline.Index(ctx, []draftrag.Document{
    {ID: "q1", Content: "Qdrant supports metadata filtering and pagination."},
})

answer, result, _ := pipeline.Search("What does Qdrant support?").TopK(3).Cite(ctx)
```

## 4. Interactive mode

```go
scanner := bufio.NewScanner(os.Stdin)
for {
    fmt.Print("Question (or 'exit'): ")
    if !scanner.Scan() { break }
    q := scanner.Text()
    if q == "exit" { break }
    answer, _, _ := pipeline.Search(q).TopK(3).Cite(ctx)
    fmt.Printf("Answer: %s\n", answer)
}
```

## Next

Proceed to [03-hybrid-search.md](03-hybrid-search.md) — hybrid search with Weaviate.
