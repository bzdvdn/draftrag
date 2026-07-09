---
title: Оценка качества RAG
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Оценка качества RAG

Количественная оценка качества RAG — необходимость для production. draftRAG включает пакет `eval` для автоматического тестирования с метриками HitRate, MRR, NDCG, Precision и Recall.

## 1. Определите тестовые кейсы

```go
import (
    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/eval"
)

cases := []eval.Case{
    {
        ID:                "goroutine",
        Question:          "Что такое горутина?",
        ExpectedParentIDs: []string{"doc2"},
        TopK:              5,
    },
    {
        ID:                "channel",
        Question:          "Как синхронизировать горутины?",
        ExpectedParentIDs: []string{"doc3"},
        TopK:              5,
    },
    {
        ID:                "compiled",
        Question:          "Go — это интерпретируемый язык?",
        ExpectedParentIDs: []string{"doc1"},
        TopK:              5,
    },
}
```

## 2. Запустите оценку

```go
pipeline, err := draftrag.NewPipelineWithChunker(store, llm, embedder, chunker)
if err != nil {
    log.Fatal(err)
}
pipeline.Index(ctx, docs)

report, err := eval.Run(ctx, pipeline, cases, eval.Options{
    DefaultTopK:     5,
    EnableNDCG:      true,
    EnablePrecision: true,
    EnableRecall:    true,
})
if err != nil {
    log.Fatal(err)
}
```

## 3. Интерпретируйте метрики

```go
fmt.Printf("=== Результаты оценки ===\n")
fmt.Printf("Всего кейсов:  %d\n", report.Metrics.TotalCases)
fmt.Printf("HitRate:        %.4f\n", report.Metrics.HitAtK)
fmt.Printf("MRR:            %.4f\n", report.Metrics.MRR)
fmt.Printf("NDCG:           %.4f\n", report.Metrics.NDCG)
fmt.Printf("Precision:      %.4f\n", report.Metrics.Precision)
fmt.Printf("Recall:         %.4f\n", report.Metrics.Recall)

for _, cr := range report.Cases {
    fmt.Printf("\nКейс %s: %s\n", cr.CaseID, map[bool]string{true: "✓", false: "✗"}[cr.Found])
    if cr.Found {
        fmt.Printf("  Ранг: %d\n", cr.Rank)
    }
}
```

## 4. Экспорт в JSON

```go
data, _ := report.MarshalJSON()
fmt.Println(string(data))
```

## 5. Интеграция в CI

Добавьте пороговые значения в тесты:

```go
if report.Metrics.HitAtK < 0.8 {
    t.Errorf("HitRate ниже порога: %.4f < 0.8", report.Metrics.HitAtK)
}
```

## Что дальше?

Переходите к [10-production-hardening.md](10-production-hardening.md) — промышленная эксплуатация.
