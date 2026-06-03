---
title: Гибридный поиск с Weaviate
related_examples:
  - examples/weaviate/
prerequisites:
  - Go 1.23+
  - Docker
---

# Гибридный поиск с Weaviate

Гибридный поиск комбинирует семантическое (векторное) и ключевое (BM25) ранжирование. Weaviate поддерживает это из коробки. В этом руководстве вы сравните гибридный поиск с обычным векторным.

## 1. Запустите Weaviate

```yaml
# docker-compose.yml
services:
  weaviate:
    image: semitechnologies/weaviate:1.27.5
    ports:
      - "8080:8080"
    environment:
      ENABLE_MODULES: ""
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: "true"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/v1/.well-known/ready"]
      interval: 5s
      retries: 10
```

```bash
docker compose up -d
```

## 2. Создайте коллекцию

```go
opts := draftrag.WeaviateOptions{
    Host:       "localhost:8080",
    Scheme:     "http",
    Collection: "HybridDemo",
}

if err := draftrag.CreateWeaviateCollection(ctx, opts); err != nil {
    log.Fatal(err)
}

store, err := draftrag.NewWeaviateStore(opts)
```

## 3. Индексируйте разнородные документы

```go
docs := []draftrag.Document{
    {ID: "h1", Content: "Weaviate — это векторная база данных с открытым исходным кодом."},
    {ID: "h2", Content: "BM25 — это алгоритм ранжирования, основанный на частоте термов."},
    {ID: "h3", Content: "Гибридный поиск объединяет векторную и ключевую релевантность."},
}
pipeline.Index(ctx, docs)
```

## 4. Сравните гибридный и векторный поиск

```go
// Векторный поиск (по умолчанию)
vectorResult, _ := pipeline.Search("алгоритм ранжирования").TopK(3).Retrieve(ctx)
fmt.Printf("Векторный поиск: %d результатов\n", len(vectorResult.Chunks))

// Гибридный поиск
hybridResult, _ := pipeline.Search("алгоритм ранжирования").
    TopK(3).
    Hybrid(draftrag.DefaultHybridConfig()).
    Retrieve(ctx)
fmt.Printf("Гибридный поиск: %d результатов\n", len(hybridResult.Chunks))
```

Гибридный поиск обычно возвращает более релевантные результаты, так как учитывает и семантику, и точное совпадение ключевых слов.

## 5. Тонкая настройка HybridConfig

```go
cfg := draftrag.HybridConfig{
    SemanticWeight: 0.7, // вес семантического поиска (0.0 — только BM25, 1.0 — только векторный)
    UseRRF:         true, // Reciprocal Rank Fusion
    RRFK:           60,   // константа сглаживания RRF
}
result, _ := pipeline.Search("гибридный поиск").
    TopK(5).
    Hybrid(cfg).
    Retrieve(ctx)
```

## Что дальше?

Переходите к [04-metadata-filter.md](04-metadata-filter.md) — фильтрация по метаданным с ChromaDB.
