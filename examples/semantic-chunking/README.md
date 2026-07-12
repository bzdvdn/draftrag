# Semantic Chunking Example

Demonstrates using `SemanticChunker` for intelligent document
chunking based on sentence semantic similarity.

## Run

```bash
# With mock provider (no external dependencies)
LLM_PROVIDER=mock go run .
```

To use real LLM/embedder providers, set
`LLM_PROVIDER=ollama` or `LLM_PROVIDER=openai` with the appropriate
environment variables (see examples in `examples/memory/`).
