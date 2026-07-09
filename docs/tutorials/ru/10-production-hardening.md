---
title: Промышленная эксплуатация
related_examples:
  - examples/memory/
  - examples/pgvector/
  - examples/qdrant/
prerequisites:
  - Go 1.23+
  - Docker
---

# Промышленная эксплуатация

Этот tutorial объединяет все предыдущие темы в единый production-ready пайплайн. Файл организован в подсекции для навигации.

<!-- Якорные ссылки -->
<a name="sec-10-1"></a>
<a name="sec-10-2"></a>
<a name="sec-10-3"></a>
<a name="sec-10-4"></a>
<a name="sec-10-5"></a>

## 10.1 Retry и Circuit Breaker

Для resilience используйте `NewRetryLLMProvider` и `NewRetryEmbedder`:

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag"

retryOpts := draftrag.RetryOptions{
    MaxRetries:   3,
    BaseDelay:    100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    JitterFactor: 0.25,
    CBThreshold:  5,      // circuit breaker открывается после 5 ошибок
    CBTimeout:    30 * time.Second,
}

llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{Model: "llama3.2"})
retryLLM := draftrag.NewRetryLLMProvider(llm, retryOpts)

embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{Model: "nomic-embed-text"})
retryEmbedder := draftrag.NewRetryEmbedder(embedder, retryOpts)
```

Можно проверять состояние circuit breaker:

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag"

stats := retryLLM.Stats() // возвращает CircuitBreakerStats
```

## 10.2 Наблюдаемость (OTel)

Подключите observability hooks из [08-observability.md](08-observability.md):

```go
hooks, _ := otel.NewHooks(otel.HooksOptions{
    TracerProvider: tp,
    MeterProvider:  mp,
})

pipeline, err := draftrag.NewPipelineWithOptions(store, retryLLM, retryEmbedder,
    draftrag.PipelineOptions{
        Chunker:             chunker,
        Hooks:               hooks,
        MaxContextChars:     4000,
        MaxContextChunks:    10,
        DedupByParentID: true,
    })
if err != nil {
    log.Fatal(err)
}
```

## 10.3 Кеширование эмбеддингов

Для повторяющихся текстов используйте `NewCachedEmbedder`:

```go
cachedEmbedder, err := draftrag.NewCachedEmbedder(retryEmbedder, draftrag.CacheOptions{
    MaxSize: 10000, // до 10000 записей в LRU-кеше
})
```

Статистика кеша:

```go
stats := cachedEmbedder.Stats()
fmt.Printf("Hit rate: %d/%d\n", stats.Hits, stats.Misses)
```

## 10.4 Регулировка скорости и конкурентность

Для массовой индексации настройте `PipelineOptions`:

```go
pipeline, err := draftrag.NewPipelineWithOptions(store, retryLLM, retryEmbedder,
    draftrag.PipelineOptions{
        Chunker:                 chunker,
        IndexConcurrency:        4,
        IndexBatchRateLimit:     10,   // не более 10 batch'ей в секунду
        IndexBatchRateLimitPerWorker: true,
        MMRCandidatePool:        20,   // MMR для разнообразия результатов
        MMREnabled:              true,
        MMRLambda:               0.5,
    })
if err != nil {
    log.Fatal(err)
}

// Batch-индексация с частичным результатом
result, err := pipeline.IndexBatch(ctx, allDocs, 10)
for _, e := range result.Errors {
    fmt.Fprintf(os.Stderr, "Ошибка в документе %s: %v\n", e.DocumentID, e.Error)
}
```

## 10.5 Ретушь секретов в логах

При логировании ошибок используйте функции ретуши из `domain`:

```go
import "github.com/bzdvdn/draftrag/internal/domain"

safeMsg := domain.RedactSecret(
    "connection failed with api_key=sk-abc123",
    "sk-abc123",
)
// safeMsg: "connection failed with api_key=***"
```

Для нескольких секретов:

```go
safeMsg := domain.RedactSecrets(rawMsg, apiKey, dbPassword)
```

## Полный пример production-пайплайна

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/bzdvdn/draftrag/pkg/draftrag"
    "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
)

func main() {
    ctx := context.Background()

    // 1. Хранилище
    store := draftrag.NewInMemoryStore()

    // 2. Resilience LLM
    llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{Model: "llama3.2"})
    retryLLM := draftrag.NewRetryLLMProvider(llm, draftrag.RetryOptions{
        MaxRetries: 3, CBThreshold: 5, CBTimeout: 30 * time.Second,
    })

    // 3. Кешированный embedder с retry
    embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{Model: "nomic-embed-text"})
    retryEmb := draftrag.NewRetryEmbedder(embedder, defaultRetry())
    cachedEmb, _ := draftrag.NewCachedEmbedder(retryEmb, draftrag.CacheOptions{MaxSize: 5000})

    // 4. OTel
    hooks, _ := otel.NewHooks(otel.HooksOptions{})
    _ = hooks

    // 5. Пайплайн
    pipeline, err := draftrag.NewPipelineWithOptions(store, retryLLM, cachedEmb,
        draftrag.PipelineOptions{
            Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
                ChunkSize: 1000, Overlap: 100,
            }),
            Hooks:               hooks,
            MaxContextChars:     4000,
            DedupByParentID: true,
            MMREnabled:          true,
        })
    if err != nil {
        log.Fatal(err)
    }

    // 6. Индексация и запрос
    pipeline.Index(ctx, []draftrag.Document{
        {ID: "prod1", Content: "Production RAG требует мониторинга, retry и кеширования."},
    })

    answer, _, _ := pipeline.Search("production RAG").TopK(3).Cite(ctx)
    log.Printf("Ответ: %s", answer)
}

func defaultRetry() draftrag.RetryOptions {
    return draftrag.RetryOptions{
        MaxRetries: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second,
        Multiplier: 2.0, JitterFactor: 0.25, CBThreshold: 5, CBTimeout: 30 * time.Second,
    }
}
```

## Заключение

Поздравляем! Вы освоили все возможности draftRAG:

1. [01-quickstart.md](01-quickstart.md) — быстрый старт с mock
2. [02-basic-rag.md](02-basic-rag.md) — базовый RAG с Qdrant
3. [03-hybrid-search.md](03-hybrid-search.md) — гибридный поиск с Weaviate
4. [04-metadata-filter.md](04-metadata-filter.md) — фильтрация по метаданным
5. [05-streaming.md](05-streaming.md) — потоковая генерация
6. [06-atomic-update.md](06-atomic-update.md) — обновление документов
7. [07-citations.md](07-citations.md) — цитирование
8. [08-observability.md](08-observability.md) — наблюдаемость
9. [09-evaluation.md](09-evaluation.md) — оценка качества
10. [10-production-hardening.md](10-production-hardening.md) — промышленная эксплуатация (этот файл)
