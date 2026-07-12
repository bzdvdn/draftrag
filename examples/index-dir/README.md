# index-dir — Directory indexing with files

Recursively walks a directory, reads `.txt` files, chunks them using `BasicChunker`, and indexes them into an in-memory store. After indexing, answers a given question and outputs sources.

## Quick start

```bash
EMBEDDER_API_KEY=sk-... \
LLM_API_KEY=sk-... \
go run ./examples/index-dir/ -dir ./docs -query "How to configure authorization?"
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `-dir` | `.` | Directory with `.txt` files (recursive) |
| `-query` | — | **Required.** Question for RAG |
| `-topk` | `5` | Number of chunks for context |
| `-chunk` | `500` | Chunk size in runes |
| `-overlap` | `60` | Overlap between chunks in runes |

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `EMBEDDER_API_KEY` | — | **Required.** API key for the embedder |
| `EMBEDDER_BASE_URL` | `https://api.openai.com` | Embedder API base URL |
| `EMBEDDER_MODEL` | `text-embedding-ada-002` | Embedding model |
| `LLM_API_KEY` | — | **Required.** API key for the LLM |
| `LLM_BASE_URL` | `https://api.openai.com` | LLM API base URL |
| `LLM_MODEL` | `gpt-4o-mini` | Language model |

## Example: indexing project documentation

```bash
# Save some .txt files to the docs/ folder
mkdir -p docs
echo "Authorization is done via a Bearer token in the Authorization header." > docs/auth.txt
echo "Configuration is stored in config.yaml in the project root." > docs/config.txt

EMBEDDER_API_KEY=sk-... LLM_API_KEY=sk-... \
go run ./examples/index-dir/ \
  -dir ./docs \
  -query "Where is the configuration stored?" \
  -topk 3 \
  -chunk 300
```

## Example output

```
Found 2 files in "docs"
Indexing 2 documents...
Indexing complete.

Question: Where is the configuration stored?
────────────────────────────────────────────────────────────

Configuration is stored in config.yaml in the project root.

Sources:
  [1] docs/config.txt (score=0.944)
      Configuration is stored in config.yaml in the project root.
```

## File format

Files with the `.txt` extension are read (case-insensitive). The file name becomes the document `ID`, the file path becomes the `path` metadata field. Empty files are skipped.
