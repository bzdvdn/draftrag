---
status: no-change
reason: Ни одна domain-модель не добавляется, не изменяется и не удаляется. Все изменения — internal-рефакторинг (перемещение кода между файлами), экспорт существующих типов через type-aliases, и добавление тестов. Data model остаётся идентичной.
---

# Data Model: hardening-2026q2

## Domain entities

- `Document`, `Chunk`, `Query`, `RetrievalResult`, `RetrievedChunk`, `InlineCitation`, `Embedding`, `MetadataFilter`, `HybridConfig`, `ParentIDFilter`, `IndexBatchResult`, `IndexBatchError` — **без изменений**.

## Domain interfaces  

- `VectorStore`, `LLMProvider`, `Embedder`, `Chunker`, `VectorStoreWithFilters`, `StreamingLLMProvider`, `HybridSearcher`, `Reranker`, `DocumentStore`, `CollectionManager`, `HybridSearcherWithFilters` — **без изменений**.

## Public API in pkg/draftrag/

- Добавляется функция `NewRedisCache` — новый экспорт, additive change.
- Ошибки `ErrEmptyDocument`, `ErrEmptyQuery`, `ErrInvalidTopK`, `ErrFiltersNotSupported`, `ErrStreamingNotSupported`, `ErrDeleteNotSupported` — становятся type-aliases на `domain.Err*` (semantically unchanged).
- `mapValidationErr` — урезается, но не удаляется (non-sentinel ошибки остаются).

## Application layer

- `PipelineConfig` — те же поля, то же поведение. `Pipeline` — те же методы, те же сигнатуры.

## Contracts

- API-контракты `pkg/draftrag` — additive only. Ни один существующий экспорт не меняет сигнатуру.
- Internal-контракты `internal/application` — не экспортируются, пользователь их не видит.
