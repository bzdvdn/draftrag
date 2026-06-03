---
title: Streaming Generation
related_examples:
  - examples/qdrant/
prerequisites:
  - Go 1.23+
  - Docker
  - Ollama (for real LLM)
---

# Streaming Generation

Streaming outputs LLM responses token by token, improving UX. draftRAG supports streaming for any `StreamingLLMProvider`.

## 1. Supported providers

| Provider | Streaming |
|----------|-----------|
| Ollama | ✓ |
| OpenAI | ✓ |
| Anthropic | ✓ |
| Mock | ✓ |

## 2. Basic streaming

```go
ch, err := pipeline.Search("What is a goroutine?").
    TopK(3).Stream(ctx)
if err != nil { log.Fatal(err) }

for token := range ch {
    fmt.Print(token)
}
fmt.Println()
```

## 3. Streaming with sources

```go
ch, result, err := pipeline.Search("goroutine").
    TopK(3).StreamSources(ctx)

fmt.Printf("Found %d sources\n", len(result.Chunks))
for token := range ch {
    fmt.Print(token)
}
```

## 4. Streaming with citations

```go
ch, result, citations, err := pipeline.Search("goroutine").
    TopK(3).StreamCite(ctx)

for token := range ch {
    fmt.Print(token)
}
for _, cit := range citations {
    fmt.Printf("[%d] score: %.4f\n", cit.Number, cit.Chunk.Score)
}
```

## 5. Interactive chat

```go
scanner := bufio.NewScanner(os.Stdin)
for {
    fmt.Print("\n> ")
    if !scanner.Scan() { break }
    q := scanner.Text()
    if q == "exit" { break }

    ch, _, _ := pipeline.Search(q).TopK(3).StreamSources(ctx)
    for token := range ch {
        fmt.Print(token)
    }
    fmt.Println()
}
```

## Next

Proceed to [06-atomic-update.md](06-atomic-update.md) — atomic document updates with pgvector.
