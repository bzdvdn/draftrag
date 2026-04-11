# Tasks: chromadb-collection-management

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/interfaces.go` | T1.1 |
| `internal/infrastructure/vectorstore/chromadb.go` | T2.1, T2.2, T2.3 |
| `internal/infrastructure/vectorstore/chromadb_collection_test.go` | T3.1 |

## Фаза 1: Domain interface

Цель: объявить `CollectionManager` в domain-слое, чтобы unblock compile-time assertion и типизированные вызовы.

- [x] T1.1 Добавить интерфейс `CollectionManager` в domain — методы `CreateCollection`, `DeleteCollection`, `CollectionExists` с `context.Context` (AC-006, DEC-001)
  Touches: `internal/domain/interfaces.go`

## Фаза 2: Реализация ChromaStore

Цель: реализовать три публичных метода, поглотить приватный `createCollection`, добавить compile-time assertion.

- [x] T2.1 Заменить `createCollection` публичным `CreateCollection` — `POST /api/v1/collections` с `get_or_create: true`, обновить вызов в `SearchWithMetadataFilter` (AC-001, DEC-002)
  Touches: `internal/infrastructure/vectorstore/chromadb.go`

- [x] T2.2 Добавить `DeleteCollection` — `DELETE /api/v1/collections/{name}`, возвращает `nil` при 200/204/404, ошибку с кодом при других статусах (AC-002, AC-003, DEC-001)
  Touches: `internal/infrastructure/vectorstore/chromadb.go`

- [x] T2.3 Добавить `CollectionExists` — `GET /api/v1/collections/{name}`, возвращает `(true, nil)` при 200, `(false, nil)` при 404, `(false, error)` при прочих ошибках; добавить `var _ domain.CollectionManager = (*ChromaStore)(nil)` (AC-004, AC-005, AC-006, DEC-003)
  Touches: `internal/infrastructure/vectorstore/chromadb.go`

## Фаза 3: Тесты

Цель: покрыть все AC через `httptest.NewServer`; существующие тесты ChromaStore не должны ломаться.

- [x] T3.1 Написать unit-тесты для `CreateCollection`, `DeleteCollection`, `CollectionExists` — happy path и error cases через mock HTTP-сервер (AC-001–AC-005)
  Touches: `internal/infrastructure/vectorstore/chromadb_collection_test.go`

## Покрытие критериев приемки

- AC-001 -> T2.1, T3.1
- AC-002 -> T2.2, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T2.3, T3.1
- AC-005 -> T2.3, T3.1
- AC-006 -> T1.1, T2.3
