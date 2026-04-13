# Compatibility & Support Policy

Этот документ фиксирует публичный “контракт поддержки” draftRAG: какие версии Go поддерживаются, насколько стабилен публичный API, и какой статус у backend’ов.

Важно:
- Это **best-effort** политика (без SLA/SLO гарантий).
- Контракт относится к публичному API `pkg/draftrag` и публичным docs. Всё в `internal/` может меняться без предупреждения.

## Go support

- Минимальная версия Go: **1.23**.
- Окно поддержки: поддерживаем **N последних minor-версий Go**, где **N = 2**, плюс минимальная (пока она остаётся в этом окне).
  - Пример: при выходе нового minor Go мы планово обновляем поддержку в ближайших релизах библиотеки.
- Повышение минимальной версии Go считается **breaking change**:
  - объявляется заранее в релиз-нотах (и/или CHANGELOG, если он ведётся);
  - вступает в силу в **major** релизе.

## SemVer & Deprecation (публичный API)

- Семантическое версионирование (SemVer) применяется к публичному API `pkg/draftrag` и документированному поведению.
- Breaking changes:
  - допускаются только в **major** релизах;
  - должны сопровождаться заметками о миграции в релиз-нотах.
- Deprecation (устаревшие API):
  - помечаем в godoc с префиксом `Deprecated:` и указываем замену;
  - **держим deprecated API минимум 2 minor релиза или 6 месяцев (что дольше)**;
  - удаляем только в следующем **major** релизе.

## Статусы backend’ов

Определения статусов:
- **stable** — поддерживается и считается пригодным для production при корректной настройке (таймауты, ретраи, наблюдаемость).
- **experimental** — работает, но контракт/поведение может меняться быстрее; используйте с дополнительным вниманием к релиз-нотам.

### Vector stores

| Backend | Статус | Notes |
|---|---|---|
| In-memory | stable | Для прототипов/тестов; **без** постоянного хранения |
| PostgreSQL + pgvector | stable | Production-ready; hybrid search (BM25+semantic), SQL-фильтры, миграции |
| Qdrant | stable | Production-ready; payload filters, управление коллекциями |
| ChromaDB | stable | Требует заранее созданной коллекции; фильтры доступны через API |
| Weaviate | experimental | См. [docs/weaviate.md](weaviate.md); используйте с оглядкой на релиз-ноты |

### Embedders

| Backend | Статус | Notes |
|---|---|---|
| OpenAI-compatible embeddings | stable | Любой совместимый `POST /v1/embeddings` |
| Ollama embeddings | stable | Локальные модели через Ollama |
| CachedEmbedder (LRU + опц. Redis L2) | stable | Кэш поверх любого embedder’а |

### LLM providers

| Backend | Статус | Notes |
|---|---|---|
| OpenAI-compatible (Responses API) | stable | Поддерживает streaming через `StreamingLLMProvider` |
| Anthropic Claude | stable | Поддерживает streaming через `StreamingLLMProvider` |
| Ollama LLM | stable | **Streaming не поддерживается** через draftRAG |

## Матрица возможностей (best-effort по docs/README)

Легенда: `✓` — поддерживается, `—` — не поддерживается, `n/a` — не применимо.

### Vector stores

| Feature | In-memory | pgvector | Qdrant | ChromaDB | Weaviate |
|---|---:|---:|---:|---:|---:|
| Постоянное хранение | — | ✓ | ✓ | ✓ | ✓ |
| Metadata filters | ✓ | ✓ | ✓ | ✓ | ✓ |
| Hybrid search (BM25) | ✓ | ✓ | — | — | — |
| SQL-миграции | n/a | ✓ | n/a | n/a | n/a |
| Управление коллекцией | n/a | n/a | ✓ | — | ✓ |

### LLM providers

| Feature | OpenAI-compatible | Anthropic | Ollama |
|---|---:|---:|---:|
| Generate (non-stream) | ✓ | ✓ | ✓ |
| Streaming (`AnswerStream`, `.Stream*`) | ✓ | ✓ | — |

### Cross-cutting (поперечные возможности)

| Feature | Поддержка |
|---|---|
| Таймауты/отмена | `context.Context` везде; у некоторых backend’ов есть опциональные timeouts в options |
| Retry + Circuit Breaker | обёртки `RetryEmbedder` и `RetryLLMProvider` |
| Кеширование эмбеддингов | `CachedEmbedder` (L1 LRU) + опционально Redis L2 |
| Observability hooks | hooks по стадиям пайплайна (chunking/embed/search/generate) |
| OpenTelemetry | публичные hooks в `pkg/draftrag/otel` |

## Update policy

- Этот документ обновляется вместе с релизами draftRAG.
- Любое изменение статуса backend’а, окна поддержки Go или правил депрекации отражается в релиз-нотах.
