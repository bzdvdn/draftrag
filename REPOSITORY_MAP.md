# Repository Map

Compact code-only navigation index for the draftRAG Go library.

## Entry Points

- `pkg/draftrag/draftrag.go` — `NewPipeline` / `NewPipelineWithChunker` / `NewPipelineWithOptions` (public Pipeline constructors)
- `pkg/draftrag/pgvector.go` — `NewPGVectorStore` / `NewPGVectorStoreWithOptions` / `NewPGVectorStoreWithRuntimeOptions`
- `pkg/draftrag/memory.go` — `NewInMemoryStore` (in-memory VectorStore)
- `pkg/draftrag/qdrant.go` — `NewQdrantStore` (Qdrant)
- `pkg/draftrag/chromadb.go` — `NewChromaDBStore` (ChromaDB)
- `pkg/draftrag/weaviate.go` — `NewWeaviateStore` (Weaviate)
- `pkg/draftrag/ollama_embedder.go` — `NewOllamaEmbedder` (Ollama embedder)
- `pkg/draftrag/openai_compatible_embedder.go` — `NewOpenAICompatibleEmbedder` (OpenAI-compatible embedder)
- `pkg/draftrag/ollama_llm.go` — `NewOllamaLLM` (Ollama LLM)
- `pkg/draftrag/openai_compatible_llm.go` — `NewOpenAICompatibleLLM` (OpenAI-compatible LLM)
- `pkg/draftrag/anthropic_llm.go` — `NewAnthropicLLM` (Anthropic Claude LLM)
- `pkg/draftrag/mistral_llm.go` — `NewMistralLLM` (Mistral Chat Completions API)
- `pkg/draftrag/mistral_embedder.go` — `NewMistralEmbedder` (Mistral embeddings)
- `pkg/draftrag/deepseek_llm.go` — `NewDeepSeekLLM` (DeepSeek Chat Completions API)
- `pkg/draftrag/basic_chunker.go` — `NewBasicChunker` (default chunker)
- `pkg/draftrag/cached_embedder.go` — `NewCachedEmbedder` + `cached_embedder_redis.go::NewRedisCache`
- `pkg/draftrag/resilience.go` — `NewRetryEmbedder` / `NewRetryLLMProvider`
- `pkg/draftrag/pgvector_migrate.go` — SQL migration runner (uses `pgvector_migrations_assets.go` for embedded SQL)
- `pkg/draftrag/otel/hooks.go` — `NewHooks` (OpenTelemetry hooks/tracing)
- `pkg/draftrag/rewriter.go` — `NewLLMRewriter` (QueryRewriter constructor)

## Top-Level Code

- `internal/domain/` — domain layer: interfaces (`VectorStore`, `TransactionalDocumentStore`, `DocumentStore`, `Embedder`, `LLMProvider`, `Chunker`, `Hooks`, `Logger`), models (`Document`, `RetrievalResult`, sentinels, `HybridConfig`), redaction helpers (`RedactSecret`/`RedactSecrets`), `models_test.go`
- `internal/application/` — application/orchestration layer: Pipeline implementation (index/query/retrieve/answer/stream), worker pool, atomic update, batch, MMR/rrf helpers, error sentinels
- `internal/infrastructure/chunker/` — chunker implementation (`BasicChunker`)
- `internal/infrastructure/rewriter/` — LLMRewriter implementation (LLM-based query rewriting strategy)
- `internal/infrastructure/embedder/` — concrete embedder HTTP clients (Ollama, OpenAI-compatible) + `cache/` subpackage (LRU + Redis + stats)
- `internal/infrastructure/llm/` — concrete LLM HTTP clients (Anthropic, Ollama, OpenAI-compatible, OpenAI Chat Completions, mock streaming)
- `internal/infrastructure/costtracker/` — `CostTracker` wrapper: LLMProvider-обёртка с подсчётом токенов и стоимости
- `internal/infrastructure/resilience/` — `circuitbreaker`, `retry`, `embedder`/`llm` wrappers, `hooks`, `errors`
- `internal/infrastructure/vectorstore/` — concrete VectorStore implementations (pgvector with transactions, memory, qdrant, chromadb, weaviate, milvus, hybrid search) + extensive `*_test.go` per backend
- `pkg/draftrag/` — public Go API surface: re-exports + facade + embedders/vectorstores/chunker/resilience/otel/eval/migrations
- `examples/` — 9 runnable per-backend examples (memory, pgvector, qdrant, chromadb, weaviate, milvus, mistral, deepseek, cost-tracking) + shared mock/print helpers + legacy (chat, index-dir) — NOT part of library API, demo only
- `pkg/draftrag/mistral_embedder.go` — `NewMistralEmbedder` factory wrapping `OpenAICompatibleEmbedder` with Mistral defaults

## Key Paths

- `internal/domain/interfaces.go` — `VectorStore`, `TransactionalDocumentStore` (BeginTx/DeleteByParentIDTx/UpsertTx/Commit/Rollback), `DocumentStore`, `Embedder`, `LLMProvider`, `Chunker`, `Hooks`, `Logger`, `UsageAwareLLMProvider`, `UsageAwareStreamingLLMProvider`, `QueryRewriter`
- `internal/domain/models.go` — `Document`, `RetrievalResult`, `HybridConfig`, sentinels (`ErrEmptyQuery`, `ErrInvalidQueryTopK`, `ErrUpdateNotAtomic`, `ErrEmbeddingDimensionMismatch`, etc.); cost-tracking: `TokenUsage`, `ModelPricing`, `CostSnapshot`, `Diff`; query-rewriting: `RewrittenQuery`, `QueryHistory`
- `internal/domain/hooks.go` — `Hooks` callback contract (OnStart/OnEnd/OnError) for instrumentation
- `internal/domain/redaction.go` — `RedactSecret` / `RedactSecrets` helpers for PII/token redaction
- `internal/application/pipeline.go` — `(*Pipeline).Index`, `(*Pipeline).Query`, `(*Pipeline).Retrieve`, `(*Pipeline).Answer`, `(*Pipeline).UpdateDocument`; `PipelineConfig` + `Pipeline` field plumbing
- `internal/application/worker_pool.go` — `processDocsConcurrently` (semaphore + ticker + per-worker or shared rate-limit)
- `internal/application/atomic_update.go` — `updateDocumentAtomic` (transactional vs best-effort with `ErrUpdateNotAtomic`)
- `internal/application/batch.go` — `IndexBatch` (thin wrapper over worker pool)
- `internal/application/stream.go` — `wrapStreamWithHook` (bounded backpressure via `streamBufferSize`)
- `internal/application/{query,answer,retrieval,mmr,rrf}.go` — retrieval/answer/rerank logic; `QueryWithQueries`, `AnswerWithQueries*` (multi-query retrieval)
- `internal/infrastructure/vectorstore/pgvector.go` — `pgVectorTx` transactional path; `BeginTx`; SQL operations
- `internal/infrastructure/vectorstore/hybrid.go` — `HybridConfig` + `HybridSearch` plumbing
- `pkg/draftrag/draftrag.go` — `Pipeline`, `PipelineOptions` (IndexConcurrency, StreamBufferSize, IndexBatchRateLimitPerWorker, HybridConfig, QueryRewriter, etc.), `NewPipeline*` constructors, `mapAppError`; re-export `TokenUsage`, `ModelPricing`, `CostSnapshot`, `UsageAwareLLMProvider`, `UsageAwareStreamingLLMProvider`, `Diff`, `QueryRewriter`, `RewrittenQuery`, `QueryHistory`
- `pkg/draftrag/costtracker.go` — `CostTracker`, `NewCostTracker` (публичная обёртка LLMProvider с подсчётом токенов/стоимости)
- `pkg/draftrag/search.go` + `search_routing.go` — public `SearchBuilder` (Retrieve/Answer/Cite/InlineCite/Stream/StreamSources/StreamCite) with `selectRetrieval`/`selectGeneration`; `Rewriter`/`History` methods + `routeRewriter` handlers
- `pkg/draftrag/errors.go` — re-exported public sentinels
- `pkg/draftrag/migrations/pgvector/` — embedded SQL migrations (`0000_…` / `0001_…` / `0002_…`)
- `pkg/draftrag/otel/` — OTel hooks (tracing + metrics)
- `pkg/draftrag/eval/` — evaluation harness (`harness.go`, `metrics.go`, `models.go`)
- `.github/workflows/ci.yml` — CI pipeline (test, lint, vet)
- `.github/workflows/examples-smoke.yml` — per-backend smoke CI (compose-validate + build + 6× mock-run matrix)
- `Makefile` — `test`, `test-cover`, `lint`, `lint-fix`, `fmt`, `fmt-check`, `vet`, `build`, `tidy`

## Where To Edit

- New VectorStore backend → implement `domain.VectorStore` in `internal/infrastructure/vectorstore/<name>.go`; add `<name>_test.go`; add `NewXxxStore` constructor in `pkg/draftrag/<name>.go`; add row to capability table in `docs/vector-stores.md`
- New embedder/LLM provider → `internal/infrastructure/{embedder,llm}/` + `pkg/draftrag/<provider>.go`
- New query rewriter → `internal/infrastructure/rewriter/` + `pkg/draftrag/rewriter.go`
- Pipeline public surface (constructor, options) → `pkg/draftrag/draftrag.go` (PipelineOptions struct)
- Pipeline orchestration (Index/Query/Answer/Stream/UpdateDocument) → `internal/application/pipeline.go`; per-method helpers in `{query,answer,stream,retrieval,batch,mmr,rrf}.go`
- Concurrency / worker pool / rate limit → `internal/application/worker_pool.go`; options in `pkg/draftrag/draftrag.go::PipelineOptions`
- Atomic update semantics → `internal/application/atomic_update.go`; `TransactionalDocumentStore` impl in `internal/infrastructure/vectorstore/pgvector.go`
- Public sentinels (re-exports + mapping) → `pkg/draftrag/errors.go` (re-exports) + `pkg/draftrag/draftrag.go::mapAppError` (mapping)
- Domain sentinels/interfaces → `internal/domain/models.go` + `interfaces.go`
- OpenTelemetry instrumentation → `pkg/draftrag/otel/hooks.go`; pipeline hooks wiring in `internal/application/pipeline.go`
- Resilience (retry, circuit breaker) → `internal/infrastructure/resilience/` + facade in `pkg/draftrag/resilience.go`
- Migrations / SQL schema → `pkg/draftrag/migrations/pgvector/*.sql` (regenerate `pgvector_migrations_assets.go` via `go generate ./...`)
- Eval harness (NDCG, Precision@K, Recall@K) → `pkg/draftrag/eval/`
- Specs/plans/tasks → `docs/specs/<slug>/{spec,plan,data-model,tasks}.md`

## Excluded

- `.speckeep/**` — workflow state, scripts, templates
- `.git/**` — version control
- `docs/**` — documentation
- `docs/specs-archive/**` — completed/archived specs
- `examples/**` — demo code, not part of library API
- `*.md` at root — README, ROADMAP, CHANGELOG, CONTRIBUTING, etc.
- `vendor/**`, `node_modules/**` — dependency sources
- `coverage/**` — generated coverage reports
- `**/*_test.go` — test files (referenced by the corresponding source file)
- `**/doc.go` — package-level documentation comments
