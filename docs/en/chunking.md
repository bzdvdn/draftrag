# Chunking

Chunking is splitting a document into fragments before indexing. Without a chunker, each `Document` is indexed as a single chunk.

## BasicChunker

A deterministic rune-based chunker with overlap and MaxChunks limit support.

```go
chunker := draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
    ChunkSize: 500,  // runes (not bytes)
    Overlap:   60,   // overlap between chunks
    MaxChunks: 0,    // 0 = no limit
})

pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Chunker: chunker,
})
if err != nil {
    log.Fatal(err)
}
```

### BasicChunkerOptions

| Field | Description |
|---|---|
| `ChunkSize` | **Required > 0.** Target chunk size in runes |
| `Overlap` | Overlap in runes. `>= 0` and `< ChunkSize` |
| `MaxChunks` | Max chunks. `0` → no limit |

### How overlap works

```
Document: [AAAAAA BBBBBB CCCCCC]
ChunkSize=6, Overlap=2:

Chunk 0: [AAAAAA]
Chunk 1:   [AABB BB]   (starts with last 2 runes of previous)
Chunk 2:       [BBCCCC]
```

Overlap helps preserve context at chunk boundaries — a sentence split between two chunks will partially appear in each.

### Recommended parameters

| Scenario | ChunkSize | Overlap |
|---|---|---|
| Technical texts | 400–600 | 50–80 |
| Long articles | 800–1200 | 100–150 |
| FAQ, short answers | 200–300 | 30–50 |
| Code | 300–500 | 0–30 |

### MaxChunks

Limits the number of chunks from a single document. Useful for very long documents to avoid creating an excessive index:

```go
chunker := draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
    ChunkSize: 500,
    Overlap:   60,
    MaxChunks: 20,  // at most 20 chunks per document
})
```

### Configuration errors

Errors are returned from `Chunk`, comparable via `errors.Is(err, draftrag.ErrInvalidChunkerConfig)`:
- `ChunkSize <= 0`
- `Overlap < 0`
- `Overlap >= ChunkSize`
- `MaxChunks < 0`

## Custom Chunker

```go
type MyChunker struct{}

func (c *MyChunker) Chunk(ctx context.Context, doc draftrag.Document) ([]draftrag.Chunk, error) {
    // split doc.Content into fragments
    var chunks []draftrag.Chunk
    for i, part := range splitByParagraph(doc.Content) {
        chunks = append(chunks, draftrag.Chunk{
            ID:       fmt.Sprintf("%s:%d", doc.ID, i),
            Content:  part,
            ParentID: doc.ID,
            Position: i,
            Metadata: doc.Metadata,
        })
    }
    return chunks, nil
}
```

## How Pipeline uses Chunker

When `Index(ctx, docs)` is called:

1. If `Chunker == nil`: each `Document` → one `Chunk` (Content = Document.Content)
2. If `Chunker` is set: `Chunker.Chunk(ctx, doc)` → `[]Chunk`, then each chunk is embedded and stored

`IndexBatch` uses the same Chunker, but in parallel.
