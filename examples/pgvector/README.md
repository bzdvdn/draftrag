# pgvector — RAG с PostgreSQL + pgvector

Интерактивный RAG-чат с pgvector как постоянным векторным хранилищем. Схема БД создаётся автоматически при первом запуске через `MigratePGVector` (идемпотентно).

## Быстрый старт

**1. Запустите PostgreSQL с pgvector:**

```bash
docker compose up -d
```

**2. Запустите пример:**

```bash
PGVECTOR_DSN="postgres://draftrag:draftrag@localhost:5432/draftrag?sslmode=disable" \
EMBEDDER_API_KEY=sk-... \
LLM_API_KEY=sk-... \
go run ./examples/pgvector/
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `PGVECTOR_DSN` | — | **Обязательно.** DSN для подключения к PostgreSQL |
| `TABLE_NAME` | `draftrag_example` | Имя таблицы для хранения чанков |
| `EMBEDDING_DIM` | `1536` | Размерность векторов (должна совпадать с моделью) |
| `EMBEDDER_API_KEY` | — | **Обязательно.** Ключ API для embedder |
| `EMBEDDER_BASE_URL` | `https://api.openai.com` | Базовый URL embedder API |
| `EMBEDDER_MODEL` | `text-embedding-ada-002` | Модель эмбеддингов |
| `LLM_API_KEY` | — | **Обязательно.** Ключ API для LLM |
| `LLM_BASE_URL` | `https://api.openai.com` | Базовый URL LLM API |
| `LLM_MODEL` | `gpt-4o-mini` | Языковая модель |

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
EMBEDDER_BASE_URL=http://localhost:11434 \
EMBEDDER_API_KEY=ollama \
EMBEDDER_MODEL=nomic-embed-text \
LLM_BASE_URL=http://localhost:11434 \
LLM_API_KEY=ollama \
LLM_MODEL=llama3.2 \
go run ./examples/pgvector/
```
