# Векторные хранилища

## In-Memory

```go
store := draftrag.NewInMemoryStore()
```

Хранит данные в памяти процесса. Не требует внешних зависимостей. Подходит для прототипирования, тестов и небольших наборов документов (до ~10k чанков).

**Поддерживает**: `VectorStore`, `VectorStoreWithFilters`, `HybridSearcher`  
**Не сохраняет данные** между перезапусками.

---

## pgvector (PostgreSQL)

Production-ready хранилище с полной поддержкой SQL-фильтрации и hybrid search.

### Зависимости

```bash
go get github.com/jackc/pgx/v5
```

PostgreSQL 15+ с расширением pgvector.

### Миграции

Перед первым использованием нужно создать схему:

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/bzdvdn/draftrag/pkg/draftrag"
)

db, err := sql.Open("pgx", "postgres://user:pass@localhost/db?sslmode=disable")

err = draftrag.MigratePGVector(ctx, db, draftrag.PGVectorMigrateOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        CreateExtension:    true, // CREATE EXTENSION IF NOT EXISTS vector
    },
})
```

`MigratePGVector` идемпотентна — безопасно вызывать при каждом старте. Для production рекомендуется применять SQL-файлы из `pkg/draftrag/migrations/pgvector/` отдельным шагом деплоя.

### Создание store

```go
store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
    TableName:          "rag_chunks",
    EmbeddingDimension: 1536,
})

// Или с runtime ограничениями:
store := draftrag.NewPGVectorStoreWithRuntimeOptions(db,
    draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        IndexMethod:        "hnsw",  // "ivfflat" (по умолчанию) или "hnsw"
        Lists:              100,     // параметр ivfflat
    },
    draftrag.PGVectorRuntimeOptions{
        SearchTimeout:   2 * time.Second,
        UpsertTimeout:   5 * time.Second,
        MaxTopK:         50,
        MaxContentBytes: 64 * 1024,
    },
)
```

### PGVectorOptions

| Поле | По умолчанию | Описание |
|---|---|---|
| `TableName` | `draftrag_chunks` | Имя таблицы |
| `EmbeddingDimension` | — | **Обязательно.** Размерность векторов |
| `CreateExtension` | `false` | Создавать pgvector extension |
| `IndexMethod` | `ivfflat` | Метод индекса: `ivfflat` или `hnsw` |
| `Lists` | `100` | Параметр ivfflat |

### Docker Compose (быстрый старт)

```yaml
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypass
      POSTGRES_DB: mydb
    ports:
      - "5432:5432"
```

**Поддерживает**: `VectorStore`, `VectorStoreWithFilters`, `HybridSearcher`

---

## Qdrant

```go
opts := draftrag.QdrantOptions{
    URL:        "http://localhost:6333",  // по умолчанию
    Collection: "my_collection",          // обязательно
    Dimension:  1536,                     // обязательно
    Timeout:    10 * time.Second,         // для операций с коллекцией
}

store, err := draftrag.NewQdrantStore(opts)
```

### Управление коллекциями

```go
// Проверить существование
exists, err := draftrag.CollectionExists(ctx, opts)

// Создать
err = draftrag.CreateCollection(ctx, opts)

// Удалить
err = draftrag.DeleteCollection(ctx, opts)
```

### Docker

```bash
docker run -d -p 6333:6333 qdrant/qdrant
```

### Qdrant Cloud

Передайте URL облачного инстанса в `QdrantOptions.URL`.

**Поддерживает**: `VectorStore`, `VectorStoreWithFilters`

---

## ChromaDB

```go
store, err := draftrag.NewChromaDBStore(draftrag.ChromaDBOptions{
    BaseURL:    "http://localhost:8000",  // по умолчанию
    Collection: "my_collection",          // обязательно
    Dimension:  1536,                     // обязательно
})
```

Коллекцию нужно создать заранее через ChromaDB API или клиент.

### Docker

```bash
docker run -d -p 8000:8000 chromadb/chroma
```

**Поддерживает**: `VectorStore`, `VectorStoreWithFilters`

---

## Сравнение

| | In-Memory | pgvector | Qdrant | ChromaDB |
|---|---|---|---|---|
| Production | ✗ | ✅ | ✅ | ✅ |
| Постоянное хранение | ✗ | ✅ | ✅ | ✅ |
| Metadata filters | ✅ | ✅ | ✅ | ✅ |
| Hybrid search (BM25) | ✅ | ✅ | ✗ | ✗ |
| SQL-миграции | — | ✅ | — | — |
