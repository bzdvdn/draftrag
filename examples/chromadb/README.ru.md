# ChromaDB — RAG с ChromaDB

Интерактивный RAG-чат с ChromaDB как векторным хранилищем. Коллекция создаётся автоматически при первом запуске.

## Быстрый старт

**1. Запустите ChromaDB:**

```bash
docker compose up -d
```

**2. Запустите пример:**

```bash
cd examples/chromadb && cp .env.example .env && go run .
```

Для mock-режима этого достаточно. Для реального LLM задайте `LLM_PROVIDER=ollama|openai|anthropic` и соответствующие ключи.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `LLM_PROVIDER` | `mock` | LLM провайдер (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Размерность векторов |
| `CHROMADB_URL` | `http://localhost:8000` | URL ChromaDB сервера |
| `COLLECTION_NAME` | `draftrag_chunks` | Имя коллекции |

## Tutorial

Подробное руководство по работе с ChromaDB и метаданными — [tutorial 04: Metadata Filter](../docs/tutorials/ru/04-metadata-filter.md).
