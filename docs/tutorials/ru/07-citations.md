---
title: Цитирование источников
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Цитирование источников

Прозрачность ответов — ключевое требование для RAG в enterprise. draftRAG предоставляет два режима цитирования: пост-фактум (`Cite`) и инлайн-цитирование (`InlineCite`).

## 1. Базовое цитирование (Cite)

Метод `Cite` возвращает ответ и список источников с оценками релевантности:

```go
answer, result, err := pipeline.Search("Что такое горутина?").
    TopK(3).
    Cite(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Ответ: %s\n", answer)
fmt.Printf("Источников: %d\n", len(result.Chunks))

for i, chunk := range result.Chunks {
    fmt.Printf("[%d] Score: %.4f | %s\n",
        i+1, chunk.Score, chunk.Chunk.Content)
}
```

## 2. Инлайн-цитирование (InlineCite)

`InlineCite` добавляет в ответ маркеры `[1]`, `[2]` и т.д., указывающие на конкретные чанки:

```go
answer, result, citations, err := pipeline.Search("горутина").
    TopK(3).
    InlineCite(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Ответ с цитатами:\n%s\n\n", answer)
fmt.Println("Источники:")
for _, cit := range citations {
    fmt.Printf("  [%d] %s (relevance: %.4f)\n",
        cit.Number, cit.Chunk.Chunk.Content, cit.Chunk.Score)
}
```

## 3. Парсинг инлайн-цитат

Для отображения в UI можно распарсить цитаты регулярным выражением:

```go
import "regexp"

re := regexp.MustCompile(`\[(\d+)\]`)
matches := re.FindAllStringSubmatch(answer, -1)
for _, m := range matches {
    fmt.Printf("Ссылка на источник №%s\n", m[1])
}
```

## 4. Streaming с цитатами

`StreamCite` комбинирует потоковую генерацию и цитирование:

```go
ch, result, citations, err := pipeline.Search("горутина").
    TopK(3).
    StreamCite(ctx)

for token := range ch {
    fmt.Print(token)
}
fmt.Println()

for _, cit := range citations {
    fmt.Printf("[%d] \n", cit.Number)
}
```

## 5. Метаданные в цитатах

Чанки содержат `ParentID` (ID исходного документа) и `Metadata`:

```go
for _, cit := range citations {
    chunk := cit.Chunk.Chunk
    fmt.Printf("[%d] doc=%s | pos=%d | %s\n",
        cit.Number, chunk.ParentID, chunk.Position, chunk.Content)
}
```

## Что дальше?

Переходите к [08-observability.md](08-observability.md) — наблюдаемость через OpenTelemetry.
