---
title: Потоковая генерация
related_examples:
  - examples/qdrant/
prerequisites:
  - Go 1.23+
  - Docker
  - Ollama (для реального LLM)
---

# Потоковая генерация

Потоковая генерация (streaming) позволяет выводить ответ LLM токен за токеном, улучшая UX. draftRAG поддерживает streaming для провайдеров, реализующих `StreamingLLMProvider`.

## 1. Требования

Не все LLM поддерживают streaming. Используйте провайдер из списка:

| Провайдер | Streaming |
|-----------|-----------|
| Ollama | ✓ |
| OpenAI | ✓ |
| Anthropic | ✓ |
| Mock | ✓ |

## 2. Базовый streaming

```go
pipeline, err := draftrag.NewPipelineWithChunker(store, llm, embedder, chunker)
if err != nil {
    log.Fatal(err)
}

ch, err := pipeline.Search("Что такое горутина?").
    TopK(3).
    Stream(ctx)
if err != nil {
    log.Fatal(err)
}

for token := range ch {
    fmt.Print(token) // вывод токенов по мере генерации
}
fmt.Println()
```

## 3. Streaming с источниками

Метод `StreamSources` возвращает канал токенов и синхронно — найденные источники:

```go
ch, result, err := pipeline.Search("горутина").
    TopK(3).
    StreamSources(ctx)
if err != nil {
    log.Fatal(err)
}

// Источники доступны ДО начала streaming
fmt.Printf("Найдено источников: %d\n", len(result.Chunks))

for token := range ch {
    fmt.Print(token)
}
fmt.Println()
```

## 4. Streaming с цитированием

`StreamCite` объединяет streaming и инлайн-цитирование:

```go
ch, result, citations, err := pipeline.Search("горутина").
    TopK(3).
    StreamCite(ctx)
if err != nil {
    log.Fatal(err)
}

for token := range ch {
    fmt.Print(token)
}
fmt.Println()

for _, cit := range citations {
    fmt.Printf("[%d] %s (score: %.4f)\n",
        cit.Number, cit.Chunk.Chunk.Content, cit.Chunk.Score)
}
```

## 5. Интерактивный чат со streaming

```go
scanner := bufio.NewScanner(os.Stdin)
for {
    fmt.Print("\n> ")
    if !scanner.Scan() {
        break
    }
    q := scanner.Text()
    if q == "exit" {
        break
    }

    ch, _, err := pipeline.Search(q).TopK(3).StreamSources(ctx)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
        continue
    }

    for token := range ch {
        fmt.Print(token)
    }
    fmt.Println()
}
```

## Что дальше?

Переходите к [06-atomic-update.md](06-atomic-update.md) — атомарное обновление документов в pgvector.
