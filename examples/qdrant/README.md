# Qdrant — RAG with Qdrant

Interactive RAG chat with Qdrant as a vector store. Creates the collection automatically on first run.

## Quick start

**1. Start Qdrant:**

```bash
docker compose up -d
```

**2. Run the example:**

```bash
cd examples/qdrant && cp .env.example .env && go run .
```

For mock mode this is sufficient. For a real LLM, set `LLM_PROVIDER=ollama|openai|anthropic` and the corresponding keys.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `mock` | LLM provider (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Vector dimension |
| `QDRANT_URL` | `http://localhost:6333` | Qdrant server URL |
| `COLLECTION_NAME` | `draftrag_chunks` | Collection name |

For `LLM_PROVIDER=ollama`:

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama URL |
| `OLLAMA_EMBED_MODEL` | `nomic-embed-text` | Embedding model |
| `OLLAMA_LLM_MODEL` | `llama3.2` | LLM model |

For `LLM_PROVIDER=openai`:

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_API_KEY` | — | **Required.** API key |
| `OPENAI_BASE_URL` | `https://api.openai.com` | Base URL |
| `OPENAI_EMBED_MODEL` | `text-embedding-3-small` | Embedding model |
| `OPENAI_LLM_MODEL` | `gpt-4o-mini` | LLM model |

For `LLM_PROVIDER=anthropic`:

| Variable | Default | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | — | **Required.** API key |
| `ANTHROPIC_LLM_MODEL` | `claude-3-5-sonnet-latest` | LLM model |
