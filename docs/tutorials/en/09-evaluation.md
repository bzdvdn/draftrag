---
title: Evaluating RAG Quality
related_examples:
  - examples/memory/
prerequisites:
  - Go 1.23+
---

# Evaluating RAG Quality

Quantitative RAG evaluation is essential for production. draftRAG includes the `eval` package with metrics: HitRate, MRR, NDCG, Precision, Recall.

## 1. Define test cases

```go
import (
    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/eval"
)

cases := []eval.Case{
    {
        ID:                "goroutine",
        Question:          "What is a goroutine?",
        ExpectedParentIDs: []string{"doc2"},
        TopK:              5,
    },
    {
        ID:                "channel",
        Question:          "How do goroutines synchronize?",
        ExpectedParentIDs: []string{"doc3"},
        TopK:              5,
    },
    {
        ID:                "compiled",
        Question:          "Is Go interpreted?",
        ExpectedParentIDs: []string{"doc1"},
        TopK:              5,
    },
}
```

## 2. Run evaluation

```go
pipeline.Index(ctx, docs)

report, err := eval.Run(ctx, pipeline, cases, eval.Options{
    DefaultTopK:     5,
    EnableNDCG:      true,
    EnablePrecision: true,
    EnableRecall:    true,
})
```

## 3. Interpret metrics

```go
fmt.Printf("HitRate:   %.4f\n", report.Metrics.HitAtK)
fmt.Printf("MRR:       %.4f\n", report.Metrics.MRR)
fmt.Printf("NDCG:      %.4f\n", report.Metrics.NDCG)
fmt.Printf("Precision: %.4f\n", report.Metrics.Precision)
fmt.Printf("Recall:    %.4f\n", report.Metrics.Recall)
```

## 4. Export to JSON

```go
data, _ := report.MarshalJSON()
fmt.Println(string(data))
```

## 5. CI integration

```go
if report.Metrics.HitAtK < 0.8 {
    t.Errorf("HitRate below threshold: %.4f < 0.8", report.Metrics.HitAtK)
}
```

## Next

Proceed to [10-production-hardening.md](10-production-hardening.md) — production hardening.
