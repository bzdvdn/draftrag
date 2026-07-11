---
status: no-change
reason: Middleware chain добавляет новые типы (Middleware interface, StageData struct) в доменный слой, но не изменяет существующие сущности (Document, Chunk, RetrievalResult, HookStage, Hooks, PIIDetector). API-контракты VectorStore, LLMProvider, Embedder, Chunker остаются без изменений. Middleware — additive change.
---

# Data Model: middleware-chain

## Новые типы

| Тип | Слой | Описание |
|-----|------|----------|
| `Middleware` | domain | `type Middleware func(next Handler) Handler` |
| `Handler` | domain | `type Handler func(ctx context.Context, data StageData) (StageData, error)` |
| `StageData` | domain | Единая структура с полями для всех стадий (Stage, Operation, Query, Document, Chunks, Answer, Embedding) |

## Неизменяемые типы

- `domain.Document`, `domain.Chunk`, `domain.RetrievalResult` — без изменений.
- `domain.Hooks`, `domain.PIIDetector` — без изменений.
- `domain.VectorStore`, `domain.LLMProvider`, `domain.Embedder`, `domain.Chunker` — без изменений.
- `domain.HookStage` — без изменений (используется в StageData.Stage).
