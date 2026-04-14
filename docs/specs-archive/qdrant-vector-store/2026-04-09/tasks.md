# Qdrant vector store Задачи

## Phase Contract

Inputs: plan qdrant-vector-store и data-model.md.
Outputs: упорядоченные исполнимые задачи с покрытием критериев приемки.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/vectorstore/qdrant.go` | T1.1, T2.1, T2.2, T2.3 |
| `internal/infrastructure/vectorstore/qdrant_test.go` | T1.2, T3.1, T3.2, T3.3 |
| `pkg/draftrag/qdrant.go` | T2.4 |
| `pkg/draftrag/qdrant_test.go` | T3.4 |

## Фаза 1: Основа и скелет

Цель: создать базовую структуру QdrantStore с HTTP клиентом и compile-time проверкой интерфейсов.

- [x] **T1.1** Создать `internal/infrastructure/vectorstore/qdrant.go` со структурой `QdrantStore`, конструктором и HTTP client setup — compile-time проверка `var _ domain.VectorStore = (*QdrantStore)(nil)` и `var _ domain.VectorStoreWithFilters = (*QdrantStore)(nil)` проходит. Touches: `internal/infrastructure/vectorstore/qdrant.go`
  - AC: RQ-001, RQ-002

- [x] **T1.2** Создать `internal/infrastructure/vectorstore/qdrant_test.go` с базовым mock HTTP server для тестирования — `httptest.Server` возвращает ожидаемые JSON ответы Qdrant. Touches: `internal/infrastructure/vectorstore/qdrant_test.go`
  - AC: DEC-001

## Фаза 2: Основная реализация

Цель: реализовать все методы VectorStore и VectorStoreWithFilters, миграции в публичном API.

- [x] **T2.1** Реализовать `Upsert`, `Delete`, `Search` — HTTP запросы к Qdrant points API, маппинг Chunk на point payload (id, content, parent_id, position, metadata.*). Touches: `internal/infrastructure/vectorstore/qdrant.go`
  - AC: AC-001 (Search), AC-004 (Upsert/Delete), DEC-002, DEC-003

- [x] **T2.2** Реализовать `SearchWithFilter` (ParentID) — маппинг `ParentIDFilter` на Qdrant payload filter с `should` для списка ParentIDs. Touches: `internal/infrastructure/vectorstore/qdrant.go`
  - AC: AC-002

- [x] **T2.3** Реализовать `SearchWithMetadataFilter` — маппинг `MetadataFilter.Fields` на Qdrant `must` фильтр с ключами `metadata.k`. Touches: `internal/infrastructure/vectorstore/qdrant.go`
  - AC: AC-003

- [x] **T2.4** Создать `pkg/draftrag/qdrant.go` с фабрикой `NewQdrantStore` и миграциями `CreateCollection`, `DeleteCollection` — HTTP запросы PUT/DELETE /collections/{name}. Touches: `pkg/draftrag/qdrant.go`
  - AC: RQ-003, AC-005

## Фаза 3: Проверка и тестирование

Цель: доказать корректность реализации через unit-тесты с mock server.

- [x] **T3.1** Добавить тест `TestQdrantStore_Search` — mock server возвращает points, тест проверяет корректность RetrievedChunk с score. Touches: `internal/infrastructure/vectorstore/qdrant_test.go`
  - AC: AC-001

- [x] **T3.2** Добавить тест `TestQdrantStore_UpsertDelete` — upsert создаёт point, delete удаляет, поиск после delete не находит. Touches: `internal/infrastructure/vectorstore/qdrant_test.go`
  - AC: AC-004

- [x] **T3.3** Добавить тесты фильтров и ошибок — `TestQdrantStore_SearchWithParentIDFilter`, `TestQdrantStore_SearchWithMetadataFilter`, `TestQdrantStore_APIErrors` (404, 400). Touches: `internal/infrastructure/vectorstore/qdrant_test.go`
  - AC: AC-002, AC-003, AC-006

- [x] **T3.4** Добавить `pkg/draftrag/qdrant_test.go` с тестом `TestQdrantStore_Migrations` — проверяет создание и удаление коллекции через mock server. Touches: `pkg/draftrag/qdrant_test.go`
  - AC: AC-005

## Покрытие критериев приемки

| Критерий | Покрытие задачами |
|----------|-------------------|
| AC-001 Базовый векторный поиск | T2.1, T3.1 |
| AC-002 Фильтрация по ParentID | T2.2, T3.3 |
| AC-003 Фильтрация по метаданным | T2.3, T3.3 |
| AC-004 Upsert и Delete | T2.1, T3.2 |
| AC-005 Создание/удаление коллекции | T2.4, T3.4 |
| AC-006 Обработка ошибок API | T3.3 |

## Покрытие требований

| Требование | Покрытие задачами |
|------------|-------------------|
| RQ-001 VectorStore интерфейс | T1.1 |
| RQ-002 VectorStoreWithFilters интерфейс | T1.1 |
| RQ-003 Factory NewQdrantStore | T2.4 |
| RQ-004 CreateCollection | T2.4 |
| RQ-005 DeleteCollection | T2.4 |
| RQ-006 Upsert с payload | T2.1 |
| RQ-007 Search с RetrievalResult | T2.1, T3.1 |
| RQ-008 SearchWithFilter | T2.2 |
| RQ-009 SearchWithMetadataFilter | T2.3 |
| RQ-010 Маппинг MetadataFilter | T2.3 |
| RQ-011 Context support | T1.1, T2.1, T2.2, T2.3 |

## Заметки

- Декомпозиция ленивая: 8 задач покрывают 6 AC и 11 RQ
- Фаза 1 создаёт скелет без business logic
- Фаза 2 добавляет всё поведение — core и filters
- Фаза 3 только тесты, нет отдельных "cleanup" задач
- Все задачи имеют конкретные `Touches:` для batch-чтения на implement
