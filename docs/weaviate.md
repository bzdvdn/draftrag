# Weaviate

Этот документ описывает использование Weaviate в draftRAG через **публичный API** `pkg/draftrag`.

Важно:
- Это **best-effort** документация (без SLA/SLO гарантий).
- В production подготовку коллекции (schema/DDL) обычно делают **отдельным шагом деплоя** (deploy job / init container), а не при старте сервиса.

## Быстрый старт

Ниже — минимальный пример: подготовка коллекции (идемпотентно) → создание store → индексация → retrieval.

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

	// Общий budget на подготовку схемы/коллекции (в деплое обычно больше, чем на запросы).
	setupCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	weaviate := draftrag.WeaviateOptions{
		Host:       "localhost:8080",
		Collection: "MyChunks",
		APIKey:     os.Getenv("WEAVIATE_API_KEY"), // опционально
		Timeout:    10 * time.Second,             // HTTP таймаут для schema операций
	}

	// Обычно это выполняется deploy job/init контейнером.
	exists, err := draftrag.WeaviateCollectionExists(setupCtx, weaviate)
	if err != nil {
		panic(err)
	}
	if !exists {
		if err := draftrag.CreateWeaviateCollection(setupCtx, weaviate); err != nil {
			panic(err)
		}
	}

	store, err := draftrag.NewWeaviateStore(weaviate)
	if err != nil {
		// Ошибки конфигурации сопоставимы через errors.Is(err, draftrag.ErrInvalidVectorStoreConfig)
		panic(err)
	}

	embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "text-embedding-3-small",
		Timeout: 10 * time.Second,
	})
	llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
		BaseURL: "https://api.openai.com",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-4o-mini",
		Timeout: 20 * time.Second,
	})

	pipeline := draftrag.NewPipeline(store, llm, embedder)

	indexCtx, cancel := context.WithTimeout(baseCtx, 2*time.Minute)
	defer cancel()
	if err := pipeline.Index(indexCtx, []draftrag.Document{
		{ID: "doc-1", Content: "Go поддерживает конкурентность через горутины и каналы."},
		{ID: "doc-2", Content: "Контекст в Go позволяет отменять операции и задавать дедлайны."},
	}); err != nil {
		panic(err)
	}

	queryCtx, cancel := context.WithTimeout(baseCtx, 20*time.Second)
	defer cancel()

	result, err := pipeline.Search("Как в Go отменять долгие операции?").TopK(5).Retrieve(queryCtx)
	if err != nil {
		panic(err)
	}
	for i, c := range result.Chunks {
		fmt.Printf("[%d] %s (%.3f)\n", i+1, c.Chunk.ParentID, c.Score)
	}
}
```

## Управление коллекцией (schema)

В Weaviate draftRAG использует коллекцию (class) для хранения чанков. Управление коллекцией доступно через публичные функции:

- `WeaviateCollectionExists(ctx, opts)` → `bool`
- `CreateWeaviateCollection(ctx, opts)` → `error` (идемпотентно: “уже существует” не считается ошибкой)
- `DeleteWeaviateCollection(ctx, opts)` → `error` (идемпотентно: 404 не ошибка)

Рекомендация для production:
- schema/создание коллекции выполняйте **отдельно от runtime** (deploy job/init);
- держите отдельные таймауты: на schema шаги обычно больше, чем на запросы retrieval.

## Возможности и ограничения

Поддерживается:
- базовый retrieval (near-vector поиск) через pipeline;
- фильтрация по `ParentIDs(...)` (когда вы хотите искать только по группе документов);
- фильтры по метаданным через `.Filter(...)` (если вы добавляете метаданные при индексации).

Ограничения:
- **Hybrid search (BM25)** не поддерживается для Weaviate в draftRAG (в отличие от pgvector).

## Типовые ошибки

### 1) 404 / collection missing

**Symptoms**
- ошибки Weaviate о том, что class/collection не найден.

**Checks**
- коллекция создана до старта сервиса (`WeaviateCollectionExists/CreateWeaviateCollection`);
- `WeaviateOptions.Collection` совпадает с фактическим именем class;
- `Host` указывает на правильный инстанс.

### 2) 401/403 / auth

**Symptoms**
- ошибки авторизации в Weaviate Cloud/защищённом инстансе.

**Checks**
- `WeaviateOptions.APIKey` корректный и передаётся (Bearer token);
- используете `https` (если это требуется инстансом) через `WeaviateOptions.Scheme`.

### 3) `context deadline exceeded` / timeouts

**Symptoms**
- таймауты при schema-операциях или retrieval.

**Checks**
- у schema шагов (`CreateWeaviateCollection`) отдельный `context.WithTimeout` и `WeaviateOptions.Timeout`;
- у retrieval пути отдельный `context.WithTimeout` и разумный budget на embed/search/generate.

## Ссылки

- Политика совместимости и поддержки: `docs/compatibility.md`
- Обзор хранилищ: `docs/vector-stores.md`

