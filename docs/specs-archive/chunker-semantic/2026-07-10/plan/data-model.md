---
status: no-change
---

# Data Model: chunker-semantic

## Domain Types

- `domain.Chunker` — не меняется
- `domain.Embedder` — не меняется
- `domain.Chunk` — не меняется
- `domain.Document` — не меняется

## New Package-Level Types (не domain)

- `SemanticChunkerOptions` в `pkg/draftrag` — публичная конфигурация
- `SemanticChunkerConfig` в `pkg/draftrag/config.go` — YAML-представление
- `semanticChunker` (internal) — реализация `domain.Chunker`

## Contracts

- Публичный API не расширяется — `NewSemanticChunker` как новая функция, `PipelineOptions.Chunker` уже существует
- Изменений совместимости нет — только добавление
