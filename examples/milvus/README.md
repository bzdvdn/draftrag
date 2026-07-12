# Milvus — RAG with Milvus

Interactive RAG chat with Milvus as a vector store.

**Note:** Milvus is the most resource-intensive backend. Requires ~2 GB RAM to run. First startup may take time to initialize (start_period: 30s).

## Quick start

**1. Start Milvus (etcd + minio + milvus standalone):**

```bash
docker compose up -d
```

**2. Run the example:**

```bash
cd examples/milvus && cp .env.example .env && go run .
```

For mock mode this is sufficient. For a real LLM, set `LLM_PROVIDER=ollama|openai|anthropic` and the corresponding keys.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `mock` | LLM provider (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Vector dimension |
| `MILVUS_ADDR` | `localhost:19121` | Milvus REST API address |
| `COLLECTION_NAME` | `draftrag_chunks` | Collection name |

## Note

MilvusStore is an internal API (`internal/infrastructure/vectorstore`), status: "API in development".
The public API will be added in one of the upcoming releases.
