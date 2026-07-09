# Embedders

## OpenAI-compatible

Works with any API compatible with the `POST /v1/embeddings` format: OpenAI, Azure OpenAI, Together AI, Mistral, self-hosted models via LiteLLM, etc.

```go
embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "text-embedding-ada-002",
})
```

### Options

| Field | Description |
|---|---|
| `BaseURL` | **Required.** Base API URL (without `/v1/embeddings`) |
| `APIKey` | **Required.** API key (`Authorization: Bearer` header) |
| `Model` | **Required.** Model name |
| `HTTPClient` | Custom `*http.Client`. `nil` → `http.DefaultClient` |
| `Timeout` | Timeout per `Embed` call. `0` → no timeout |

### Popular models

| Provider | Model | Dimensions |
|---|---|---|
| OpenAI | `text-embedding-ada-002` | 1536 |
| OpenAI | `text-embedding-3-small` | 1536 |
| OpenAI | `text-embedding-3-large` | 3072 |
| Mistral | `mistral-embed` | 1024 |

### Configuration errors

Errors are returned from `Embed` and are comparable via `errors.Is(err, draftrag.ErrInvalidEmbedderConfig)`:
- `BaseURL` empty or missing scheme/host
- `APIKey` empty
- `Model` empty
- `Timeout < 0`

---

## Ollama

Local models via [Ollama](https://ollama.com/). No API key required.

```go
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
    Model: "nomic-embed-text",
})
```

### Options

| Field | Default | Description |
|---|---|---|
| `Model` | — | **Required.** Model name |
| `BaseURL` | `http://localhost:11434` | Ollama server URL |
| `APIKey` | `""` | Optional key (for custom instances) |
| `HTTPClient` | `http.DefaultClient` | Custom HTTP client |
| `Timeout` | `0` | Timeout per one call |

### Installing models

```bash
ollama pull nomic-embed-text     # 274M, good quality
ollama pull mxbai-embed-large    # 670M, better quality
ollama pull all-minilm           # 46M, fast
```

### Dimensions

| Model | Dimensions |
|---|---|
| `nomic-embed-text` | 768 |
| `mxbai-embed-large` | 1024 |
| `all-minilm` | 384 |

**Important**: make sure `EmbeddingDimension` in VectorStore matches the selected model's dimensions.

---

## Performance

For production loads it is recommended:

**1. IndexBatch with concurrency:**

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    IndexConcurrency:    8,   // 8 concurrent embed requests
    IndexBatchRateLimit: 100, // no more than 100 req/s
})
if err != nil {
    log.Fatal(err)
}

result, batchErr := pipeline.IndexBatch(ctx, docs, 8)
```

**2. Timeouts:**

```go
embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
    // ...
    Timeout: 10 * time.Second,
})
```

---

## CachedEmbedder

In-memory LRU cache on top of any Embedder. Repeated calls for the same text do not hit the API.

```go
embedder := draftrag.NewCachedEmbedder(
    draftrag.NewOpenAICompatibleEmbedder(...),
    draftrag.CacheOptions{MaxSize: 5000}, // 0 → 1000
)

pipeline, err := draftrag.NewPipeline(store, llm, embedder)
if err != nil {
    log.Fatal(err)
}
```

### Structured logging (optional)

For degradation events (Redis L2) you can attach a structured logger:

```go
type myLogger struct{}

func (l *myLogger) Log(ctx context.Context, level draftrag.LogLevel, msg string, fields ...draftrag.LogField) {
    // adapt to slog/zap/logrus
}

embedder := draftrag.NewCachedEmbedder(
    base,
    draftrag.CacheOptions{
        MaxSize: 5000,
        Logger:  &myLogger{},
    },
)
```

### Redis L2 (optional)

For horizontal scaling you can enable Redis as a second-level cache (L2).
Integration is done via an adapter interface — the library is not tied to any specific Redis client.

```go
type myRedisClient struct{ /* your client */ }

func (c *myRedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) { /* ... */ return nil, nil }
func (c *myRedisClient) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    /* ... */
    return nil
}

embedder := draftrag.NewCachedEmbedder(
    base,
    draftrag.CacheOptions{
        MaxSize: 5000,
        Redis: draftrag.RedisCacheOptions{
            Client:    &myRedisClient{},
            TTL:       10 * time.Minute,      // 0 → no TTL
            KeyPrefix: "myapp:embedder:",      // "" → draftrag:embedder:
        },
    },
)
```

### Statistics

```go
stats := embedder.Stats()
fmt.Printf("hits=%d misses=%d evictions=%d hit_rate=%.2f\n",
    stats.Hits, stats.Misses, stats.Evictions, stats.HitRate(),
)
```

### Compatibility with RetryEmbedder

Cache and retry can be combined. Order: cache first (don't waste retry attempts on cached requests):

```go
base := draftrag.NewOpenAICompatibleEmbedder(...)
retry := draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{})
cached := draftrag.NewCachedEmbedder(retry, draftrag.CacheOptions{})
```
