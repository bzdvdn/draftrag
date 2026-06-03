---
title: Source Citations
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Source Citations

Answer transparency is key for enterprise RAG. draftRAG provides two citation modes: post-hoc (`Cite`) and inline (`InlineCite`).

## 1. Basic citations (Cite)

```go
answer, result, err := pipeline.Search("What is a goroutine?").
    TopK(3).Cite(ctx)

fmt.Printf("Answer: %s\n", answer)
for i, chunk := range result.Chunks {
    fmt.Printf("[%d] Score: %.4f | %s\n",
        i+1, chunk.Score, chunk.Chunk.Content)
}
```

## 2. Inline citations (InlineCite)

```go
answer, result, citations, err := pipeline.Search("goroutine").
    TopK(3).InlineCite(ctx)

fmt.Printf("Answer:\n%s\n\n", answer)
fmt.Println("Sources:")
for _, cit := range citations {
    fmt.Printf("  [%d] relevance: %.4f\n",
        cit.Number, cit.Chunk.Score)
}
```

## 3. Parsing inline citations

```go
import "regexp"
re := regexp.MustCompile(`\[(\d+)\]`)
matches := re.FindAllStringSubmatch(answer, -1)
```

## 4. Streaming with citations

```go
ch, result, citations, err := pipeline.Search("goroutine").
    TopK(3).StreamCite(ctx)

for token := range ch {
    fmt.Print(token)
}
```

## 5. Metadata in citations

Each chunk carries `ParentID` (source document ID) and `Metadata`:

```go
for _, cit := range citations {
    fmt.Printf("[%d] doc=%s\n", cit.Number, cit.Chunk.Chunk.ParentID)
}
```

## Next

Proceed to [08-observability.md](08-observability.md) — observability with OpenTelemetry.
