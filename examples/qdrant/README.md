# qdrant — RAG с Qdrant

Интерактивный RAG-чат с Qdrant как векторным хранилищем. При первом запуске автоматически создаёт коллекцию, если она ещё не существует.

## Быстрый старт

**1. Запустите Qdrant:**

```bash
docker run -d -p 6333:6333 --name qdrant qdrant/qdrant
```

**2. Запустите пример:**

```bash
EMBEDDER_API_KEY=sk-... \
LLM_API_KEY=sk-... \
go run ./examples/qdrant/
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `QDRANT_URL` | `http://localhost:6333` | URL Qdrant сервера |
| `QDRANT_COLLECTION` | `draftrag_example` | Имя коллекции |
| `EMBEDDING_DIM` | `1536` | Размерность векторов (должна совпадать с моделью) |
| `EMBEDDER_API_KEY` | — | **Обязательно.** Ключ API для embedder |
| `EMBEDDER_BASE_URL` | `https://api.openai.com` | Базовый URL embedder API |
| `EMBEDDER_MODEL` | `text-embedding-ada-002` | Модель эмбеддингов |
| `LLM_API_KEY` | — | **Обязательно.** Ключ API для LLM |
| `LLM_BASE_URL` | `https://api.openai.com` | Базовый URL LLM API |
| `LLM_MODEL` | `gpt-4o-mini` | Языковая модель |

## Управление коллекцией

Пример создаёт коллекцию автоматически. При необходимости можно управлять коллекциями вручную через публичный API:

```go
opts := draftrag.QdrantOptions{
    URL:        "http://localhost:6333",
    Collection: "my_collection",
    Dimension:  1536,
}

// Проверить существование
exists, err := draftrag.CollectionExists(ctx, opts)

// Создать
err = draftrag.CreateCollection(ctx, opts)

// Удалить
err = draftrag.DeleteCollection(ctx, opts)
```

## Размерность векторов

Размерность задаётся при создании коллекции и не может быть изменена. Если нужно изменить размерность — удалите коллекцию и создайте заново:

```bash
# Удалить коллекцию через API Qdrant
curl -X DELETE http://localhost:6333/collections/draftrag_example
```

| Модель | `EMBEDDING_DIM` |
|---|---|
| `text-embedding-ada-002` | `1536` |
| `text-embedding-3-small` | `1536` |
| `text-embedding-3-large` | `3072` |
| `nomic-embed-text` (Ollama) | `768` |

## Локальный режим (Ollama)

```bash
ollama pull nomic-embed-text
ollama pull llama3.2

EMBEDDING_DIM=768 \
EMBEDDER_BASE_URL=http://localhost:11434 \
EMBEDDER_API_KEY=ollama \
EMBEDDER_MODEL=nomic-embed-text \
LLM_BASE_URL=http://localhost:11434 \
LLM_API_KEY=ollama \
LLM_MODEL=llama3.2 \
go run ./examples/qdrant/
```

## Qdrant Cloud

```bash
QDRANT_URL=https://xyz.cloud.qdrant.io:6333 \
EMBEDDER_API_KEY=sk-... \
LLM_API_KEY=sk-... \
go run ./examples/qdrant/
```

Для аутентификации в Qdrant Cloud передайте API-ключ через заголовок — это потребует небольшой модификации примера или использования `draftrag.QdrantOptions` с расширением.
