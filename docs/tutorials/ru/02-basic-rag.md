---
title: Базовый RAG с Qdrant
related_examples:
  - examples/qdrant/
prerequisites:
  - Go 1.23+
  - Docker
  - Ollama (опционально)
---

# Базовый RAG с Qdrant

В этом руководстве мы заменим in-memory хранилище на Qdrant — production-ready векторную базу данных. Вы научитесь управлять коллекциями и использовать docker-compose для инфраструктуры.

## 1. Запустите Qdrant

```yaml
# docker-compose.yml
services:
  qdrant:
    image: qdrant/qdrant:v1.12.4
    ports:
      - "6333:6333"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:6333/health"]
      interval: 5s
      retries: 10
```

```bash
docker compose up -d
```

## 2. Управление коллекцией

Перед созданием хранилища проверьте, существует ли коллекция:

```go
package main

import (
    "context"
    "log"

    "github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
    ctx := context.Background()

    opts := draftrag.QdrantOptions{
        URL:        "http://localhost:6333",
        Collection: "my_rag_docs",
        Dimension:  768,
    }

    exists, err := draftrag.CollectionExists(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }
    if !exists {
        if err := draftrag.CreateCollection(ctx, opts); err != nil {
            log.Fatal(err)
        }
        log.Println("Коллекция создана")
    }

    store, err := draftrag.NewQdrantStore(opts)
    if err != nil {
        log.Fatal(err)
    }
    _ = store
}
```

## 3. Индексация и запрос

```go
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{Model: "llama3.2"})
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{Model: "nomic-embed-text"})

pipeline, err := draftrag.NewPipelineWithChunker(store, llm, embedder,
    draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{ChunkSize: 1000, Overlap: 100}))
if err != nil {
    log.Fatal(err)
}

// Индексация
pipeline.Index(ctx, []draftrag.Document{
    {ID: "q1", Content: "Qdrant поддерживает фильтрацию по метаданным и пагинацию."},
})

// Запрос с citate
answer, result, _ := pipeline.Search("Что поддерживает Qdrant?").TopK(3).Cite(ctx)
log.Printf("Ответ: %s\nИсточники: %d", answer, len(result.Chunks))
```

## 4. Интерактивный режим

Добавьте цикл для диалога:

```go
scanner := bufio.NewScanner(os.Stdin)
for {
    fmt.Print("Вопрос (или 'exit'): ")
    if !scanner.Scan() {
        break
    }
    q := scanner.Text()
    if q == "exit" {
        break
    }
    answer, _, _ := pipeline.Search(q).TopK(3).Cite(ctx)
    fmt.Printf("Ответ: %s\n", answer)
}
```

## Провайдеры LLM

| Провайдер | Переменные окружения |
|-----------|---------------------|
| Mock | `LLM_PROVIDER=mock` |
| Ollama | `LLM_PROVIDER=ollama`, `OLLAMA_LLM_MODEL`, `OLLAMA_EMBED_MODEL` |
| OpenAI | `LLM_PROVIDER=openai`, `OPENAI_API_KEY`, `OPENAI_EMBED_MODEL`, `OPENAI_LLM_MODEL` |
| Anthropic | `LLM_PROVIDER=anthropic`, `ANTHROPIC_API_KEY`, `ANTHROPIC_LLM_MODEL` |

## Что дальше?

Переходите к [03-hybrid-search.md](03-hybrid-search.md) — гибридный поиск с Weaviate.
