# Production checklist + runbook

This document is a **set of starting practices**, not an SLO/latency guarantee. Goal: provide short, verifiable steps for "what to do before a release" and "what to do during an incident".

## Checklist (before release)

1. **Use `context.Context` and timeouts everywhere.** Every path (`Index`, `Retrieve/Answer/Cite`, migrations, collection operations) must have `context.WithTimeout(...); defer cancel()`. See `pipeline.md`.
2. **Retry + Circuit Breaker enabled on external APIs.** For LLM/Embedder use `NewRetryLLMProvider` / `NewRetryEmbedder` and set reasonable limits (max retries, backoff, CB threshold/timeout). See `README.md` → "Retry + Circuit Breaker".
3. **Embedding cache enabled.** `NewCachedEmbedder` (L1 LRU) and (optionally) Redis L2 for repeated queries. See `README.md` → "Embedding caching" and "Redis L2".
4. **Limits set explicitly.** `DefaultTopK`, context limits (`MaxContextChars/Chunks`), `IndexBatch` concurrency/rate limit — fixed and tested for your traffic.
5. **pgvector: migrations run as a separate deploy step.** Do not run DDL "at service startup"; use `MigratePGVector/SetupPGVector` in a deploy job/init container. See `pkg/draftrag/pgvector.go` and `pkg/draftrag/pgvector_migrations.md`.
6. **pgvector: runtime timeouts/limits configured.** `PGVectorRuntimeOptions` (Search/Upsert/Delete timeouts, MaxTopK, MaxParentIDs) correspond to your latency budget.
7. **Qdrant/Weaviate: collections prepared.** Either auto-create in a deploy job, or an explicit check with `CollectionExists/CreateCollection` (Qdrant) / `WeaviateCollectionExists/CreateWeaviateCollection` (Weaviate). Vector dimensions match the embedder.
8. **Observability enabled.** Minimum: hooks per stage (chunking/embed/search/generate). Better: OTel hooks (`pkg/draftrag/otel`) and metrics/traces in your system. See `README.md` → "Observability hooks".
9. **Logs are safe.** Do not log raw documents/queries without your own policy. Ensure secrets (APIKey/bearer token) do not appear in errors/logs (best-effort redaction from the library). See `README.md` → "Redaction and log security".
10. **Retrieval quality eval run.** Check Hit@K/MRR on your use cases, establish a baseline before release. See `README.md` → "Eval harness".
11. **Regression check.** `go test ./...` green; a minimal smoke test for indexing/retrieval run on staging.

## Backend notes (what matters in operations)

| Backend | What matters | Typical mistakes |
|---|---|---|
| PostgreSQL + pgvector | DDL migrations separate from the service; runtime timeouts; `CREATE EXTENSION` permissions | permission denied, long-running DDL, dimension mismatch |
| Qdrant | collection exists; dimension matches; HTTP timeout | 404/collection missing, dimension mismatch, timeouts |
| Weaviate | class/schema exists; APIKey in headers; HTTP timeout | 401/403, schema errors, timeouts |

## Runbook (incidents)

Below "quickly" = **short and step-by-step**.

### 1) Empty results / low recall

**Symptoms**
- `Retrieve/Answer` returns 0 sources or irrelevant context.

**Checks**
- Indexing actually ran (document/chunk count).
- Embedder dimension matches storage (pgvector dimension, Qdrant/Weaviate dimension).
- `TopK` is not too small; no `Filter/ParentIDs` that "cut off everything".
- Hooks/OTel: `search` stage does not return errors and is not too fast/empty.

**Actions**
- Increase `TopK`, temporarily remove filters, test the query.
- Rebuild the index (if embedder model/dimension changed).
- For pgvector: ensure migrations are applied, indexes are created.

### 2) Latency increase (p95/p99) or timeouts

**Symptoms**
- `context deadline exceeded`, response degradation, p95 increase.

**Checks**
- Hooks/OTel: which stage grew (`embed`, `search`, `generate`).
- Is the circuit breaker opening (`CB open` / `retry attempt failed` increase).
- pgvector/Qdrant/Weaviate: network timeouts, connection pool, database/cluster overload.

**Actions**
- Increase timeouts only where needed (e.g., `generate`), but keep the overall budget.
- Reduce concurrency (`IndexBatch`, query parallelism), enable/increase embedding cache.
- For pgvector: check query plan/indexes, `MaxTopK`, content/chunk size.

### 3) Circuit breaker "open" / retry spike

**Symptoms**
- Errors like "circuit breaker: open", retries increasing, responses unstable.

**Checks**
- Error type: rate limit/5xx/timeout vs non-retryable.
- Load: concurrent requests, burst, is the cache warm.
- Logs should not contain secrets in retry/cache messages (best-effort redaction).

**Actions**
- Reduce throughput (limit concurrency), increase backoff/jitter, raise `CBTimeout` for "cooling down".
- If rate limit: add rate limiting on the service side, calibrate `MaxRetries`.
- If persistent 4xx: make the error non-retryable (on the provider/config side), fix the key/model/endpoint.

### 4) pgvector migrations not applied / permission errors

**Symptoms**
- DDL errors, missing tables/indexes, permission denied.

**Checks**
- Migrations run as a separate step (deploy job/init container), not at service startup.
- DB role has DDL permissions and (if needed) `CREATE EXTENSION vector`.
- Migration timeout is sufficient (DDL can be slow).

**Actions**
- Move migrations to a separate deploy step; separate runtime vs migrate permissions.
- Enable `CreateExtension` only if you are certain about permissions/policy.

### 5) Qdrant/Weaviate: "collection/class missing" or dimension mismatch

**Symptoms**
- 404/collection missing, schema errors, dimension mismatch.

**Checks**
- Collection/class is created before service startup (or the first request triggers create).
- `Dimension` matches the current embedding model.

**Actions**
- Create the collection/class via a deploy job, pin dimension as an environment constant.
- When changing embedder model: recreate the collection/reindex data.

## Security/Redaction

- draftRAG best-effort redacts **secrets known to the library** (e.g., `APIKey`/bearer token from options) from error messages it generates.
- draftRAG **does not** auto-detect PII in arbitrary text.
- User responsibility: do not log raw documents/queries without your own policy (redaction/masking/retention).

## Index rate limiting

`PipelineOptions.IndexBatchRateLimit` controls the indexing throughput
(for `Index` and `IndexBatch`); `0` means unlimited. By default, a
**single shared ticker per pool** is used: `IndexBatchRateLimit=10` with
`IndexConcurrency=4` gives **~10 embed/sec total** per pool.

### Per-worker rate limiting (DEC-007, RQ-007)

If each worker has its own independent quota limit (e.g., multiple
API keys with per-key rate limits), enable `IndexBatchRateLimitPerWorker: true`.
In this mode each worker gets its **own ticker** with an interval of
`time.Second / IndexBatchRateLimit`, and the total throughput
scales with the number of workers:

| `IndexBatchRateLimit` | `IndexConcurrency` | `PerWorker` | Approx total rate |
|---|---|---|---|
| 10 | 4 | `false` (default) | 10 embed/sec |
| 10 | 4 | `true`             | 40 embed/sec |
| 50 | 8 | `false`            | 50 embed/sec |
| 50 | 8 | `true`             | 400 embed/sec |

Choose the mode for your use case:

- **shared (default, `false`)** — global indexing limit, independent
  of concurrency. Safe for downstream services with a single shared rate limit.
- **per-worker (`true`)** — each worker independently stays within its
  limit. Suitable for fan-out across different API keys/pods, where each
  has its own quota.

See `IndexBatch` godoc and `PipelineOptions.IndexBatchRateLimitPerWorker` for
implementation details.
