# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- —

### Changed
- —

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
