# Weaviate — RAG с Weaviate

Интерактивный RAG-чат с Weaviate как векторным хранилищем. Коллекция создаётся автоматически при первом запуске.

## Быстрый старт

**1. Запустите Weaviate:**

```bash
docker compose up -d
```

**2. Запустите пример:**

```bash
cd examples/weaviate && cp .env.example .env && go run .
```

Для mock-режима этого достаточно. Для реального LLM задайте `LLM_PROVIDER=ollama|openai|anthropic` и соответствующие ключи.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `LLM_PROVIDER` | `mock` | LLM провайдер (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Размерность векторов |
| `WEAVIATE_URL` | `http://localhost:8080` | URL Weaviate сервера |
| `COLLECTION_NAME` | `DraftragChunk` | Имя класса Weaviate |

## Tutorial

Подробное руководство по гибридному поиску — [tutorial 03: Hybrid Search](../docs/tutorials/ru/03-hybrid-search.md).
