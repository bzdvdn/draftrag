# Memory example

In-memory RAG example based on Go documentation. Does not require Docker or external services.

## Quick start

```bash
cd examples/memory && cp .env.example .env && go run .
```

## Environment variables

Basic (from `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `mock` | LLM provider (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Embedding dimension |

For a real LLM (ollama/openai/anthropic), additional variables are required — see [examples/shared/config.go](../shared/config.go).

## What the example does

1. Creates an in-memory vector store
2. Indexes 10 documents about Go (goroutines, channels, context, interfaces, etc.)
3. Asks the question "What is a goroutine?"
4. Outputs the answer with sources

## Requirements

- Go 1.21+
- For `LLM_PROVIDER=mock` — nothing else needed
- For `LLM_PROVIDER=ollama` — running [Ollama](https://ollama.ai) with models
- For `LLM_PROVIDER=openai` — `OPENAI_API_KEY`
- For `LLM_PROVIDER=anthropic` — `ANTHROPIC_API_KEY`
