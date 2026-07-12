# Qdrant — RAG с Qdrant

Интерактивный RAG-чат с Qdrant как векторным хранилищем. Создаёт коллекцию автоматически при первом запуске.

## Быстрый старт

**1. Запустите Qdrant:**

```bash
docker compose up -d
```

**2. Запустите пример:**

```bash
cd examples/qdrant && cp .env.example .env && go run .
```

Для mock-режима этого достаточно. Для реального LLM задайте `LLM_PROVIDER=ollama|openai|anthropic` и соответствующие ключи.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `LLM_PROVIDER` | `mock` | LLM провайдер (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Размерность векторов |
| `QDRANT_URL` | `http://localhost:6333` | URL Qdrant сервера |
| `COLLECTION_NAME` | `draftrag_chunks` | Имя коллекции |

Для `LLM_PROVIDER=ollama`:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `OLLAMA_HOST` | `http://localhost:11434` | URL Ollama |
| `OLLAMA_EMBED_MODEL` | `nomic-embed-text` | Модель эмбеддингов |
| `OLLAMA_LLM_MODEL` | `llama3.2` | LLM модель |

Для `LLM_PROVIDER=openai`:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `OPENAI_API_KEY` | — | **Обязательно.** API ключ |
| `OPENAI_BASE_URL` | `https://api.openai.com` | Базовый URL |
| `OPENAI_EMBED_MODEL` | `text-embedding-3-small` | Модель эмбеддингов |
| `OPENAI_LLM_MODEL` | `gpt-4o-mini` | LLM модель |

Для `LLM_PROVIDER=anthropic`:

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `ANTHROPIC_API_KEY` | — | **Обязательно.** API ключ |
| `ANTHROPIC_LLM_MODEL` | `claude-3-5-sonnet-latest` | LLM модель |
