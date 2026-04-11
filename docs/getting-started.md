# Начало работы

## Установка

```bash
go get github.com/bzdvdn/draftrag
```

Для pgvector:

```bash
go get github.com/jackc/pgx/v5
```

Минимальная версия Go: **1.23**.

## Минимальный пример

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

pipeline := draftrag.NewPipeline(draftrag.NewInMemoryStore(), llm, embedder)

pipeline.Index(ctx, []draftrag.Document{
    {ID: "doc1", Content: "..."},
})

answer, err := pipeline.Answer(ctx, "Вопрос?")
```

## Типичные конфигурации

### С чанкингом

По умолчанию `Index` индексирует каждый документ как один чанк. Для автоматического разбиения — передайте `Chunker` в опциях:

```go
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DefaultTopK: 5,
    Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
        ChunkSize: 500,  // рун
        Overlap:   60,
    }),
})
```

### С pgvector

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, _ := sql.Open("pgx", "postgres://user:pass@localhost/mydb?sslmode=disable")

// Создать схему (идемпотентно)
draftrag.MigratePGVector(ctx, db, draftrag.PGVectorMigrateOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        CreateExtension:    true,
    },
})

store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
    TableName:          "rag_chunks",
    EmbeddingDimension: 1536,
})
```

### С Qdrant

```go
store, _ := draftrag.NewQdrantStore(draftrag.QdrantOptions{
    URL:        "http://localhost:6333",
    Collection: "my_collection",
    Dimension:  1536,
})
// Создать коллекцию заранее:
draftrag.CreateCollection(ctx, draftrag.QdrantOptions{...})
```

### С Ollama (локально, без API-ключей)

```go
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
    Model: "nomic-embed-text",  // ollama pull nomic-embed-text
})
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
    Model: "llama3.2",  // ollama pull llama3.2
})
// BaseURL по умолчанию: http://localhost:11434
```

### С Anthropic Claude

```go
llm := draftrag.NewAnthropicLLM(draftrag.AnthropicLLMOptions{
    BaseURL: "https://api.anthropic.com",
    APIKey:  "sk-ant-...",
    Model:   "claude-3-haiku-20240307",
})
```

## Следующие шаги

- [Концепции](concepts.md) — понять ключевые абстракции
- [Pipeline API](pipeline.md) — полный справочник методов
- [Продвинутые возможности](advanced.md) — streaming, цитаты, hybrid search
