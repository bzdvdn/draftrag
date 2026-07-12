# pgvector — RAG with PostgreSQL + pgvector

Interactive RAG chat with pgvector as a persistent vector store. The database schema is created automatically on first run via `MigratePGVector` (idempotent).

## Quick start

**1. Start PostgreSQL with pgvector:**

```bash
docker compose up -d
```

**2. Run the example:**

```bash
cd examples/pgvector && cp .env.example .env && go run .
```

For mock mode this is sufficient. For a real LLM, set `LLM_PROVIDER=ollama|openai|anthropic` and the corresponding keys.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `mock` | LLM provider (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Vector dimension (must match the model) |
| `PGVECTOR_DSN` | — | **Required.** DSN for connecting to PostgreSQL |
| `TABLE_NAME` | `draftrag_chunks` | Table name for storing chunks |

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

## Vector dimension

The dimension must match the embedding model used:

| Model | `EMBEDDING_DIM` |
|---|---|
| `text-embedding-ada-002` | `1536` |
| `text-embedding-3-small` | `1536` |
| `text-embedding-3-large` | `3072` |
| `nomic-embed-text` (Ollama) | `768` |

If the dimension changes after the first run, you need to recreate the table or use a different `TABLE_NAME`.

## Migrations

`MigratePGVector` creates the table and index on first run. Re-running is safe — migrations are idempotent.

For production, it is recommended to apply SQL migrations as a separate deployment step:

```bash
psql $PGVECTOR_DSN -f pkg/draftrag/migrations/pgvector/0000_pgvector_extension.sql
psql $PGVECTOR_DSN -f pkg/draftrag/migrations/pgvector/0001_chunks_table.sql
psql $PGVECTOR_DSN -f pkg/draftrag/migrations/pgvector/0002_metadata_and_indexes.sql
```

## Local mode (Ollama)

```bash
ollama pull nomic-embed-text
ollama pull llama3.2

PGVECTOR_DSN="postgres://draftrag:draftrag@localhost:5432/draftrag?sslmode=disable" \
EMBEDDING_DIM=768 \
LLM_PROVIDER=ollama \
OLLAMA_HOST=http://localhost:11434 \
go run ./examples/pgvector/
```
