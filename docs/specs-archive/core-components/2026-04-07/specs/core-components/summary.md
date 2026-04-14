---
slug: core-components
generated_at: 2026-04-05
---

## Goal

Разработчики получают набор абстракций для работы с RAG-системами без привязки к конкретным провайдерам.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Интерфейсы определены и документированы | godoc выводит описание методов на русском |
| AC-002 | Domain-модели описывают типичный RAG-сценарий | unit-тест с in-memory store показывает Upsert и Search |
| AC-003 | Контекст поддерживается во всех операциях | unit-тест с отменённым контекстом возвращает context.Canceled |
| AC-004 | In-memory VectorStore проходит базовые тесты | unit-тест BasicSearch проходит успешно |
| AC-005 | Публичный API позволяет скомпоновать pipeline | integration-тест демонстрирует полный цикл Index+Query |

## Out of Scope

- Конкретные реализации провайдеров (pgvector, Qdrant, OpenAI и др.)
- Стриминг ответов от LLM
- Асинхронная индексация через worker queue
- HTTP handlers или middleware
- CLI утилиты
