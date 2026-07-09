---
status: extends
---

# Data Model: reranker-cross-encoder

## Изменения

### 1. `internal/domain/interfaces.go` — новый опциональный интерфейс

```go
// BatchReranker — опциональное расширение Reranker для batch-режима.
// Позволяет переранжировать один набор чанков по нескольким query одновременно.
// Pipeline проверяет реализацию через type assertion в multi-query режиме.
type BatchReranker interface {
    Reranker
    // RerankBatch принимает несколько query и один набор чанков.
    // Возвращает список результатов той же длины, что и queries.
    // Каждый результат — переранжированная версия chunks для соответствующего query.
    RerankBatch(ctx context.Context, queries []string, chunks []RetrievedChunk) ([][]RetrievedChunk, error)
}
```

### 2. `pkg/draftrag/reranker/errors.go` — sentinel

```go
// ErrInvalidRerankerConfig возвращается при невалидной конфигурации reranker'а.
var ErrInvalidRerankerConfig = errors.New("invalid reranker config")
```

### 3. `pkg/draftrag/reranker/cohere.go` — структуры

```go
// CohereRerankOptions — конфигурация для Cohere Rerank API v2.
type CohereRerankOptions struct {
    // APIKey — ключ авторизации (Bearer token). Обязательно.
    APIKey string

    // Model — имя модели. По умолчанию "rerank-english-v3.0".
    Model string

    // BaseURL — базовый URL API. По умолчанию "https://api.cohere.com/v2".
    BaseURL string

    // Timeout — таймаут HTTP-запроса. 0 = без таймаута.
    Timeout time.Duration

    // MaxRetries — количество retry при 429/5xx. По умолчанию 2.
    MaxRetries int

    // MaxTokensPerDoc — макс. токенов на документ. По умолчанию 4096.
    MaxTokensPerDoc int
}

// CohereReranker реализует domain.Reranker и domain.BatchReranker.
type CohereReranker struct {
    // содержит unexported поля: httpClient, options
}
```

### 4. Внутренние модели (unexported)

```go
// cohereRerankRequest — тело запроса к POST /v2/rerank
type cohereRerankRequest struct {
    Model           string   `json:"model"`
    Query           string   `json:"query"`
    Documents       []string `json:"documents"`
    TopN            *int     `json:"top_n,omitempty"`
    MaxTokensPerDoc *int     `json:"max_tokens_per_doc,omitempty"`
}

// cohereRerankResponse — ответ от POST /v2/rerank
type cohereRerankResponse struct {
    ID      string               `json:"id"`
    Results []cohereRerankResult `json:"results"`
    Meta    cohereRerankMeta     `json:"meta"`
}

type cohereRerankResult struct {
    Index          int     `json:"index"`
    RelevanceScore float64 `json:"relevance_score"`
}

type cohereRerankMeta struct {
    BilledUnits cohereBilledUnits `json:"billed_units"`
}

type cohereBilledUnits struct {
    SearchUnits float64 `json:"search_units"`
}
```

### 5. `pkg/draftrag/draftrag.go` — re-export

```go
// Re-export для BatchReranker
type BatchReranker = domain.BatchReranker
```

## Что НЕ меняется

- `domain.Reranker` — интерфейс остаётся без изменений
- `PipelineOptions` — поле `Reranker Reranker` уже существует
- `domain.RetrievedChunk` — не меняется
- Все store-реализации — не меняются
- Существующие тесты — проходят без изменений

## Cohere API Contract (внешний)

```
POST https://api.cohere.com/v2/rerank
Authorization: Bearer <APIKey>
Content-Type: application/json

{
    "model": "rerank-english-v3.0",
    "query": "search query",
    "documents": ["doc1", "doc2", ...],
    "top_n": null,
    "max_tokens_per_doc": 4096
}

Response 200:
{
    "id": "uuid",
    "results": [
        {"index": 3, "relevance_score": 0.999},
        {"index": 0, "relevance_score": 0.327}
    ],
    "meta": {"billed_units": {"search_units": 1}}
}
```
