# Sub-Query Decomposition Example

Demonstrates splitting a complex query into sub-questions
using `SearchBuilder.SubDecompose()` to improve recall.

## Run

```bash
# With mock provider (no external dependencies)
LLM_PROVIDER=mock go run .
```

To use real LLM/embedder providers, set
`LLM_PROVIDER=ollama` or `LLM_PROVIDER=openai` with the appropriate
environment variables (see examples in `examples/memory/`).
