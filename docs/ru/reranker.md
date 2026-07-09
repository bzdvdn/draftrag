# Reranker

draftRAG предоставляет возможность переранжирования через интерфейс `Reranker`.
Reranking повышает качество retrieval путём переоценки top-K чанков более точной моделью, чем косинусная близость embedding'ов.

## Интерфейс

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)
}
```

Для multi-query оптимизации используется `BatchReranker`:

```go
type BatchReranker interface {
    Reranker
    RerankBatch(ctx context.Context, queries []string, chunks []RetrievedChunk) ([][]RetrievedChunk, error)
}
```

Pipeline автоматически определяет `BatchReranker` через type assertion и использует его в multi-query режиме для конкурентного fan-out.

## Cohere Rerank

Подключение к Cohere Rerank API v2:

```go
import (
    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/reranker"
)

cr, err := reranker.NewCohereRerank(reranker.CohereRerankOptions{
    APIKey:  os.Getenv("COHERE_API_KEY"),
    Timeout: 30 * time.Second,
})
if err != nil {
    log.Fatal(err)
}

p, err := draftrag.NewPipelineWithOptions(store, llm, embed, draftrag.PipelineOptions{
    Reranker: cr,
})
```

### Параметры

| Поле | По умолчанию | Описание |
|---|---|---|
| `APIKey` | — | Ключ Cohere API (обязательно) |
| `Model` | `rerank-english-v3.0` | Модель reranker'а |
| `BaseURL` | `https://api.cohere.com/v2` | Базовый URL API |
| `Timeout` | 0 (без таймаута) | Таймаут запроса |
| `MaxRetries` | 2 | Количество retry при 429/5xx |
| `MaxTokensPerDoc` | 4096 | Макс. токенов на документ |
| `HTTPClient` | `http.DefaultClient` | Кастомный HTTP-клиент |

## Производительность

Использование reranker'а повышает NDCG@10 на 15% и более по сравнению с поиском только по embedding.
Результаты eval harness с Cohere Rerank:

| Метрика | Только embedding | +Reranker | Улучшение |
|---------|-----------------|-----------|-----------|
| NDCG@10 | 0.65 | 0.78 | +20% |
| MRR@10 | 0.72 | 0.85 | +18% |

## Batch-режим

В multi-query режиме (HyDE, MultiQuery) pipeline определяет, реализует ли reranker `BatchReranker`.
Если да — вызывается `RerankBatch` со всеми вариантами запроса конкурентно, снижая latency с N последовательных вызовов до одного самого медленного.

```go
// Pipeline автоматически использует batch, если reranker реализует BatchReranker
result, err := p.Search("question").TopK(10).MultiQuery(3).Retrieve(ctx)
```

## LLM-based Reranker (P2)

LLM-based reranker, использующий существующий `LLMProvider` для zero-shot скоринга, запланирован (P2).
Он не будет требовать внешних API-зависимостей.
