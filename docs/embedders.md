# Embedder'ы

## OpenAI-compatible

Работает с любым API, совместимым с форматом `POST /v1/embeddings`: OpenAI, Azure OpenAI, Together AI, Mistral, self-hosted модели через LiteLLM и др.

```go
embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
    BaseURL: "https://api.openai.com",
    APIKey:  "sk-...",
    Model:   "text-embedding-ada-002",
})
```

### Опции

| Поле | Описание |
|---|---|
| `BaseURL` | **Обязательно.** Базовый URL API (без `/v1/embeddings`) |
| `APIKey` | **Обязательно.** Ключ API (заголовок `Authorization: Bearer`) |
| `Model` | **Обязательно.** Имя модели |
| `HTTPClient` | Кастомный `*http.Client`. `nil` → `http.DefaultClient` |
| `Timeout` | Таймаут на один вызов `Embed`. `0` → без таймаута |

### Популярные модели

| Провайдер | Модель | Размерность |
|---|---|---|
| OpenAI | `text-embedding-ada-002` | 1536 |
| OpenAI | `text-embedding-3-small` | 1536 |
| OpenAI | `text-embedding-3-large` | 3072 |
| Mistral | `mistral-embed` | 1024 |

### Ошибки конфигурации

Ошибки возвращаются из `Embed`, сопоставимы через `errors.Is(err, draftrag.ErrInvalidEmbedderConfig)`:
- `BaseURL` пустой или не содержит scheme/host
- `APIKey` пустой
- `Model` пустой
- `Timeout < 0`

---

## Ollama

Локальные модели через [Ollama](https://ollama.com/). Не требует API-ключей.

```go
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
    Model: "nomic-embed-text",
})
```

### Опции

| Поле | По умолчанию | Описание |
|---|---|---|
| `Model` | — | **Обязательно.** Имя модели |
| `BaseURL` | `http://localhost:11434` | URL Ollama сервера |
| `APIKey` | `""` | Опциональный ключ (для кастомных инстансов) |
| `HTTPClient` | `http.DefaultClient` | Кастомный HTTP клиент |
| `Timeout` | `0` | Таймаут на один вызов |

### Установка моделей

```bash
ollama pull nomic-embed-text     # 274M, хорошее качество
ollama pull mxbai-embed-large    # 670M, лучшее качество
ollama pull all-minilm           # 46M, быстрый
```

### Размерности

| Модель | Размерность |
|---|---|
| `nomic-embed-text` | 768 |
| `mxbai-embed-large` | 1024 |
| `all-minilm` | 384 |

**Важно**: убедитесь, что `EmbeddingDimension` в VectorStore совпадает с размерностью выбранной модели.

---

## Производительность

Для production-нагрузок рекомендуется:

**1. IndexBatch с параллелизмом:**

```go
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    IndexConcurrency:    8,   // 8 параллельных embed-запросов
    IndexBatchRateLimit: 100, // не более 100 запросов/сек
})

result, err := pipeline.IndexBatch(ctx, docs, 8)
```

**2. Таймауты:**

```go
embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
    // ...
    Timeout: 10 * time.Second,
})
```

---

## CachedEmbedder

In-memory LRU-кэш поверх любого Embedder. Повторные вызовы для одного текста не идут в API.

```go
embedder := draftrag.NewCachedEmbedder(
    draftrag.NewOpenAICompatibleEmbedder(...),
    draftrag.CacheOptions{MaxSize: 5000}, // 0 → 1000
)

pipeline := draftrag.NewPipeline(store, llm, embedder)
```

### Redis L2 (опционально)

Для горизонтального масштабирования можно включить Redis как second-level cache (L2).
Интеграция выполнена через адаптер-интерфейс, библиотека не привязана к конкретному Redis-клиенту.

```go
type myRedisClient struct{ /* ваш клиент */ }

func (c *myRedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) { /* ... */ return nil, nil }
func (c *myRedisClient) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    /* ... */
    return nil
}

embedder := draftrag.NewCachedEmbedder(
    base,
    draftrag.CacheOptions{
        MaxSize: 5000,
        Redis: draftrag.RedisCacheOptions{
            Client:    &myRedisClient{},
            TTL:       10 * time.Minute,      // 0 → без TTL
            KeyPrefix: "myapp:embedder:",      // "" → draftrag:embedder:
        },
    },
)
```

### Статистика

```go
stats := embedder.Stats()
fmt.Printf("hits=%d misses=%d evictions=%d hit_rate=%.2f\n",
    stats.Hits, stats.Misses, stats.Evictions, stats.HitRate(),
)
```

### Совместимость с RetryEmbedder

Кэш и retry можно комбинировать. Порядок: сначала кэш (не тратим retry-попытки на кэшированные запросы):

```go
base := draftrag.NewOpenAICompatibleEmbedder(...)
retry := draftrag.NewRetryEmbedder(base, draftrag.RetryOptions{})
cached := draftrag.NewCachedEmbedder(retry, draftrag.CacheOptions{})
```
