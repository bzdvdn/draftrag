# Milvus — RAG с Milvus

Интерактивный RAG-чат с Milvus как векторным хранилищем. 

**Внимание:** Milvus — самый ресурсоёмкий бэкенд. Требуется ~2 GB RAM для работы. При первом запуске может потребоваться время на инициализацию (start_period: 30s).

## Быстрый старт

**1. Запустите Milvus (etcd + minio + milvus standalone):**

```bash
docker compose up -d
```

**2. Запустите пример:**

```bash
cd examples/milvus && cp .env.example .env && go run .
```

Для mock-режима этого достаточно. Для реального LLM задайте `LLM_PROVIDER=ollama|openai|anthropic` и соответствующие ключи.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `LLM_PROVIDER` | `mock` | LLM провайдер (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Размерность векторов |
| `MILVUS_ADDR` | `localhost:19121` | Адрес Milvus REST API |
| `COLLECTION_NAME` | `draftrag_chunks` | Имя коллекции |

## Примечание

MilvusStore — внутренний API (`internal/infrastructure/vectorstore`), статус: "API в разработке". 
Публичный API будет добавлен в одном из следующих релизов.
