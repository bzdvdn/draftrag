# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
