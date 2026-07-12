# ChromaDB — RAG with ChromaDB

Interactive RAG chat with ChromaDB as a vector store. The collection is created automatically on first run.

## Quick start

**1. Start ChromaDB:**

```bash
docker compose up -d
```

**2. Run the example:**

```bash
cd examples/chromadb && cp .env.example .env && go run .
```

For mock mode this is sufficient. For a real LLM, set `LLM_PROVIDER=ollama|openai|anthropic` and the corresponding keys.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `mock` | LLM provider (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Vector dimension |
| `CHROMADB_URL` | `http://localhost:8000` | ChromaDB server URL |
| `COLLECTION_NAME` | `draftrag_chunks` | Collection name |

## Tutorial

Detailed guide on working with ChromaDB and metadata — [tutorial 04: Metadata Filter](../docs/tutorials/en/04-metadata-filter.md).
