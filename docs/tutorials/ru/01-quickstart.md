---
title: Быстрый старт
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Быстрый старт с draftRAG

В этом руководстве вы создадите свою первую RAG-систему за 5 минут. Мы используем in-memory векторное хранилище и mock-LLM — никаких внешних зависимостей не требуется.

## 1. Создайте проект

```bash
mkdir my-first-rag && cd my-first-rag
go mod init my-first-rag
go get github.com/bzdvdn/draftrag@latest
```

## 2. Напишите код RAG-пайплайна

Создайте `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
    ctx := context.Background()

    // 1. In-memory хранилище (не требует Docker)
    store := draftrag.NewInMemoryStore()

    // 2. Embedder и LLM — используем mock для первого запуска
    embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
        Model: "nomic-embed-text",
    })
    llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
        Model: "llama3.2",
    })

    // 3. Собираем пайплайн
    pipeline, err := draftrag.NewPipelineWithChunker(
        store, llm, embedder,
        draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
            ChunkSize: 1000,
            Overlap:   100,
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 4. Индексируем документы
    docs := []draftrag.Document{
        {ID: "doc1", Content: "Go — это компилируемый язык программирования с явной типизацией."},
        {ID: "doc2", Content: "Горутины — это легковесные потоки, запускаемые с помощью ключевого слова go."},
        {ID: "doc3", Content: "Каналы в Go используются для синхронизации горутин."},
    }
    if err := pipeline.Index(ctx, docs); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Индексация завершена")

    // 5. Задаём вопрос
    answer, result, err := pipeline.Search("Что такое горутина?").TopK(3).Cite(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Вопрос: Что такое горутина?\nОтвет: %s\nНайдено источников: %d\n",
        answer, len(result.Chunks))
}
```

## 3. Запустите

```bash
go run .
```

Вывод будет содержать сгенерированный ответ и список источников с оценками релевантности.

## Что дальше?

Попробуйте заменить `LLM_PROVIDER` на реальный — установите [Ollama](https://ollama.ai) и запустите:

```bash
export LLM_PROVIDER=ollama
export OLLAMA_LLM_MODEL=llama3.2
export OLLAMA_EMBED_MODEL=nomic-embed-text
```

Затем переходите к [02-basic-rag.md](02-basic-rag.md) — подключим Qdrant для постоянного хранения.
