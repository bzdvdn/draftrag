# Compatibility & Support Policy

This document defines the public "support contract" of draftRAG: which Go versions are supported, how stable the public API is, and the status of each backend.

Important:
- This is a **best-effort** policy (no SLA/SLO guarantees).
- The contract applies to the public API of `pkg/draftrag` and public docs. Everything in `internal/` may change without notice.

## Go support

- Minimum Go version: **1.23**.
- Support window: we support the **N latest minor Go releases**, where **N = 2**, plus the minimum (as long as it stays within this window).
  - Example: when a new Go minor version is released, we update the support window in the next library releases.
- Raising the minimum Go version is a **breaking change**:
  - announced in advance in release notes (and/or CHANGELOG, if maintained);
  - takes effect in a **major** release.

## SemVer & Deprecation (public API)

- Semantic versioning (SemVer) applies to the public API of `pkg/draftrag` and documented behavior.
- Breaking changes:
  - allowed only in **major** releases;
  - must come with migration notes in the release notes.
- Deprecation:
  - mark in godoc with a `Deprecated:` prefix and specify a replacement;
  - **keep deprecated API for at least 2 minor releases or 6 months (whichever is longer)**;
  - remove only in the next **major** release.

## Backend statuses

Status definitions:
- **stable** — supported and considered production-ready with proper configuration (timeouts, retries, observability).
- **experimental** — functional, but the contract/behavior may change faster; use with extra attention to release notes.

### Vector stores

| Backend | Status | Notes |
|---|---|---|
| In-memory | stable | For prototypes/tests; **no** persistent storage |
| PostgreSQL + pgvector | stable | Production-ready; hybrid search (BM25+semantic), SQL filters, migrations |
| Qdrant | stable | Production-ready; payload filters, collection management |
| ChromaDB | stable | Requires a pre-created collection; filters available via API |
| Weaviate | stable | Production-ready; basic retrieval, filters, collection management; **hybrid search not supported** |

### Embedders

| Backend | Status | Notes |
|---|---|---|
| OpenAI-compatible embeddings | stable | Any compatible `POST /v1/embeddings` |
| Ollama embeddings | stable | Local models via Ollama |
| CachedEmbedder (LRU + opt. Redis L2) | stable | Cache on top of any embedder |

### LLM providers

| Backend | Status | Notes |
|---|---|---|
| OpenAI-compatible (Responses API) | stable | Supports streaming via `StreamingLLMProvider` |
| Anthropic Claude | stable | Supports streaming via `StreamingLLMProvider` |
| Ollama LLM | stable | **Streaming not supported** via draftRAG |

## Capability matrix (best-effort via docs/README)

Legend: `✓` — supported, `—` — not supported, `n/a` — not applicable.

### Vector stores

| Feature | In-memory | pgvector | Qdrant | ChromaDB | Weaviate |
|---|---|---:|---:|---:|---:|---:|
| Persistent storage | — | ✓ | ✓ | ✓ | ✓ |
| Metadata filters | ✓ | ✓ | ✓ | ✓ | ✓ |
| Hybrid search (BM25) | ✓ | ✓ | — | — | — |
| SQL migrations | n/a | ✓ | n/a | n/a | n/a |
| Collection management | n/a | n/a | ✓ | ✓ | ✓ |

### LLM providers

| Feature | OpenAI-compatible | Anthropic | Ollama |
|---|---|---:|---:|---:|
| Generate (non-stream) | ✓ | ✓ | ✓ |
| Streaming (`AnswerStream`, `.Stream*`) | ✓ | ✓ | — |

### Cross-cutting

| Feature | Support |
|---|---|
| Timeouts/cancellation | `context.Context` everywhere; some backends have optional timeouts in options |
| Retry + Circuit Breaker | wrappers `RetryEmbedder` and `RetryLLMProvider` |
| Embedding caching | `CachedEmbedder` (L1 LRU) + optional Redis L2 |
| Observability hooks | hooks per pipeline stage (chunking/embed/search/generate) |
| OpenTelemetry | public hooks in `pkg/draftrag/otel` |

## Update policy

- This document is updated alongside draftRAG releases.
- Any change to backend status, Go support window, or deprecation rules is reflected in the release notes.
