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
store := draftrag.NewPGVectorStoreWithOptions(db, draftrag.PGVectorStoreOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "rag_chunks",
        EmbeddingDimension: 1536,
        IndexMethod:        "hnsw",  // "ivfflat" (по умолчанию) или "hnsw"
        Lists:              100,     // параметр ivfflat
    },
    Runtime: draftrag.PGVectorRuntimeOptions{
        SearchTimeout:   2 * time.Second,
        UpsertTimeout:   5 * time.Second,
        MaxTopK:         50,
        MaxContentBytes: 64 * 1024,
    },
})
```

Миграция:

- было: `NewPGVectorStoreWithRuntimeOptions(db, opts, runtime)` (deprecated)
- стало: `NewPGVectorStoreWithOptions(db, PGVectorStoreOptions{PGVectorOptions: opts, Runtime: runtime})`

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

## Weaviate

Weaviate-backed store через публичный API `pkg/draftrag`.

Статус: **experimental** (см. `docs/compatibility.md`). Подробная инструкция и troubleshooting: `docs/weaviate.md`.

```go
opts := draftrag.WeaviateOptions{
    Host:       "localhost:8080", // обязательно
    Scheme:     "http",           // "" → http
    Collection: "MyChunks",       // обязательно (Weaviate class)
    APIKey:     "",               // опционально (Weaviate Cloud)
    Timeout:    10 * time.Second, // HTTP таймаут для schema операций
}

store, err := draftrag.NewWeaviateStore(opts)
```

### Управление коллекцией (рекомендация для production)

В production schema/создание коллекции обычно делается отдельным шагом деплоя (deploy job/init container), а не при старте сервиса:

```go
exists, err := draftrag.WeaviateCollectionExists(ctx, opts)
if !exists {
    err = draftrag.CreateWeaviateCollection(ctx, opts)
}
```

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

<!-- @sk-task api-consistency-pass#T3.5: docs sync — Milvus section (DEC-008, RQ-008, AC-013) -->

## Milvus / Zilliz

Высокопроизводительный distributed векторный поиск (REST API).

**Статус**: ⚠️ public API в разработке (`pkg/draftrag.NewMilvusStore` пока отсутствует, см. `internal/infrastructure/vectorstore/milvus.go`).

Подробная инструкция и troubleshooting: `docs/milvus.md` (планируется).

```go
// Внутренний API (не рекомендуется к прямому использованию):
store, err := vectorstore.NewMilvusStore(baseURL, collection, token)
```

Коллекцию нужно создать заранее через Milvus API или клиент. Поддерживает basic retrieval, фильтры по метаданным и ParentID. **Hybrid search не поддерживается** в публичной модели.

### Docker (быстрый старт)

```bash
docker run -d -p 19530:19530 -p 9091:9091 milvusdb/milvus:latest
```

**Поддерживает** (внутренний API): `VectorStore`, `VectorStoreWithFilters`

---

## Capability-таблица

<!-- @sk-task docs-and-examples#T3.6: ссылки на examples/ в первой колонке capability-таблицы (AC-015) -->
<!-- @sk-task api-consistency-pass#T3.5: docs sync — capability-таблица 6×6 = 36 ячеек (DEC-008, RQ-008, AC-014) -->

| | [In-Memory](examples/memory/) | [pgvector](examples/pgvector/) | [Qdrant](examples/qdrant/) | [ChromaDB](examples/chromadb/) | [Weaviate](examples/weaviate/) | [Milvus](examples/milvus/) |
|---|---|---|---|---|---|---|
| **Basic retrieval** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Metadata filter** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **ParentID filter** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Hybrid (BM25 + semantic)** | ✅ | ✅ | ❌ | ❌ | ❌ | ❌[^hybrid] |
| **DeleteByParentID** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Collection management** | N/A | ✅ (SQL миграции) | ✅ | ✅ | ✅ | ✅ |

**Условные обозначения**:
- ✅ — поддерживается
- ❌ — не поддерживается (см. footnote для исключений)
- N/A — не применимо (нет коллекций/схем)

**Footnotes**:

[^hybrid]: Hybrid search (BM25) на уровне публичного API не поддерживается для Weaviate и Milvus. Внутренние реализации `internal/infrastructure/vectorstore/{weaviate,milvus}.go` содержат `SearchHybrid`-методы, но они не экспонированы в `pkg/draftrag/`. Используйте pgvector (production-ready) или In-Memory (прототипирование) для гибридного поиска.

**Несовместимые комбинации**:

- `Hybrid(cfg)` в SearchBuilder + Weaviate/Milvus/Qdrant/ChromaDB → `draftrag.ErrHybridNotSupported` (в runtime).
- `UpdateDocument` + In-Memory/Qdrant/ChromaDB/Weaviate/Milvus → `draftrag.ErrUpdateNotAtomic` (нет транзакционного store; T3.2 best-effort path). Только pgvector гарантирует атомарность.

---

## Production status

| Store | Статус |
|---|---|
| In-Memory | для тестов и прототипов |
| pgvector | production-ready |
| Qdrant | production-ready |
| ChromaDB | production-ready |
| Weaviate | ⚠️ experimental (см. `docs/compatibility.md` и `docs/weaviate.md`) |
| Milvus | ⚠️ public API в разработке (см. `internal/infrastructure/vectorstore/milvus.go`) |

