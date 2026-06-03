# pgvector — RAG с PostgreSQL + pgvector

Интерактивный RAG-чат с pgvector как постоянным векторным хранилищем. Схема БД создаётся автоматически при первом запуске через `MigratePGVector` (идемпотентно).

## Быстрый старт

**1. Запустите PostgreSQL с pgvector:**

```bash
docker compose up -d
```

**2. Запустите пример:**

```bash
cd examples/pgvector && cp .env.example .env && go run .
```

Для mock-режима этого достаточно. Для реального LLM задайте `LLM_PROVIDER=ollama|openai|anthropic` и соответствующие ключи.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|-----------|-------------|---------|
| `LLM_PROVIDER` | `mock` | LLM провайдер (`mock`, `ollama`, `openai`, `anthropic`) |
| `EMBEDDING_DIM` | `1536` | Размерность векторов (должна совпадать с моделью) |
| `PGVECTOR_DSN` | — | **Обязательно.** DSN для подключения к PostgreSQL |
| `TABLE_NAME` | `draftrag_chunks` | Имя таблицы для хранения чанков |

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

## Размерность векторов

Размерность должна соответствовать используемой модели эмбеддингов:

| Модель | `EMBEDDING_DIM` |
|---|---|
| `text-embedding-ada-002` | `1536` |
| `text-embedding-3-small` | `1536` |
| `text-embedding-3-large` | `3072` |
| `nomic-embed-text` (Ollama) | `768` |

Если размерность изменилась после первого запуска — нужно пересоздать таблицу или использовать другое `TABLE_NAME`.

## Миграции

`MigratePGVector` создаёт таблицу и индекс при первом запуске. Повторный запуск безопасен — миграции идемпотентны.

Для production рекомендуется применять SQL-миграции отдельным шагом деплоя:

```bash
psql $PGVECTOR_DSN -f pkg/draftrag/migrations/pgvector/0000_pgvector_extension.sql
psql $PGVECTOR_DSN -f pkg/draftrag/migrations/pgvector/0001_chunks_table.sql
psql $PGVECTOR_DSN -f pkg/draftrag/migrations/pgvector/0002_metadata_and_indexes.sql
```

## Локальный режим (Ollama)

```bash
ollama pull nomic-embed-text
ollama pull llama3.2

PGVECTOR_DSN="postgres://draftrag:draftrag@localhost:5432/draftrag?sslmode=disable" \
EMBEDDING_DIM=768 \
LLM_PROVIDER=ollama \
OLLAMA_HOST=http://localhost:11434 \
go run ./examples/pgvector/
```
