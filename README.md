# draftRAG

Go-библиотека для построения RAG (Retrieval-Augmented Generation) пайплайнов. Предоставляет единый API для индексации документов, семантического поиска и генерации ответов с различными backend'ами.

## Возможности

**Векторные хранилища**
- **In-memory** — быстрое прототипирование и тесты
- **PostgreSQL + pgvector** — production-ready с гибридным поиском (BM25 + semantic), фильтрами по метаданным и автомиграциями
- **Qdrant** — production-ready с payload-фильтрами и управлением коллекциями
- **ChromaDB** — векторный поиск с фильтрами по ParentID

**Embedder'ы**
- **OpenAI-compatible API** — `text-embedding-ada-002`, `text-embedding-3-*` и любые совместимые
- **Ollama** — локальные модели (`nomic-embed-text`, `mxbai-embed-large` и др.)
- **CachedEmbedder** — LRU-кэш эмбеддингов поверх любого embedder'а

**LLM провайдеры**
- **OpenAI-compatible Responses API** — OpenAI, Azure, и другие совместимые сервисы
- **Anthropic Claude** — нативный Messages API с поддержкой streaming
- **Ollama** — локальные модели

**Search Builder — единый fluent API для всех сценариев**

| Метод | Возвращает | Описание |
|-------|-----------|----------|
| `.Retrieve(ctx)` | `RetrievalResult` | Только поиск, без LLM |
| `.Answer(ctx)` | `string` | Ответ без источников |
| `.Cite(ctx)` | `string, RetrievalResult` | Ответ + список источников |
| `.InlineCite(ctx)` | `string, RetrievalResult, []Citation` | Ответ с inline-цитатами `[1]` |
| `.Stream(ctx)` | `<-chan string` | Потоковый ответ |
| `.StreamSources(ctx)` | `<-chan string, RetrievalResult` | Потоковый ответ + источники |
| `.StreamCite(ctx)` | `<-chan string, RetrievalResult, []Citation` | Потоковый ответ + inline-цитаты |

**Стратегии retrieval** (модификаторы Builder)
- `.HyDE()` — Hypothetical Document Embeddings, улучшает recall
- `.MultiQuery(n)` — n перефраз вопроса с объединением результатов
- `.Hybrid(cfg)` — BM25 + semantic (только pgvector)
- `.ParentIDs(ids...)` — фильтр по родительским документам
- `.Filter(f)` — фильтр по произвольным метаданным

**Production-ready**
- **Retry + Circuit Breaker** — `RetryEmbedder`, `RetryLLMProvider` с exponential backoff
- **Observability hooks** — latency и ошибки по стадиям: chunking / embed / search / generate
- **Eval harness** — Hit@K, MRR для оффлайн-оценки качества retrieval
- **Batch indexing** — `IndexBatch` с контролем concurrency и rate limiting
- **MMR reranking** — диверсификация контекста (Maximal Marginal Relevance)
- **Дедупликация** — устранение дублей из retrieval результатов

## Установка

```bash
go get github.com/bzdvdn/draftrag
```

Минимальная версия Go: **1.23**.

Для pgvector дополнительно:

```bash
go get github.com/jackc/pgx/v5
```

## Быстрый старт

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
    ctx := context.Background()

    embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
        BaseURL: "https://api.openai.com",
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        Model:   "text-embedding-ada-002",
    })
    llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
        BaseURL: "https://api.openai.com",
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        Model:   "gpt-4o-mini",
    })

    pipeline := draftrag.NewPipelineWithOptions(
        draftrag.NewInMemoryStore(), llm, embedder,
        draftrag.PipelineOptions{
            DefaultTopK: 3,
            Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
                ChunkSize: 500,
                Overlap:   60,
            }),
        },
    )

    docs := []draftrag.Document{
        {ID: "doc1", Content: "Go использует горутины для конкурентного программирования..."},
        {ID: "doc2", Content: "Каналы в Go обеспечивают синхронизацию между горутинами..."},
    }
    if err := pipeline.Index(ctx, docs); err != nil {
        panic(err)
    }

    answer, sources, err := pipeline.Search("Как работают горутины?").TopK(3).Cite(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println(answer)
    for i, r := range sources.Chunks {
        fmt.Printf("[%d] %s (%.3f)\n", i+1, r.Chunk.ParentID, r.Score)
    }
}
```

## Примеры использования

### Production-ready end-to-end (pgvector)

Ниже — копипастабельный пример с **таймаутами через контекст**, **LRU-кешом эмбеддингов** и **retry/circuit breaker**. Значения таймаутов/ретраев — стартовые ориентиры: калибруйте под свой трафик и latency провайдеров.

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
	baseCtx := context.Background()

	// Рекомендуемые стартовые таймауты (ориентиры):
	// - миграции/создание схемы: 30s
	// - индексация батча документов: 2m
	// - запрос + генерация ответа: 20s
	setupCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", os.Getenv("PG_DSN"))
	if err != nil {
		panic(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	embeddingDim := 1536 // пример; укажите размерность вашей embedding-модели
	pg := draftrag.PGVectorOptions{
		TableName:          "draftrag_chunks",
		EmbeddingDimension: embeddingDim,
		CreateExtension:    false, // в production чаще делается отдельным шагом деплоя
	}

	// Для production обычно применяют миграции отдельной job/init-container.
	if err := draftrag.MigratePGVector(setupCtx, db, draftrag.PGVectorMigrateOptions{PGVectorOptions: pg}); err != nil {
		panic(err)
	}

	store := draftrag.NewPGVectorStoreWithOptions(db, draftrag.PGVectorStoreOptions{
		PGVectorOptions: pg,
		Runtime: draftrag.PGVectorRuntimeOptions{
			SearchTimeout: 2 * time.Second,  // используется только если у ctx нет deadline
			UpsertTimeout: 10 * time.Second, // используется только если у ctx нет deadline
			DeleteTimeout: 5 * time.Second,  // используется только если у ctx нет deadline
		},
	})

	baseEmbedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "text-embedding-3-small",
	})
	cachedEmbedder, err := draftrag.NewCachedEmbedder(baseEmbedder, draftrag.CacheOptions{
		MaxSize: 5000,
		// Опционально: Redis L2 — см. раздел ниже про Redis L2.
	})
	if err != nil {
		panic(err)
	}

	retry := draftrag.RetryOptions{
		MaxRetries:  3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		JitterFactor: 0.2,
		CBThreshold: 5,
		CBTimeout:   30 * time.Second,
	}
	embedder := draftrag.NewRetryEmbedder(cachedEmbedder, retry)

	baseLLM := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-4o-mini",
	})
	llm := draftrag.NewRetryLLMProvider(baseLLM, retry)

	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 5,
		Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
			ChunkSize: 500,
			Overlap:   60,
		}),
	})

	docs := []draftrag.Document{
		{ID: "doc1", Content: "Go использует горутины для конкурентного программирования..."},
		{ID: "doc2", Content: "Каналы в Go обеспечивают синхронизацию между горутинами..."},
	}

	indexCtx, cancel := context.WithTimeout(baseCtx, 2*time.Minute)
	defer cancel()
	if err := pipeline.Index(indexCtx, docs); err != nil {
		panic(err)
	}

	queryCtx, cancel := context.WithTimeout(baseCtx, 20*time.Second)
	defer cancel()
	answer, sources, err := pipeline.Search("Как работают горутины?").TopK(5).Cite(queryCtx)
	if err != nil {
		panic(err)
	}

	fmt.Println(answer)
	for i, r := range sources.Chunks {
		fmt.Printf("[%d] %s (%.3f)\n", i+1, r.Chunk.ParentID, r.Score)
	}
}
```

### Production-ready end-to-end (Qdrant)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
	baseCtx := context.Background()

	// Ориентиры таймаутов: создание коллекции 10s, индексация 2m, запрос/ответ 20s.
	setupCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
	defer cancel()

	embeddingDim := 1536 // пример; укажите размерность вашей embedding-модели
	qopts := draftrag.QdrantOptions{
		URL:        os.Getenv("QDRANT_URL"), // например http://localhost:6333
		Collection: "draftrag_chunks",
		Dimension:  embeddingDim,
		Timeout:    10 * time.Second,
	}

	exists, err := draftrag.CollectionExists(setupCtx, qopts)
	if err != nil {
		panic(err)
	}
	if !exists {
		if err := draftrag.CreateCollection(setupCtx, qopts); err != nil {
			panic(err)
		}
	}

	store, err := draftrag.NewQdrantStore(qopts)
	if err != nil {
		panic(err)
	}

	baseEmbedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "text-embedding-3-small",
	})
	cachedEmbedder, err := draftrag.NewCachedEmbedder(baseEmbedder, draftrag.CacheOptions{MaxSize: 5000})
	if err != nil {
		panic(err)
	}

	retry := draftrag.RetryOptions{
		MaxRetries:  3,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		Multiplier:  2.0,
		JitterFactor: 0.2,
		CBThreshold: 5,
		CBTimeout:   30 * time.Second,
	}
	embedder := draftrag.NewRetryEmbedder(cachedEmbedder, retry)

	baseLLM := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-4o-mini",
	})
	llm := draftrag.NewRetryLLMProvider(baseLLM, retry)

	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 5,
		Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
			ChunkSize: 500,
			Overlap:   60,
		}),
	})

	docs := []draftrag.Document{
		{ID: "doc1", Content: "Go использует горутины для конкурентного программирования..."},
		{ID: "doc2", Content: "Каналы в Go обеспечивают синхронизацию между горутинами..."},
	}

	indexCtx, cancel := context.WithTimeout(baseCtx, 2*time.Minute)
	defer cancel()
	if err := pipeline.Index(indexCtx, docs); err != nil {
		panic(err)
	}

	queryCtx, cancel := context.WithTimeout(baseCtx, 20*time.Second)
	defer cancel()
	answer, _, err := pipeline.Search("Как работают горутины?").TopK(5).Cite(queryCtx)
	if err != nil {
		panic(err)
	}
	fmt.Println(answer)
}
```

### Потоковый ответ с источниками

```go
// StreamSources — потоковый аналог Cite: источники готовы сразу, токены — асинхронно
tokenChan, sources, err := pipeline.
    Search("Как работают горутины?").
    TopK(3).
    StreamSources(ctx)
if err != nil {
    panic(err)
}

// Показываем источники сразу, не дожидаясь завершения генерации
for i, r := range sources.Chunks {
    fmt.Printf("[%d] %s\n", i+1, r.Chunk.ParentID)
}

// Читаем токены по мере генерации
for token := range tokenChan {
    fmt.Print(token)
}
```

### Потоковый ответ с inline-цитатами

```go
tokenChan, sources, citations, err := pipeline.
    Search("Как работают горутины?").
    TopK(3).
    StreamCite(ctx)
if err != nil {
    panic(err)
}

for token := range tokenChan {
    fmt.Print(token) // LLM расставляет [1], [2] в тексте
}

for i, c := range citations {
    fmt.Printf("[%d] %s (score: %.3f)\n", i+1, c.Chunk.Chunk.ParentID, c.Chunk.Score)
}
_ = sources // полный список всех найденных чанков
```

### HyDE (Hypothetical Document Embeddings)

```go
// LLM сгенерирует гипотетический ответ, затем поиск выполняется по нему
answer, err := pipeline.
    Search("Как устроен GC в Go?").
    TopK(3).
    HyDE().
    Answer(ctx)
```

### Multi-query retrieval

```go
// 3 перефразы вопроса → объединение результатов → дедупликация
answer, sources, err := pipeline.
    Search("Что такое горутины?").
    TopK(5).
    MultiQuery(3).
    Cite(ctx)
```

### Гибридный поиск (только pgvector)

```go
cfg := draftrag.DefaultHybridConfig()
cfg.SemanticWeight = 0.7 // 70% semantic, 30% BM25

answer, err := pipeline.
    Search("PostgreSQL full-text search").
    TopK(5).
    Hybrid(cfg).
    Answer(ctx)
```

### Фильтрация по метаданным

```go
// Искать только в документах с определённым тегом
answer, err := pipeline.
    Search("безопасность").
    TopK(5).
    Filter(draftrag.MetadataFilter{
        Fields: map[string]string{"category": "security"},
    }).
    Answer(ctx)
```

### Retry + Circuit Breaker для production

```go
resilientEmbedder := draftrag.NewRetryEmbedder(embedder, draftrag.RetryOptions{
    MaxRetries:  3,
    CBThreshold: 5,             // открыть circuit после 5 ошибок
    CBTimeout:   30 * time.Second,
})
resilientLLM := draftrag.NewRetryLLMProvider(llm, draftrag.RetryOptions{
    MaxRetries:  2,
    CBThreshold: 3,
    CBTimeout:   60 * time.Second,
})

pipeline := draftrag.NewPipeline(store, resilientLLM, resilientEmbedder)
```

### Observability hooks

```go
type myHooks struct{}

func (h *myHooks) StageStart(ctx context.Context, ev draftrag.StageStartEvent) {
    log.Printf("→ %s/%s", ev.Operation, ev.Stage)
}
func (h *myHooks) StageEnd(ctx context.Context, ev draftrag.StageEndEvent) {
    log.Printf("← %s/%s %s err=%v", ev.Operation, ev.Stage, ev.Duration, ev.Err)
}

pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder,
    draftrag.PipelineOptions{
        Hooks: &myHooks{},
    },
)
```

Стадии: `chunking`, `embed`, `search`, `generate`.

#### OpenTelemetry (опционально)

Подключение OTel — это **минимальный код, без форка библиотеки**: вы настраиваете OTel SDK в своём приложении, а в pipeline передаёте hooks-адаптер.

Важно: hooks вызываются **синхронно**, поэтому используйте неблокирующие/батчевые exporters.

```go
import (
    draftragotel "github.com/bzdvdn/draftrag/pkg/draftrag/otel"
)

hooks, err := draftragotel.NewHooks(draftragotel.HooksOptions{})
if err != nil {
    panic(err)
}

pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder,
    draftrag.PipelineOptions{
        Hooks: hooks,
    },
)
```

Контракт v1 (stable naming):
- span attributes: `draftrag.operation`, `draftrag.stage`
- metrics:
  - `draftrag.pipeline.stage.duration_ms` (histogram, labels: `operation`, `stage`)
  - `draftrag.pipeline.stage.errors` (counter, labels: `operation`, `stage`)

### Eval harness

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag/eval"

cases := []eval.Case{
    {
        Question:       "Как работают горутины?",
        ExpectedDocIDs: []string{"doc1", "doc2"},
    },
}

report, err := eval.Run(ctx, pipeline, cases, eval.Options{TopK: 5})
if err != nil {
    panic(err)
}
fmt.Printf("Hit@5: %.2f  MRR: %.2f\n", report.Metrics.HitAtK, report.Metrics.MRR)
```

### Локальный стек (Ollama)

```go
embedder := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
    BaseURL: "http://localhost:11434",
    Model:   "nomic-embed-text",
})
llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
    BaseURL: "http://localhost:11434",
    Model:   "llama3",
})
pipeline := draftrag.NewPipeline(draftrag.NewInMemoryStore(), llm, embedder)
```

### Кэширование эмбеддингов

```go
cached, err := draftrag.NewCachedEmbedder(embedder, draftrag.CacheOptions{
    MaxSize: 1000, // LRU-кэш на 1000 записей
})
if err != nil {
    panic(err)
}
pipeline := draftrag.NewPipeline(store, llm, cached)
```

### Options pattern (публичный API)

Для консистентности публичные конструкторы `pkg/draftrag` используют единый паттерн: `...Options` struct как последний параметр (zero-values = defaults). Если у компонента есть несколько логических групп опций — они объединяются в один options-контейнер с вложенными секциями (например, `Runtime`).

### Структурированное логирование (опционально)

draftRAG — библиотека, поэтому логирование по умолчанию выключено. Для событий деградации (Redis L2, retry/circuit breaker) можно подключить свой структурированный логгер:

#### Redaction и безопасность логов

Библиотека **best-effort** редактирует известные ей секреты (например, `APIKey`/bearer token из options) из сообщений ошибок, которые она формирует, чтобы они не утекали в APM/лог-агрегаторы.

Границы ответственности:
- draftRAG не делает автоматическое распознавание PII в произвольном тексте;
- не логируйте сырые документы/запросы в своём приложении без собственной политики/редакции.

```go
type myLogger struct{}

func (l *myLogger) Log(ctx context.Context, level draftrag.LogLevel, msg string, fields ...draftrag.LogField) {
    // адаптируйте под slog/zap/logrus
}

cached, err := draftrag.NewCachedEmbedder(embedder, draftrag.CacheOptions{
    MaxSize: 1000,
    Logger:  &myLogger{},
})
```

Redis L2 (опционально) включается через адаптер-интерфейс клиента:

```go
cached, err := draftrag.NewCachedEmbedder(embedder, draftrag.CacheOptions{
    MaxSize: 1000,
    Redis: draftrag.RedisCacheOptions{
        Client:    myRedisClient,          // реализует GetBytes/SetBytes
        TTL:       10 * time.Minute,       // 0 → без TTL
        KeyPrefix: "myapp:embedder:",      // "" → draftrag:embedder:
    },
})
```

### Batch-индексация больших корпусов

```go
result, err := pipeline.IndexBatch(ctx, docs, 10) // 10 документов параллельно
if err != nil {
    panic(err)
}
fmt.Printf("ok=%d failed=%d\n", len(result.Successful), len(result.Failed))
for _, fe := range result.Failed {
    fmt.Printf("failed %s: %v\n", fe.DocumentID, fe.Err)
}
```

## Полный список примеров

| Пример | Описание |
|--------|----------|
| [examples/chat](examples/chat/) | Интерактивный RAG-чат, in-memory store, inline citations |
| [examples/index-dir](examples/index-dir/) | Индексация директории с `.txt` файлами |
| [examples/pgvector](examples/pgvector/) | RAG с PostgreSQL+pgvector, docker-compose |
| [examples/qdrant](examples/qdrant/) | RAG с Qdrant, auto-create collection |

Документация:
- [docs/production.md](docs/production.md) — production checklist + runbook
- [docs/getting-started.md](docs/getting-started.md) — начало работы

Дополнительные примеры в тестах:
- [pipeline_answer_test.go](pkg/draftrag/pipeline_answer_test.go) — базовые сценарии
- [answer_stream_test.go](pkg/draftrag/answer_stream_test.go) — streaming
- [search_builder_test.go](pkg/draftrag/search_builder_test.go) — Search Builder, HyDE, Multi-query, фильтры

## Структура пакета

```
pkg/draftrag/          — публичный API (используйте его)
pkg/draftrag/eval/     — eval harness (Hit@K, MRR)
internal/
  domain/              — интерфейсы и модели данных
  application/         — бизнес-логика pipeline
  infrastructure/      — реализации: vectorstore, embedder, llm, chunker, resilience
```

## Лицензия

MIT
