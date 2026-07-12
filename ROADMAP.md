# ROADMAP — draftRAG

Spec-driven roadmap. Каждая фича = spec slug в `docs/specs-archive/<slug>/` (✅) или будущий spec (📋).

## Workflow

```
/spk.spec <slug>     → spec.md
/spk.plan            → plan.md + tasks.md
/spk.implement       → implementation
/spk.verify          → verify report
speckeep archive     → docs/specs-archive/<slug>/<date>/
```

---

## Core Pipeline

| Фича | Spec-ы | Статус |
|---|---|---|
| Pipeline config & constructors | `pipeline-config`, `public-api-options-unification` | ✅ |
| Index (single + batch + chunker) | `pipeline-index-with-chunker`, `batch-indexing` | ✅ |
| Query, Answer, Citations, Streaming | `pipeline-answer`, `answer-with-citations`, `answer-inline-citations`, `streaming-responses`, `fix-inline` | ✅ |
| Fluent SearchBuilder API | `fluent-search-api`, `search-builder-routing-fix`, `search-builder-stream-sources` | ✅ |
| Document lifecycle (update, delete, atomic) | `document-lifecycle` | ✅ |

## Vector Stores

| Фича | Spec-ы | Статус |
|---|---|---|
| In-memory | `core-components` | ✅ |
| pgvector (full: filters, migrations, hybrid, production) | `vectorstore-pgvector`, `vectorstore-pgvector-migrations`, `vectorstore-pgvector-dimension-guard`, `vectorstore-pgvector-production` | ✅ |
| Qdrant (+ hybrid search) | `qdrant-vector-store`, `qdrant-hybrid-search` | ✅ |
| ChromaDB (collections, миграции, parity) | `chromadb-vector-store`, `chromadb-collection-management`, `chromadb-migrations`, `chromadb-parity` | ✅ |
| Weaviate (basic + docs) | `vectorstore-weaviate`, `weaviate-docs`, `weaviate-full-support`, `weaviate-hybrid-search` | ✅ experimental |
| Milvus (basic + hybrid research) | `milvus-vectorstore`, `milvus-hybrid-search` | ✅ experimental |

## LLM Providers

| Фича | Spec-ы | Статус |
|---|---|---|
| OpenAI-compatible (Responses API) | `llm-openai-compatible` | ✅ |
| Anthropic Claude (native Messages API) | `anthropic-claude-llm` | ✅ |
| Ollama (streaming + non-streaming) | `ollama-llm-embedder`, `ollama-llm-no-streaming` | ✅ |
| Mistral, DeepSeek | `llm-providers-mistral-deepseek` | ✅ |

## Embedders

| Фича | Spec-ы | Статус |
|---|---|---|
| OpenAI-compatible | `embedder-openai-compatible` | ✅ |
| Ollama | `ollama-llm-embedder` | ✅ |
| Cached embedder (LRU + Redis L2) | `embedding-cache`, `cached-embedder-public`, `redis-cache-backend` | ✅ |

## Retrieval Strategies

| Фича | Spec-ы | Статус |
|---|---|---|
| Hybrid search (BM25 + semantic) | `hybrid-search-bm25-semantic` | ✅ pgvector only |
| Metadata filtering | `metadata-filtering` | ✅ all stores |
| HyDE, Multi-query | `retrieval-strategies` | ✅ |
| MMR reranking | `retrieval-reranker-mmr` | ✅ |
| Deduplication by ParentID | `retrieval-deduplication` | ✅ |
| Reranker (Cohere Rerank API + batch) | `reranker-cross-encoder` | ✅ |

## Resilience

| Фича | Spec-ы | Статус |
|---|---|---|
| Retry + Circuit Breaker | `retry-circuit-breaker`, `resilience-public-api` | ✅ |

## Observability

| Фича | Spec-ы | Статус |
|---|---|---|
| OTel tracing + metrics | `otel-observability`, `observability-hooks` | ✅ |
| Structured logger + slog adapter | `structured-logger-hooks`, `slog-otel-adapters` | ✅ |
| **Health check интерфейс** | — | 📋 |

## Eval / Testing

| Фича | Spec-ы | Статус |
|---|---|---|
| Retrieval metrics (Hit@K, MRR, NDCG) | `eval-harness-basic`, `eval-harness-retrieval-only` | ✅ |
| Contract tests for stores | `contract-tests-stores` | ✅ |
| Fuzz + property tests | `fuzz-property-tests` | ✅ |
| E2E benchmarks | `pipeline-e2e-benchmarks` | ✅ |

## Cross-cutting / Quality

| Фича | Spec-ы | Статус |
|---|---|---|
| Arch quality pass + generics refactor | `arch-quality-pass`, `arch-generics`, `searchbuilder-generics` | ✅ |
| API consistency pass | `api-consistency-pass` | ✅ |
| Security (redaction) | `security-redaction` | ✅ |
| Hardening 2026Q2 | `hardening-2026q2`, `go-version-target`, `production-checklist-runbook`, `compatibility-support-policy`, `api-resilience-fixes` | ✅ |
| Docs (bilingual: en + ru) | `docs-and-examples` | ✅ |
| Public examples (6 backends) | `public-examples` | ✅ |

## Cross-cutting / Advanced RAG

| Фича | Spec-ы | Статус |
|---|---|---|
| Tool calling (agentic RAG) | `arch-issues` | ✅ |
| CJK tokenization | `cjk-tokenization` | ✅ |

---

## Backlog — Specs к созданию

Приоритет: P0 → P1 → P2. Каждый item — будущий `/spk.spec <slug>`.

### P1 — Quality & maturity

_(пусто — все P1 фичи реализованы)_

### P2 — Ecosystem & advanced RAG

_(пусто — все P2 фичи реализованы)_

---

**Legend**:
- ✅ — spec archived, feature complete
- 🚧 — spec created, in progress
- 📋 — backlog, `/spk.spec <slug>` to start
