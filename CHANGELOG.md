# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-07-13

### Added

**Reranker — Cohere Cross-Encoder**
- `pkg/draftrag/reranker/` — `NewCohereRerank(CohereRerankOptions)` для Cohere Rerank API v2 (одиночный + batch fan-out режим)
- Поддержка Model, BaseURL, Timeout, MaxRetries, MaxTokensPerDoc, кастомного HTTPClient

**LLM-as-judge Reranker**
- `NewLLMReranker(llm, opts...)` — ранжирование чанков через LLMProvider с батчингом
- Опции: `WithBatchSize`, `WithPromptTemplate`, `WithMaxRetries`

**Health Check Interface**
- `NewHealthChecker`, `Register`, `Check` — конкурентная проверка компонентов
- Готовые HTTP-обработчики: `LivenessHandler`, `ReadinessHandler`, `StartupHandler` (K8s probes)

**Cost Tracking**
- `NewCostTracker(llm, pricing)` — прозрачная обёртка LLMProvider с подсчётом токенов и стоимости
- `Snapshot`, `Checkpoint`, `Reset` — атомарные срезы статистики
- `GenerateStream` — поддержка streaming
- Re-export: `TokenUsage`, `ModelPricing`, `CostSnapshot`, `Diff`

**Graceful Degradation / Fallback**
- `NewFallbackLLMProvider`, `NewFallbackStreamingLLMProvider`, `NewFallbackUsageAwareLLMProvider` — fallback-цепочки провайдеров
- Автоматическое переключение при ошибке с логированием и статистикой
- Re-export: `FallbackStats`, `ErrAllProvidersFailed`

**Rate Limiting (Token Bucket)**
- `NewTokenBucketLLMProvider`, `NewTokenBucketEmbedder`, `NewTokenBucketStreamingLLMProvider` — token bucket rate limiter
- Настраиваемые `TokensPerSecond` и `BurstSize`

**Query Rewriting**
- `NewLLMRewriter(llm, promptTemplate)` — LLM-based переформулировка запроса перед retrieval
- `SearchBuilder.Rewriter(rw).History(history).Answer(ctx)` — fluent API для multi-turn RAG
- Re-export: `QueryRewriter`, `RewrittenQuery`, `QueryHistory`

**Eval Harness + RAGAS Metrics**
- `Run`, `RunWithAnswer` — прогон retrieval-датасета, опционально с ответами
- Retrieval-метрики: Hit@K, MRR, NDCG, Precision@K, Recall@K
- RAGAS-метрики: `ComputeFaithfulness`, `ComputeAnswerRelevance`, `ComputeContextRelevance`
- Options для включения метрик, кастомного LLM/embedder для оценки

**Unified Config Management**
- `Config` struct с суб-конфигами: Pipeline, Store, Embedder, LLM, Chunker, Reranker, Resilience, CostTracking
- `LoadConfig(path)`, `LoadConfigFromEnv()` — загрузка из YAML + env-оверрайды
- `NewPipelineFromConfig(ctx, cfg, deps...)` — полное конструирование Pipeline одной функцией
- Валидация known keys, sentinel-ошибки `ErrUnknownConfigKey`, `ErrMissingRequiredField`

**Semantic Chunker**
- `NewSemanticChunker(SemanticChunkerOptions)` — чанкинг на основе косинусного сходства эмбеддингов соседних предложений
- Параметры: Embedder, SimilarityThreshold, MinChunkSize, MaxChunkSize

**PII Guardrails**
- `NewDefaultPIIDetector(categories)`, `NewCompositePIIDetector(detectors...)` — паттерн-детекторы PII
- Категории: Email, Phone, SSN, CreditCard
- Интеграция в Pipeline: `PipelineOptions.PIIDetector`, PII redaction в Index/Query/Answer

**Sub-Query Decomposition**
- `NewLLMQueryDecomposer(llm)`, `NewRuleQueryDecomposer()` — LLM-based и rule-based декомпозиция
- `SearchBuilder.SubDecompose()` — fluent-метод для включения декомпозиции
- QueryDecomposer интерфейс + re-export

**Hierarchical Indices (ParentDocumentStore)**
- `ParentDocumentStore` capability: `UpsertParent`, `GetParentDocument`, `DeleteParent`
- `PipelineOptions.ParentContextEnabled` — опция включения parent-контекста
- Реализации: InMemory, pgvector, Qdrant, Pinecone

**Contextual Chunking**
- `NewContextualChunker(ContextualChunkerOptions)` — декоратор чанкера с обогащением контекстом
- Шаблонный подход: плейсхолдеры `{context}` и `{content}`

**Middleware Chain**
- `PipelineOptions.Middleware` — плагинная цепочка для стадий pipeline (chunking/embed/search/generate)
- Готовые middleware: `NewLoggingMiddleware`, `NewPIIDetectorMiddleware`
- Re-export: `Middleware`, `Handler`, `StageData`

**Pinecone Vector Store**
- `NewPineconeStore(PineconeOptions)` — полноценный VectorStore через Pinecone REST API
- Capabilities: DocumentStore, CollectionManager, ParentDocumentStore

**CJK Tokenization**
- `splitSentences` с поддержкой CJK-пунктуации (。！？，、)
- `BasicRuneChunker` — чанкинг по рунам для корректной обработки многобайтовых символов

**Architecture Quality Pass**
- Go generics: типизированный `router[T]` для 7 output-методов SearchBuilder
- `checkCtx` guard: nil context возвращает `ErrNilContext` вместо panic
- Error-returns: все конструкторы возвращают `error` вместо panic
- Централизованное `mapAppError` для маппинга internal → public sentinel-ошибок
- OTEL span lifecycle: `StageStart` возвращает `context.Context` для корректного закрытия span
- Единый `PipelineOptions` struct вместо `PipelineConfig` во всех конструкторах

### Changed
- Версия модуля зафиксирована: `v1.0.0` (SemVer)
- `PipelineConfig` полностью удалён (используйте `PipelineOptions`)
- nil context теперь возвращает `ErrNilContext`, а не panic

### Fixed
- gofmt во всех файлах проекта
- `health_test.go` — nil context в тесте (intentional, с nolint)
- `health.go` — удалена избыточная `comp := comp` (Go 1.22+ loopvar)
- `config.go` — `var fracMul float64 = 0.1` → `var fracMul = 0.1`
- `config.go` — unused ctx parameter в `NewPipelineFromConfig`
- `ratelimit_embedder_test.go` — удалено unused поле `mu`
- `health_test.go` — unused parameter `t`

### Docs
- ROADMAP.md — backlog приведён в соответствие с реализованными фичами

## [0.2.0] - 2026-06-11

### Added
- Search Builder API с fluent интерфейсом и generic routing (Search, Retrieve, Hybrid, Stream)
- LLM провайдеры: Mistral AI, DeepSeek (Chat Completions API)
- Mistral Embedder API
- SpecKeep workflow — формализованный процесс разработки через спецификации
- Полноценные примеры: memory, chromadb, qdrant, pgvector, milvus, weaviate, deepseek, mistral
- Pipeline E2E benchmarks
- Fuzz/property-based тесты для ядра (domain, pkg/draftrag)
- Contract tests для всех VectorStore (InMemory, ChromaDB, Qdrant, Milvus, Weaviate)
- Stream backpressure контроль с настраиваемым размером буфера
- Atomic update (UpdateDocument с rollback на ошибке)
- OpenTelemetry tracing (StageStart/StageEnd span lifecycle)
- slog adapter для structured logging
- Retry + Circuit Breaker с интеграционными тестами
- Pipeline coverage tests (error paths, edge cases)
- Production hardening checklist и документация (production.md)
- 10 туториалов на русском и английском (quickstart → production)

### Changed
- `PipelineConfig` → `PipelineOptions` (унифицированная конфигурация)
- Рефакторинг pipeline: выделены answer.go, query.go, stream.go, retrieval.go, batch.go, worker_pool.go
- Оптимизация структуры internal/application (декомпозиция на модули)
- `DedupSourcesByParentID` → `DedupByParentID` (сокращение)
- Обновлены все примеры под новый PipelineOptions API
- Улучшены тесты pipeline (более 700 строк coverage тестов)
- `.golangci.yml` — расширены правила линтинга (revive, dupl, gosec, gofmt)

### Docs
- 10 туториалов (EN + RU): quickstart, basic-rag, hybrid-search, metadata-filter, streaming, atomic-update, citations, observability, evaluation, production-hardening
- Обновлена README с примерами и ссылками
- Обновлена карта репозитория (REPOSITORY_MAP.md)
- Production checklist (docs/production.md)
- Compatibility matrix (docs/vector-stores.md)
- Roadmap (ROADMAP.md)

### Fixed
- gosec: crypto/md5 → crypto/sha1 в Qdrant ID generation
- dupl: вынесены валидаторы embedder/LLM опций в единый validateOptions
- gofmt/govet во всех пакетах
- nil context panic-tests корректно обрабатываются

## [0.1.0] - 2026-04-16

### Added
- Векторные хранилища: In-memory, PostgreSQL+pgvector, Qdrant, ChromaDB
- Embedder'ы: OpenAI-compatible API, Ollama, CachedEmbedder
- LLM провайдеры: OpenAI-compatible Responses API, Anthropic Claude, Ollama
- Search Builder API с fluent интерфейсом
- Стратегии retrieval: HyDE, MultiQuery, Hybrid (BM25+semantic)
- Фильтрация по метаданным и ParentID
- Retry + Circuit Breaker для production
- Observability hooks и OpenTelemetry интеграция
- Eval harness (Hit@K, MRR)
- Batch индексация с контролем concurrency
- MMR reranking и дедупликация
- Структурированное логирование с redaction
- Redis L2 кэш для эмбеддингов
- Примеры использования: chat, index-dir, pgvector, qdrant
- Документация: compatibility.md, production.md, getting-started.md
- Тестовое покрытие: internal/domain 100%, internal/application 83.3%, internal/infrastructure/vectorstore 60%
- Дополнительные тесты для InMemoryStore, HybridConfig, MetadataFilter, ParentIDFilter
- Тесты для контекстной отмены и косинусной схожести
- Тесты для конструкторов ChromaStore, QdrantStore, MilvusStore, WeaviateStore
- Интеграционные тесты для InMemoryStore с полным workflow
- Tooling: `.gitignore`, `Makefile`, `.golangci.yml`
- CI: GitHub Actions (`go test ./...` + `golangci-lint run ./...`)

### Changed
- Улучшены существующие тесты для pipeline методов
