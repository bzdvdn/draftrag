# Weaviate — RAG with Weaviate

Interactive RAG chat with Weaviate as a vector store. The collection is created automatically on first run.

## Quick start

**1. Start Weaviate:**

```bash
docker compose up -d
```

**2. Run the example:**

```bash
cd examples/weaviate && cp .env.example .env && go run .
```

For mock mode this is sufficient. For a real LLM, set `LLM_PROVIDER=ollama|openai|anthropic` and the corresponding keys.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `mock` | LLM provider (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Vector dimension |
| `WEAVIATE_URL` | `http://localhost:8080` | Weaviate server URL |
| `COLLECTION_NAME` | `DraftragChunk` | Weaviate class name |

## Tutorial

Detailed guide on hybrid search — [tutorial 03: Hybrid Search](../docs/tutorials/en/03-hybrid-search.md).
