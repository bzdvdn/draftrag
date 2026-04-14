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
  - **Причина:** Weaviate не имеет нативной реализации BM25. Реализация через external index избыточна и не соответствует философии draftRAG (минимализм > расширяемость).
  - **Рекомендация:** Используйте pgvector или Qdrant для hybrid search, если требуется BM25+semantic.

## Production Checklist

### Перед деплоем

- [ ] **Schema setup:** Коллекция создана через `CreateWeaviateCollection` в отдельном deploy job или init container
- [ ] **Timeouts:** Установлены разумные таймауты для schema операций (рекомендуется 30s) и retrieval (рекомендуется 10s)
- [ ] **Auth:** API key настроен корректно для Weaviate Cloud или self-hosted инстанса
- [ ] **Monitoring:** Настроены метрики для latency, error rates, connection pool
- [ ] **Backup:** Настроены бэкапы Weaviate (если используете self-hosted)

### Runtime

- [ ] **Context cancellation:** Все операции используют `context.Context` для отмены и дедлайнов
- [ ] **Retry logic:** При необходимости добавьте `RetryEmbedder`/`RetryLLMProvider` для устойчивости к временным сбоям
- [ ] **Observability:** Используйте hooks из `pkg/draftrag/otel` для трассировки операций
- [ ] **Resource limits:** Установлены разумные лимиты на CPU/memory для Weaviate контейнера

### После деплоя

- [ ] **Smoke test:** Проверить базовый retrieval через pipeline
- [ ] **Error handling:** Проверить, что 401/403/404/500 errors обрабатываются корректно
- [ ] **Performance:** Проверить latency для retrieval (должен быть < 1s для типичных запросов)
- [ ] **Alerts:** Настроены alerts для high error rate (> 5%) или high latency (> 5s)

## Performance Guidance

### Batch size для индексации

- **Рекомендация:** Индексируйте документы пакетами по 10–100 документов за операцию
- **Почему:** Слишком большие пакеты (> 100) могут вызвать timeout или memory pressure
- **Почему:** Слишком маленькие пакеты (< 10) увеличивают overhead HTTP запросов

### Timeouts

| Операция | Рекомендуемый timeout | Примечание |
|----------|----------------------|------------|
| Schema (CreateWeaviateCollection) | 30s | Schema операции медленные, дайте больше времени |
| Retrieval (pipeline.Search) | 10s | Включает embedding + search + generate |
| WeaviateCollectionExists | 5s | Быстрая проверка |
| DeleteWeaviateCollection | 10s | Может быть медленным при большом объёме данных |

### Индексирование и performance tuning

- **Векторная размерность:** Используйте размерность, соответствующую вашей embedding модели (например, 768 для `text-embedding-3-small`)
- **Кэширование эмбеддингов:** Используйте `CachedEmbedder` для снижения нагрузки на embedding provider
- **Фильтрация:** Фильтры по метаданным могут замедлять retrieval. Используйте их осознанно.
- **Connection pooling:** Weaviate HTTP client не поддерживает connection pooling в draftRAG — учитывайте это при высокой нагрузке

### Мониторинг

- **Latency:** Целевой latency для retrieval < 1s (p95)
- **Error rate:** Error rate < 1% в steady state
- **Throughput:** Ориентир: 10–100 QPS в зависимости от размера коллекции и Weaviate конфигурации

## Migration Guide

### Breaking changes

**Нет breaking changes в текущей версии.**

Функции уже используют консистентные имена с префиксом `Weaviate*`:
- `WeaviateCollectionExists`
- `CreateWeaviateCollection`
- `DeleteWeaviateCollection`

Эти имена обеспечивают уникальность в пакете `draftrag` (Qdrant использует без префикса).

### Если вы используете старую версию

Если вы использовали draftRAG до v0.x и имена функций отличались, используйте следующую таблицу для миграции:

| Старое имя (если было) | Новое имя |
|----------------------|-----------|
| `CollectionExists` | `WeaviateCollectionExists` |
| `CreateCollection` | `CreateWeaviateCollection` |
| `DeleteCollection` | `DeleteWeaviateCollection` |

**Примечание:** Breaking changes допустимы до v1.0. После релиза v1.0 мы будем следовать SemVer.

### Пример миграции

```go
// Было (если использовалось)
exists, err := draftrag.CollectionExists(ctx, opts)

// Стало
exists, err := draftrag.WeaviateCollectionExists(ctx, opts)
```

## Troubleshooting Guide

### Common Issues

#### 1) High latency (> 5s)

**Possible causes:**
- Timeout слишком большой в `WeaviateOptions`
- Weaviate перегружен (высокая нагрузка)
- Сетевые проблемы между сервисом и Weaviate

**Debugging steps:**
1. Проверьте `WeaviateOptions.Timeout` — уменьшите до разумного значения (10s для retrieval)
2. Проверьте latency до Weaviate через `curl` или `ping`
3. Проверьте метрики Weaviate (CPU, memory, QPS)
4. Уменьшите batch size для индексации

#### 2) Intermittent 401/403 errors

**Possible causes:**
- API key истёк или недействителен
- API key не настроен в `WeaviateOptions`
- Weaviate Cloud изменил политику auth

**Debugging steps:**
1. Проверьте, что `WeaviateOptions.APIKey` установлен корректно
2. Проверьте валидность API key в Weaviate Cloud dashboard
3. Проверьте, что используете `https` scheme для Weaviate Cloud
4. Проверьте logs draftRAG — API key redacted (проверьте, что не leak)

#### 3) Connection refused / network errors

**Possible causes:**
- Weaviate недоступен (контейнер не запущен)
- Неверный `WeaviateOptions.Host`
- Firewall блокирует подключение
- Port mismatch (Weaviate на 8080, вы подключаетесь к другому)

**Debugging steps:**
1. Проверьте, что Weaviate запущен: `curl http://localhost:8080/v1/.well-known/ready`
2. Проверьте `WeaviateOptions.Host` — должен быть `host:port` (без scheme)
3. Проверьте firewall rules
4. Проверьте logs Weaviate контейнера

#### 4) 422 error при создании коллекции

**Possible causes:**
- Коллекция уже существует (нормально, идемпотентно)
- Schema конфликт (коллекция с другим schema уже существует)

**Debugging steps:**
1. Проверьте существование коллекции через `WeaviateCollectionExists`
2. Если коллекция существует с другим schema — удалите через `DeleteWeaviateCollection` и пересоздайте
3. Проверьте logs Weaviate для деталей ошибки

#### 5) Empty results from retrieval

**Possible causes:**
- Коллекция пуста (нет индексированных документов)
- Фильтры слишком строгие
- Embedding dimension mismatch

**Debugging steps:**
1. Проверьте, что документы индексированы через pipeline.Index
2. Уберите фильтры и попробуйте retrieval без них
3. Проверьте, что embedding dimension в `WeaviateOptions.Dimension` соответствует модели

### Debugging Tips

#### Включение verbose logging

Используйте hooks для логирования операций:

```go
import "github.com/bzdvdn/draftrag/pkg/draftrag/otel"

// Добавьте hooks для наблюдаемости
pipeline.WithHooks(
    draftragotel.NewEmbeddingHook(),
    draftragotel.NewSearchHook(),
    draftragotel.NewGenerationHook(),
)
```

#### Тестирование локально

Используйте Docker Compose для локального Weaviate:

```bash
docker run -d -p 8080:8080 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  semitechnologies/weaviate:latest
```

#### Проверка schema

Проверьте schema через Weaviate API:

```bash
curl http://localhost:8080/v1/schema
```

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

