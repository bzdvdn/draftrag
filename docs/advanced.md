# Продвинутые возможности

## Цитаты

### Cite

Возвращает ответ + список использованных чанков со score:

```go
answer, sources, err := pipeline.Search("вопрос").TopK(5).Cite(ctx)
for i, r := range sources.Chunks {
    fmt.Printf("[%d] %s (score=%.3f)\n", i+1, r.Chunk.ParentID, r.Score)
    fmt.Printf("    %s\n", r.Chunk.Content[:100])
}
```

### InlineCite

LLM получает пронумерованный контекст `[1] текст... [2] текст...` и расставляет ссылки в ответе:

```go
answer, sources, citations, err := pipeline.Search("вопрос").TopK(5).InlineCite(ctx)
// answer: "Горутины создаются ключевым словом go [1]. Они мультиплексируются на OS-потоках [1][2]."

for _, c := range citations {
    fmt.Printf("[%d] ParentID=%s Score=%.3f\n",
        c.Number,
        c.Chunk.Chunk.ParentID,
        c.Chunk.Score,
    )
}
```

`citations` содержит только те чанки, которые реально упомянуты в ответе (LLM может не использовать все источники).

---

## Streaming

Требует LLM, реализующего `StreamingLLMProvider` (OpenAI-compatible, Anthropic):

```go
tokenChan, err := pipeline.Search("вопрос").TopK(5).Stream(ctx)
if errors.Is(err, draftrag.ErrStreamingNotSupported) {
    // fallback к обычному Answer
}
for token := range tokenChan {
    fmt.Print(token)
    // flush для SSE/http.Flusher в web-приложениях
}
```

### Streaming с inline-цитатами

```go
tokenChan, sources, citations, err := pipeline.Search("вопрос").TopK(5).StreamCite(ctx)
// sources и citations готовы сразу (поиск + построение контекста синхронны)
// токены генерируются асинхронно
for token := range tokenChan {
    fmt.Print(token)
}
```

### HTTP Server-Sent Events

```go
func handleChat(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    flusher := w.(http.Flusher)

    tokenChan, err := pipeline.Search(r.URL.Query().Get("q")).TopK(5).Stream(r.Context())
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    for token := range tokenChan {
        fmt.Fprintf(w, "data: %s\n\n", token)
        flusher.Flush()
    }
    fmt.Fprint(w, "data: [DONE]\n\n")
    flusher.Flush()
}
```

---

## Hybrid Search (BM25 + Semantic)

Комбинирует семантический поиск с keyword-поиском. Улучшает recall для точных терминов и аббревиатур.

Требует `HybridSearcher` (pgvector или in-memory).

```go
config := draftrag.DefaultHybridConfig()
// или кастомный:
config := draftrag.HybridConfig{
    UseRRF:         true,  // Reciprocal Rank Fusion (рекомендуется)
    SemanticWeight: 0.7,   // игнорируется при UseRRF=true
    RRFK:           60,    // константа RRF
}

result, err := pipeline.Search("вопрос").TopK(5).Hybrid(config).Retrieve(ctx)
answer, err := pipeline.Search("вопрос").TopK(5).Hybrid(config).Answer(ctx)
```

Если store не поддерживает hybrid search: `errors.Is(err, draftrag.ErrHybridNotSupported)`.

### HybridConfig

| Поле | По умолчанию | Описание |
|---|---|---|
| `UseRRF` | `true` | Reciprocal Rank Fusion |
| `SemanticWeight` | `0.7` | Вес semantic при weighted fusion |
| `RRFK` | `60` | Константа RRF: `score = 1/(k + rank)` |

---

## MMR Reranking

Maximal Marginal Relevance — выбирает разнообразные источники, балансируя релевантность и новизну информации:

```go
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    MMREnabled:       true,
    MMRLambda:        0.6,   // 0 = только разнообразие, 1 = только релевантность
    MMRCandidatePool: 20,    // запросить 20 кандидатов, выбрать topK через MMR
    DefaultTopK:      5,
})
```

При `MMRCandidatePool > 0` pipeline запрашивает `MMRCandidatePool` чанков из store и затем отбирает `topK` через MMR.

---

## Дедупликация источников

Убирает несколько чанков из одного документа, оставляя самый релевантный:

```go
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DedupSourcesByParentID: true,
    DefaultTopK:            10, // запросить больше, часть уберётся дедупликацией
})
```

Полезно когда один документ создаёт много чанков: без дедупликации ответ может ссылаться на 5 чанков из одного документа.

---

## Фильтрация по метаданным

### По ParentID (документам)

```go
// Искать только в документах из конкретного источника
result, err := pipeline.Search("вопрос").TopK(5).ParentIDs("doc-1", "doc-2", "doc-3").Retrieve(ctx)
answer, err := pipeline.Search("вопрос").TopK(5).ParentIDs("doc-1", "doc-2").Answer(ctx)
```

### По метаданным

AND-фильтр по произвольным полям метаданных документа:

```go
filter := draftrag.MetadataFilter{
    Fields: map[string]string{
        "category": "legal",
        "lang":     "ru",
        "year":     "2024",
    },
}

result, err := pipeline.Search("вопрос").TopK(5).Filter(filter).Retrieve(ctx)
answer, err := pipeline.Search("вопрос").TopK(5).Filter(filter).Answer(ctx)
```

Требует `VectorStoreWithFilters`. Если store не поддерживает: `errors.Is(err, draftrag.ErrFiltersNotSupported)`.

---

## Observability Hooks

Хуки вызываются на каждой стадии pipeline. Используйте для метрик, логирования, трейсинга:

```go
type Hooks struct {
    OnChunkingStart  func(ctx context.Context, op string)
    OnChunkingEnd    func(ctx context.Context, op string, duration time.Duration, err error)
    OnEmbedStart     func(ctx context.Context, op string)
    OnEmbedEnd       func(ctx context.Context, op string, duration time.Duration, err error)
    OnSearchStart    func(ctx context.Context, op string)
    OnSearchEnd      func(ctx context.Context, op string, duration time.Duration, err error)
    OnGenerateStart  func(ctx context.Context, op string)
    OnGenerateEnd    func(ctx context.Context, op string, duration time.Duration, err error)
}
```

```go
hooks := draftrag.Hooks{
    OnEmbedEnd: func(ctx context.Context, op string, d time.Duration, err error) {
        metrics.EmbedDuration.Observe(d.Seconds())
        if err != nil {
            metrics.EmbedErrors.Inc()
        }
    },
    OnGenerateEnd: func(ctx context.Context, op string, d time.Duration, err error) {
        slog.InfoContext(ctx, "llm generate", "op", op, "duration", d, "err", err)
    },
}

pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Hooks: hooks,
})
```

---

## Eval Harness

Оценка качества retrieval по набору вопросов с ожидаемыми источниками:

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag/eval"

cases := []eval.Case{
    {
        Question:        "Как работают горутины?",
        ExpectedParents: []string{"go-goroutines", "go-concurrency"},
    },
    {
        Question:        "Что такое каналы?",
        ExpectedParents: []string{"go-channels"},
    },
}

results, err := eval.Run(ctx, pipeline, cases, eval.Options{DefaultTopK: 5})

fmt.Printf("Hit@5: %.3f\n", results.HitAtK)
fmt.Printf("MRR:   %.3f\n", results.MRR)
```

### Метрики

| Метрика | Описание |
|---|---|
| `Hit@K` | Доля вопросов, для которых хотя бы один ожидаемый источник попал в топ-K |
| `MRR` | Mean Reciprocal Rank — средний обратный ранг первого попадания |

---

## Лимиты контекста

Ограничение размера контекста, передаваемого в LLM:

```go
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    MaxContextChars:  4000,  // не более 4000 символов в секции контекста
    MaxContextChunks: 8,     // не более 8 чанков
})
```

При превышении лимита чанки обрезаются по приоритету score (наиболее релевантные остаются).

---

## HyDE (Hypothetical Document Embeddings)

Улучшает recall для сложных вопросов: LLM сначала генерирует гипотетический ответ, затем его embedding используется для поиска (вместо embedding вопроса).

```go
result, err := pipeline.Search("Как работает сборщик мусора в Go?").TopK(5).HyDE().Retrieve(ctx)
answer, err := pipeline.Search("Как работает сборщик мусора в Go?").TopK(5).HyDE().Answer(ctx)
```

Когда использовать: вопросы технические или узкоспециализированные, где формулировка вопроса далека от формулировки ответа в документах.

HyDE совместим с Cite, InlineCite, Stream и другими методами:

```go
answer, sources, err := pipeline.Search("вопрос").TopK(5).HyDE().Cite(ctx)
```

---

## Multi-Query Retrieval

Генерирует N перефразировок вопроса, выполняет поиск по каждой, объединяет результаты через Reciprocal Rank Fusion (RRF). Уменьшает влияние конкретной формулировки на качество поиска.

```go
// MultiQuery(n) — количество перефразировок (рекомендуется 2-4)
result, err := pipeline.Search("горутины в Go").TopK(5).MultiQuery(3).Retrieve(ctx)
answer, err := pipeline.Search("горутины в Go").TopK(5).MultiQuery(3).Answer(ctx)
```

При `n=3` pipeline выполняет 4 поиска (оригинальный + 3 парафраза) и объединяет через RRF (k=60):

```
score(chunk) = Σ  1 / (60 + rank_i)
              per list i
```

Чанки сортируются по убыванию суммарного RRF-score, выбираются топ-K.

Совместим с HyDE (сначала применяется HyDE, потом MultiQuery):

```go
result, err := pipeline.Search("вопрос").TopK(5).HyDE().MultiQuery(2).Retrieve(ctx)
```

---

## Reranker

Post-retrieval переранжирование — позволяет подключить cross-encoder или любую другую модель переоценки релевантности. Вызывается после поиска в vector store.

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, chunks []RetrievedChunk) ([]RetrievedChunk, error)
}
```

Подключение:

```go
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Reranker: myReranker,
})
```

Reranker вызывается автоматически во всех методах поиска (Retrieve, Answer, Cite, HyDE, MultiQuery и т.д.).

Пример реализации (заглушка для тестирования):

```go
type scoreReranker struct{}

func (r *scoreReranker) Rerank(_ context.Context, _ string, chunks []draftrag.RetrievedChunk) ([]draftrag.RetrievedChunk, error) {
    // переупорядочить chunks по своей логике
    sort.Slice(chunks, func(i, j int) bool {
        return chunks[i].Score > chunks[j].Score
    })
    return chunks, nil
}
```

---

## Resilience (Retry + Circuit Breaker)

Обёртки с retry-логикой и circuit breaker для Embedder и LLM. Защищают от transient failures и каскадных отказов.

```go
// Defaults: MaxRetries=3, CBThreshold=5, CBTimeout=30s
embedder := draftrag.NewRetryEmbedder(
    draftrag.NewOpenAICompatibleEmbedder(...),
    draftrag.RetryOptions{},
)

llm := draftrag.NewRetryLLMProvider(
    draftrag.NewAnthropicLLM(...),
    draftrag.RetryOptions{},
)

pipeline := draftrag.NewPipeline(store, llm, embedder)
```

### RetryOptions

| Поле | По умолчанию | Описание |
|---|---|---|
| `MaxRetries` | `3` | Максимум повторных попыток |
| `BaseDelay` | `100ms` | Начальная задержка |
| `MaxDelay` | `10s` | Максимальная задержка |
| `Multiplier` | `2.0` | Множитель exponential backoff |
| `JitterFactor` | `0.25` | Доля случайной составляющей |
| `CBThreshold` | `5` | Порог ошибок для открытия CB |
| `CBTimeout` | `30s` | Время восстановления CB |

### Кастомные параметры

```go
embedder := draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{
    MaxRetries:   5,
    BaseDelay:    200 * time.Millisecond,
    CBThreshold:  10,
    CBTimeout:    60 * time.Second,
})
```

### Состояние Circuit Breaker

```go
re := draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{})

stats := re.CircuitBreakerStats()
fmt.Printf("state=%s failures=%d\n", re.CircuitBreakerState(), stats.FailureCount)

// errors.Is(err, draftrag.ErrCircuitOpen) — CB заблокировал запрос
```

### Классификация ошибок

По умолчанию все ошибки (кроме context.Canceled/DeadlineExceeded) считаются retryable. Явная пометка:

```go
// Пометить ошибку как non-retryable (не будет повторяться)
return draftrag.WrapNonRetryable(fmt.Errorf("invalid api key"))

// Пометить как retryable явно
return draftrag.WrapRetryable(fmt.Errorf("service unavailable"))

// Проверить
if draftrag.IsRetryable(err) {
    // ...
}
```
