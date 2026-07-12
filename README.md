# draftRAG

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](go.mod)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue)](LICENSE)

**draftRAG** is a Go library for building Retrieval-Augmented Generation (RAG) pipelines. It provides a unified API for document indexing, semantic search, and answer generation across multiple backends.

> Русская версия: [README.ru.md](README.ru.md)

## Features

**Vector Stores**

- **In-memory** — fast prototyping and testing
- **PostgreSQL + pgvector** — production-ready with hybrid search (BM25 + semantic), metadata filters, auto-migrations
- **Qdrant** — production-ready with payload filters and collection management
- **ChromaDB** — vector search with ParentID filters
- **Weaviate** — basic retrieval, metadata/ParentID filters, collection management
- **Milvus / Zilliz** — high-performance distributed vector search

Capability table: [docs/en/vector-stores.md](docs/en/vector-stores.md)

**Embedders & LLM Providers**

- OpenAI-compatible, Anthropic Claude, Ollama (local)
- CachedEmbedder (LRU + optional Redis L2)
- Retry + Circuit Breaker wrappers

**Search Builder — fluent API for all scenarios**
| Method | Returns | Description |
|--------|---------|-------------|
| `.Retrieve(ctx)` | `RetrievalResult` | Search only, no LLM |
| `.Answer(ctx)` | `string` | Answer without sources |
| `.Cite(ctx)` | `string, RetrievalResult` | Answer + source list |
| `.InlineCite(ctx)` | `string, RetrievalResult, []Citation` | Answer with `[n]` inline citations |
| `.Stream(ctx)` | `<-chan string` | Streaming answer |
| `.StreamCite(ctx)` | `<-chan string, RetrievalResult, []InlineCitation` | Streaming with inline citations |

**Retrieval strategies**: `.HyDE()`, `.MultiQuery(n)`, `.Hybrid(cfg)`, `.ParentIDs(ids...)`, `.Filter(f)`

## Quick Start

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag"

embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "text-embedding-ada-002",
})
llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "gpt-4o-mini",
})

pipeline, err := draftrag.NewPipeline(draftrag.NewInMemoryStore(), llm, embedder)
if err != nil {
    log.Fatal(err)
}

pipeline.Index(ctx, []draftrag.Document{
    {ID: "doc1", Content: "..."},
})

answer, err := pipeline.Answer(ctx, "Question?")
```

## Documentation

| English                                       | Russian                                         |
| --------------------------------------------- | ----------------------------------------------- |
| [Getting Started](docs/en/getting-started.md) | [Начало работы](docs/ru/getting-started.md)     |
| [Concepts](docs/en/concepts.md)               | [Концепции](docs/ru/concepts.md)                |
| [Pipeline API](docs/en/pipeline.md)           | [Pipeline API](docs/ru/pipeline.md)             |
| [Advanced Features](docs/en/advanced.md)      | [Продвинутые возможности](docs/ru/advanced.md)  |
| [Vector Stores](docs/en/vector-stores.md)     | [Векторные хранилища](docs/ru/vector-stores.md) |
| [Embedders](docs/en/embedders.md)             | [Embedder'ы](docs/ru/embedders.md)              |
| [LLM Providers](docs/en/llm-providers.md)     | [LLM провайдеры](docs/ru/llm-providers.md)      |
| [Chunking](docs/en/chunking.md)               | [Чанкинг](docs/ru/chunking.md)                  |
| [Weaviate](docs/en/weaviate.md)               | [Weaviate](docs/ru/weaviate.md)                 |
| [Compatibility](docs/en/compatibility.md)     | [Совместимость](docs/ru/compatibility.md)       |
| [Production Checklist](docs/en/production.md) | [Production checklist](docs/ru/production.md)   |

### Tutorials

| #   | English                                                              | Russian                                                              |
| --- | -------------------------------------------------------------------- | -------------------------------------------------------------------- |
| 01  | [Quickstart](docs/tutorials/en/01-quickstart.md)                     | [Быстрый старт](docs/tutorials/ru/01-quickstart.md)                  |
| 02  | [Basic RAG](docs/tutorials/en/02-basic-rag.md)                       | [Basic RAG](docs/tutorials/ru/02-basic-rag.md)                       |
| 03  | [Hybrid Search](docs/tutorials/en/03-hybrid-search.md)               | [Гибридный поиск](docs/tutorials/ru/03-hybrid-search.md)             |
| 04  | [Metadata Filter](docs/tutorials/en/04-metadata-filter.md)           | [Фильтрация метаданных](docs/tutorials/ru/04-metadata-filter.md)     |
| 05  | [Streaming](docs/tutorials/en/05-streaming.md)                       | [Потоковый ответ](docs/tutorials/ru/05-streaming.md)                 |
| 06  | [Atomic Update](docs/tutorials/en/06-atomic-update.md)               | [Атомарное обновление](docs/tutorials/ru/06-atomic-update.md)        |
| 07  | [Citations](docs/tutorials/en/07-citations.md)                       | [Цитаты](docs/tutorials/ru/07-citations.md)                          |
| 08  | [Observability](docs/tutorials/en/08-observability.md)               | [Наблюдаемость](docs/tutorials/ru/08-observability.md)               |
| 09  | [Evaluation](docs/tutorials/en/09-evaluation.md)                     | [Оценка качества](docs/tutorials/ru/09-evaluation.md)                |
| 10  | [Production Hardening](docs/tutorials/en/10-production-hardening.md) | [Production hardening](docs/tutorials/ru/10-production-hardening.md) |

## Package Structure

```
pkg/draftrag/          — public API (use this)
pkg/draftrag/eval/     — eval harness (Hit@K, MRR)
internal/
  domain/              — interfaces and data models
  application/         — pipeline business logic
  infrastructure/      — implementations: vectorstore, embedder, llm, chunker, resilience
```

## Examples

| Example                                 | Backend             | Docker | LLM | Description                             |
| --------------------------------------- | ------------------- | ------ | --- | --------------------------------------- |
| [examples/memory](examples/memory/)     | In-memory           | No     | Any | Quick start without Docker              |
| [examples/pgvector](examples/pgvector/) | PostgreSQL+pgvector | Yes    | Any | Production-ready, hybrid search         |
| [examples/qdrant](examples/qdrant/)     | Qdrant              | Yes    | Any | Payload filters, auto-create collection |
| [examples/chromadb](examples/chromadb/) | ChromaDB            | Yes    | Any | Vector search with metadata             |
| [examples/weaviate](examples/weaviate/) | Weaviate            | Yes    | Any | GraphQL API, class management           |
| [examples/milvus](examples/milvus/)     | Milvus              | Yes    | Any | High-performance distributed            |
| [examples/semantic-chunking](examples/semantic-chunking/) | In-memory | No | Any | Semantic chunking demo                  |
| [examples/sub-query-decomposition](examples/sub-query-decomposition/) | In-memory | No | Any | Multi-query decomposition     |

## Installation

```bash
go get github.com/bzdvdn/draftrag
```

Minimum Go version: **1.23**. For pgvector: `go get github.com/jackc/pgx/v5`.

## License

Apache License 2.0
