# LLM Providers

## OpenAI-compatible (Responses API)

Works with OpenAI Responses API (`POST /v1/responses`) and compatible providers.

```go
llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "gpt-4o-mini",
})
```

Implements `LLMProvider` and `StreamingLLMProvider`.

### Options

| Field | Description |
|---|---|
| `BaseURL` | **Required.** Base URL (without `/v1/responses`) |
| `APIKey` | **Required.** API key |
| `Model` | **Required.** Model name |
| `Temperature` | `*float64`. `nil` → not sent in the request |
| `MaxOutputTokens` | `*int`. `nil` → not sent |
| `HTTPClient` | `nil` → `http.DefaultClient` |
| `Timeout` | Timeout for `Generate`. Does not apply to `GenerateStream` |

### Streaming

```go
// type assertion to get streaming capability
streamingLLM, ok := llm.(draftrag.StreamingLLMProvider)
if !ok {
    // not supported
}
ch, err := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
```

Via Pipeline:

```go
tokenChan, err := pipeline.AnswerStream(ctx, "question", 5)
if errors.Is(err, draftrag.ErrStreamingNotSupported) {
    // LLM does not support streaming
}
for token := range tokenChan {
    fmt.Print(token)
}
```

---

## Anthropic Claude

Native support for [Anthropic Messages API](https://docs.anthropic.com/en/api/messages).

```go
llm := draftrag.NewAnthropicLLM(draftrag.AnthropicLLMOptions{
    BaseURL: "https://api.anthropic.com",
    APIKey:  "sk-ant-...",
    Model:   "claude-3-haiku-20240307",
})
```

Implements `LLMProvider` and `StreamingLLMProvider`.

### Options

| Field | Default | Description |
|---|---|---|
| `BaseURL` | — | **Required.** `https://api.anthropic.com` |
| `APIKey` | — | **Required.** API key (`X-API-Key` header) |
| `Model` | `claude-3-haiku-20240307` | Model (empty string → default) |
| `AnthropicVersion` | `2023-06-01` | API version (`anthropic-version` header) |
| `Temperature` | `nil` | `*float64` |
| `MaxTokens` | `nil` (→ 1024) | `*int` |
| `HTTPClient` | `http.DefaultClient` | |
| `Timeout` | `0` | Timeout for `Generate` |

### Popular models

| Model | Speed | Quality |
|---|---|---|
| `claude-3-haiku-20240307` | Fast | Good |
| `claude-3-5-sonnet-20241022` | Medium | Excellent |
| `claude-opus-4-6` | Slow | Maximum |

---

## Ollama

Local LLM models via [Ollama](https://ollama.com/).

```go
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
    Model: "llama3.2",
})
```

### Options

| Field | Default | Description |
|---|---|---|
| `Model` | — | **Required.** |
| `BaseURL` | `http://localhost:11434` | Ollama server URL |
| `APIKey` | `""` | Optional (for authorization) |
| `Temperature` | `nil` | `*float64` |
| `MaxTokens` | `nil` | `*int` |
| `HTTPClient` | `http.DefaultClient` | |
| `Timeout` | `0` | Timeout for `Generate` |

### Installing models

```bash
ollama pull llama3.2        # 2B, fast
ollama pull llama3.1:8b     # 8B, good quality
ollama pull mistral         # 7B
ollama pull qwen2.5:7b      # good for Russian
```

**Ollama does not support streaming** via draftRAG. `ErrStreamingNotSupported` when using `AnswerStream`.

---

## Custom LLM

Implement the interface for any provider:

```go
type MyLLM struct{}

func (m *MyLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
    // your implementation
}

pipeline, err := draftrag.NewPipeline(store, &MyLLM{}, embedder)
if err != nil {
    log.Fatal(err)
}
```

For streaming, additionally implement `GenerateStream(ctx, systemPrompt, userMessage) (<-chan string, error)`.
