# chat — Basic RAG chat (CLI)

Interactive RAG chat with in-memory storage. Loads a set of documents, then accepts questions via stdin and answers with inline citations `[1]`, `[2]`.

The store lives in memory — data is not persisted between runs.

## Quick start

```bash
EMBEDDER_API_KEY=sk-... \
LLM_API_KEY=sk-... \
go run ./examples/chat/
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `EMBEDDER_API_KEY` | — | **Required.** API key for the embedder |
| `EMBEDDER_BASE_URL` | `https://api.openai.com` | Embedder API base URL |
| `EMBEDDER_MODEL` | `text-embedding-ada-002` | Embedding model |
| `LLM_API_KEY` | — | **Required.** API key for the LLM |
| `LLM_BASE_URL` | `https://api.openai.com` | LLM API base URL |
| `LLM_MODEL` | `gpt-4o-mini` | Language model |

## Local mode (Ollama)

```bash
# Start Ollama and pull the required models:
ollama pull nomic-embed-text
ollama pull llama3.2

EMBEDDER_BASE_URL=http://localhost:11434 \
EMBEDDER_API_KEY=ollama \
EMBEDDER_MODEL=nomic-embed-text \
LLM_BASE_URL=http://localhost:11434 \
LLM_API_KEY=ollama \
LLM_MODEL=llama3.2 \
go run ./examples/chat/
```

## Example session

```
Indexing knowledge base...
Indexed 8 documents.

RAG chat ready. Enter your question (Ctrl+C to exit):
────────────────────────────────────────────────────────────

> How do I add a Zigbee device?

To add a Zigbee device, open the SmartHome application,
select "Add Device" -> "Zigbee" and put the device
into pairing mode [1]. The hub will detect it within 30 seconds [1].

Sources:
  [1] smarthome-zigbee (score=0.921)
────────────────────────────────────────────────────────────
```

## Knowledge base

The example uses a built-in knowledge base about the SmartHome Hub product (8 documents). To use your own documents, replace the `knowledgeBase` slice in `main.go`.
