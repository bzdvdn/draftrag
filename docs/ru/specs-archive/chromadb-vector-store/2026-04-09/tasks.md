# ChromaDB vector store — Задачи

## Phase Contract

Inputs: plan `chromadb-vector-store`, spec summary, data-model.
Outputs: исполнимые задачи с покрытием AC.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/vectorstore/chromadb.go` | T1.1, T2.1, T2.2, T2.3, T2.4 |
| `internal/infrastructure/vectorstore/chromadb_test.go` | T3.1, T3.2, T3.3 |
| `internal/infrastructure/vectorstore/qdrant.go` | Reference only |

## Фаза 1: Основа

Цель: создать структуру ChromaStore и HTTP клиент, подготовить compile-time проверки.

- [x] T1.1 Создать `ChromaStore` структуру с полями (baseURL, collection, dimension, client), `ChromaRuntimeOptions`, `NewChromaStore()` с defaults — AC-001, RQ-001, RQ-002, DEC-001. Touches: `internal/infrastructure/vectorstore/chromadb.go`

## Фаза 2: Основная реализация

Цель: реализовать все операции VectorStore и VectorStoreWithFilters.

- [x] T2.1 Реализовать `Upsert()` с валидацией размерности, маппингом metadata, HTTP POST `/api/v1/collections/{name}/upsert` — AC-001, AC-005, RQ-003, RQ-008, DEC-002. Touches: `internal/infrastructure/vectorstore/chromadb.go`

- [x] T2.2 Реализовать `Search()` с HTTP POST `/api/v1/collections/{name}/query`, парсингом distances → score, маппингом в `RetrievalResult` — AC-002, RQ-005. Touches: `internal/infrastructure/vectorstore/chromadb.go`

- [x] T2.3 Реализовать `Delete()` с HTTP POST `/api/v1/collections/{name}/delete` и `SearchWithFilter()` (ParentID) — AC-004, RQ-004. Touches: `internal/infrastructure/vectorstore/chromadb.go`

- [x] T2.4 Реализовать `SearchWithMetadataFilter()` с where-фильтром, автосоздание коллекции при отсутствии, compile-time проверки интерфейсов — AC-003, AC-007, RQ-006, RQ-007, DEC-003. Touches: `internal/infrastructure/vectorstore/chromadb.go`

## Фаза 3: Проверка

Цель: доказать работоспособность через unit-тесты с mock HTTP server.

- [x] T3.1 Добавить тесты `TestChromaStore_Upsert`, `TestChromaStore_Search`, `TestChromaStore_SearchWithMetadataFilter` с httptest.Server — AC-001, AC-002, AC-003. Touches: `internal/infrastructure/vectorstore/chromadb_test.go`

- [x] T3.2 Добавить тесты `TestChromaStore_Delete`, `TestChromaStore_DimensionMismatch`, `TestChromaStore_AutocreateCollection` — AC-004, AC-005, AC-007. Touches: `internal/infrastructure/vectorstore/chromadb_test.go`

- [x] T3.3 Добавить тест `TestChromaStore_ContextCancellation` и verify покрытие ≥60%, `go vet` без ошибок — AC-006, SC-001. Touches: `internal/infrastructure/vectorstore/chromadb_test.go`

## Покрытие критериев приемки

| AC | Покрывается задачами |
|----|---------------------|
| AC-001 Успешный upsert | T1.1, T2.1, T3.1 |
| AC-002 Поиск по эмбеддингу | T2.2, T3.1 |
| AC-003 Фильтрация по метаданным | T2.4, T3.1 |
| AC-004 Удаление чанка | T2.3, T3.2 |
| AC-005 Валидация размерности | T2.1, T3.2 |
| AC-006 Context cancellation | T3.3 |
| AC-007 Автосоздание коллекции | T2.4, T3.2 |

## Заметки

- Задачи следуют порядку из плана: структура → upsert/search → delete/metadata → тесты
- Каждая задача имеет конкретный файл в `Touches:` для batch-чтения implement-агентом
- AC-006 (context cancellation) покрывается отдельной тестовой задачей T3.3
