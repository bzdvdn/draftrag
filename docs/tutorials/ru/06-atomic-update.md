---
title: Атомарное обновление документов
related_examples:
  - examples/pgvector/
prerequisites:
  - Go 1.23+
  - Docker
---

# Атомарное обновление документов

В production данные меняются: документы редактируются, устаревают, удаляются. draftRAG поддерживает обновление и удаление документов на уровне пайплайна для хранилищ с поддержкой `DocumentStore`.

## 1. Запустите PostgreSQL с pgvector

```yaml
# docker-compose.yml
services:
  pgvector:
    image: pgvector/pgvector:pg16
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: draftrag
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      retries: 10
```

```bash
docker compose up -d
```

## 2. Подготовьте хранилище

```go
dsn := "postgres://postgres:postgres@localhost:5432/draftrag?sslmode=disable"
db, _ := sql.Open("postgres", dsn)

migrateOpts := draftrag.PGVectorMigrateOptions{
    PGVectorOptions: draftrag.PGVectorOptions{
        TableName:          "my_docs",
        EmbeddingDimension: 768,
        CreateExtension:    true,
    },
}

if err := draftrag.MigratePGVector(ctx, db, migrateOpts); err != nil {
    log.Fatal(err)
}

store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
    TableName:          "my_docs",
    EmbeddingDimension: 768,
})
```

## 3. Индексируйте начальные документы

```go
pipeline := draftrag.NewPipelineWithChunker(store, llm, embedder, chunker)

pipeline.Index(ctx, []draftrag.Document{
    {ID: "ug1", Content: "pgvector — это расширение PostgreSQL для векторного поиска."},
    {ID: "ug2", Content: "Обновление документа пересоздаёт все его чанки."},
})
```

## 4. Обновите документ

```go
err := pipeline.UpdateDocument(ctx, draftrag.Document{
    ID:      "ug1",
    Content: "pgvector — расширение PostgreSQL, добавляющее индексы IVFFlat и HNSW.",
})
if err != nil {
    log.Fatal(err)
}
```

`UpdateDocument` удаляет старые чанки документа `ug1` и индексирует новые. Если хранилище не поддерживает транзакционное обновление, возвращается `ErrUpdateNotAtomic`.

## 5. Удалите документ

```go
if err := pipeline.DeleteDocument(ctx, "ug2"); err != nil {
    log.Fatal(err)
}
```

## 6. Транзакционное обновление (pgvector)

Если хранилище реализует `TransactionalDocumentStore`, обновление выполняется в рамках одной транзакции:

```go
// pgvector поддерживает транзакции
// При ошибке между DeleteByParentID и Upsert все изменения откатываются
```

## Управление документами по бэкендам

| Бэкенд | UpdateDocument | DeleteDocument | Transactional |
|--------|---------------|----------------|---------------|
| Memory | ✓ | ✓ | — |
| PGVector | ✓ | ✓ | ✓ |
| Qdrant | ✓ | ✓ | — |
| ChromaDB | ✓ | ✓ | — |
| Weaviate | ✓ | ✓ | — |
| Milvus | ✓ | ✓ | — |

## Что дальше?

Переходите к [07-citations.md](07-citations.md) — цитирование источников в ответах.
