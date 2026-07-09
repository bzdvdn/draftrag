---
slug: qdrant-vector-store
archive_date: 2026-04-09
status: completed
generated_at: 2026-04-09T00:46:00+03:00
---

# Qdrant vector store — Archive Summary

## Goal

Реализация бэкенда Qdrant для интерфейсов `VectorStore` и `VectorStoreWithFilters` с поддержкой payload-фильтров.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|----|---------|--------------|
| AC-001 | Базовый векторный поиск | Тест проходит, возвращает RetrievedChunk |
| AC-002 | Фильтрация по ParentID | Тест проходит, только matching чанки |
| AC-003 | Фильтрация по метаданным | Тест проходит, все key=value совпадают |
| AC-004 | Upsert и Delete | Тест проходит, чанк удаляется |
| AC-005 | Создание/удаление коллекции | Тест проходит, коллекция видна в Qdrant |
| AC-006 | Обработка ошибок API | Тесты проходят, ошибки содержат HTTP status |

## Out of Scope

- gRPC клиент для Qdrant
- Гибридный поиск (BM25)
- Аутентификация API ключами
- Управление кластером (шардирование, репликация)
- Снапшоты, бэкапы, квантизация, sparse vectors

## Implementation

- **Файлы реализации**: `internal/infrastructure/vectorstore/qdrant.go`, `pkg/draftrag/qdrant.go`
- **Тесты**: 13 unit-тестов с mock HTTP server
- **Статус**: Все 6 AC выполнены, код проходит `go vet` и `go build`
