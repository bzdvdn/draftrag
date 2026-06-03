---
title: Метаданные и фильтры
related_examples:
  - examples/chromadb/
prerequisites:
  - Go 1.23+
  - Docker
---

# Метаданные и фильтры

При реальной эксплуатации документы содержат метаданные: автор, дата, категория, теги. draftRAG позволяет фильтровать поиск по метаданным, что критически важно для точности ответов.

## 1. Запустите ChromaDB

```yaml
# docker-compose.yml
services:
  chromadb:
    image: chromadb/chroma:0.5.20
    ports:
      - "8000:8000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/api/v1/heartbeat"]
      interval: 5s
      retries: 10
```

```bash
docker compose up -d
```

## 2. Создайте коллекцию

```go
opts := draftrag.ChromaDBOptions{
    BaseURL:    "http://localhost:8000",
    Collection: "FilterDemo",
    Dimension:  768,
}

if err := draftrag.CreateChromaCollection(ctx, opts); err != nil {
    log.Fatal(err)
}

store, err := draftrag.NewChromaDBStore(opts)
```

## 3. Индексируйте с метаданными

```go
docs := []draftrag.Document{
    {
        ID:      "f1",
        Content: "Go 1.21 ввёл улучшенное управление ошибками.",
        Metadata: map[string]string{
            "category":  "release",
            "author":    "team-go",
            "version":   "1.21",
            "year":      "2023",
        },
    },
    {
        ID:      "f2",
        Content: "Generics в Go позволяют писать типобезопасные коллекции.",
        Metadata: map[string]string{
            "category":  "feature",
            "author":    "community",
            "version":   "1.18",
            "year":      "2022",
        },
    },
}
pipeline.Index(ctx, docs)
```

## 4. Фильтрация по метаданным

```go
filter := draftrag.MetadataFilter{
    Fields: map[string]string{
        "category": "feature",
    },
}

result, _ := pipeline.Search("типобезопасные коллекции").
    TopK(5).
    Filter(filter).
    Retrieve(ctx)

for _, chunk := range result.Chunks {
    fmt.Printf("Чанк: %s (Score: %.4f)\n", chunk.Chunk.Content, chunk.Score)
}
```

## 5. Комбинирование фильтров

Добавьте несколько полей в `MetadataFilter.Fields` — все условия объединяются через AND:

```go
filter := draftrag.MetadataFilter{
    Fields: map[string]string{
        "category": "feature",
        "year":     "2022",
    },
}
```

## Доступность фильтрации по бэкендам

| Бэкенд | Metadata Filter |
|--------|----------------|
| Memory | — |
| PGVector | ✓ |
| Qdrant | ✓ |
| ChromaDB | ✓ |
| Weaviate | ✓ |
| Milvus | ✓ |

## Что дальше?

Переходите к [05-streaming.md](05-streaming.md) — потоковая генерация ответов.
