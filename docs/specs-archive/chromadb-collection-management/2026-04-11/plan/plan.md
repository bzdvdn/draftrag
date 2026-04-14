# ChromaDB: управление коллекциями — План

## Phase Contract

Inputs: spec.md, inspect.md, `internal/domain/interfaces.go`, `internal/infrastructure/vectorstore/chromadb.go`.
Outputs: plan.md, data-model.md.
Stop if: ChromaDB REST API v1 меняет контракт — но это repository constraint, уже подтверждённый существующим кодом.

## Цель

Добавить интерфейс `CollectionManager` в domain-слой и реализовать три метода в `ChromaStore`: `CreateCollection`, `DeleteCollection`, `CollectionExists`. Приватный `createCollection` поглощается публичным `CreateCollection`. Изменения ограничены двумя файлами + тесты.

## Scope

- Новый интерфейс `CollectionManager` в `internal/domain/interfaces.go`
- Три новых публичных метода и одна compile-time assertion в `internal/infrastructure/vectorstore/chromadb.go`
- Unit-тесты через `httptest.NewServer` в новом файле `internal/infrastructure/vectorstore/chromadb_collection_test.go`
- Существующие методы `VectorStore`, `VectorStoreWithFilters`, `DocumentStore` и тесты — не трогаются

## Implementation Surfaces

- `internal/domain/interfaces.go` — существующий файл, добавляется интерфейс `CollectionManager` (опциональная capability, как `DocumentStore`)
- `internal/infrastructure/vectorstore/chromadb.go` — существующий файл; приватный `createCollection` становится публичным `CreateCollection`; добавляются `DeleteCollection` и `CollectionExists`; вызов `createCollection` внутри `SearchWithMetadataFilter` заменяется на `CreateCollection`
- `internal/infrastructure/vectorstore/chromadb_collection_test.go` — новый файл для тестов трёх методов

## Влияние на архитектуру

- Domain-слой получает новый опциональный интерфейс — не ломает ни одного существующего потребителя
- `ChromaStore` реализует четыре domain-интерфейса (было три) — additive изменение
- Нет изменений в публичном API (`pkg/draftrag/`), application-слое или других хранилищах
- Rollout: чисто additive, никаких migration или флагов не нужно

## Acceptance Approach

- AC-001 (CreateCollection idempotent) → `CreateCollection` использует `POST /api/v1/collections` с `get_or_create: true`; возвращает `nil` при 200 и 201; тест: mock-сервер, two calls — оба ожидают `nil`
- AC-002 (DeleteCollection, happy path) → `DeleteCollection` использует `DELETE /api/v1/collections/{name}`; тест: mock захватывает `r.Method == "DELETE"` и путь
- AC-003 (DeleteCollection, HTTP error) → при статусе ≠ 200/204 → `fmt.Errorf("chromadb: status=%d", ...)` с кодом; тест: mock возвращает 404, проверка `err != nil` и строки
- AC-004 (CollectionExists → true) → `GET /api/v1/collections/{name}` возвращает 200 → `(true, nil)`; тест: mock с 200
- AC-005 (CollectionExists → false при 404) → статус 404 → `(false, nil)` без ошибки; тест: mock с 404, проверка `exists == false && err == nil`
- AC-006 (compile-time assertion) → `var _ domain.CollectionManager = (*ChromaStore)(nil)` в chromadb.go; proof: `go build ./...`

## Данные и контракты

Эта фича не вводит новых domain-сущностей, persisted state или event contracts.
`data-model.md` содержит только placeholder.
Contracts не создаются — ChromaDB REST API является внешней зависимостью, не API boundary самой библиотеки.

## Стратегия реализации

- DEC-001 `CollectionManager` как опциональный domain-интерфейс (не embed в `VectorStore`)
  Why: паттерн уже применён для `DocumentStore`, `VectorStoreWithFilters`, `HybridSearcher` — не все хранилища управляют коллекциями, forcing его в `VectorStore` сломало бы in-memory и pgvector
  Tradeoff: клиентский код должен делать type assertion `if cm, ok := store.(domain.CollectionManager); ok` — небольшая awkwardness
  Affects: `internal/domain/interfaces.go`, `internal/infrastructure/vectorstore/chromadb.go`
  Validation: compile-time assertion (AC-006); `go build ./...` без ошибок

- DEC-002 `CreateCollection` поглощает приватный `createCollection`
  Why: иметь и публичный и приватный метод с одной логикой — дублирование; внутренний autocreate в `SearchWithMetadataFilter` может просто вызвать `s.CreateCollection(ctx)`
  Tradeoff: слегка меняется поведение `SearchWithMetadataFilter` — вместо автосоздания при 404 будет явный вызов `CreateCollection`, но результат идентичен
  Affects: `internal/infrastructure/vectorstore/chromadb.go` (метод `SearchWithMetadataFilter`)
  Validation: существующие тесты `chromadb_test.go` и `chromadb_delete_test.go` должны оставаться зелёными

- DEC-003 `CollectionExists` возвращает `(false, nil)` при HTTP 404
  Why: 404 означает «не существует», а не «ошибка сервера» — клиентский код получает чёткое разделение
  Tradeoff: любой другой не-200 статус (400, 500) возвращает `(false, error)` — это корректно, так как причина неизвестна
  Affects: `internal/infrastructure/vectorstore/chromadb.go`, тест AC-005
  Validation: unit-тест: mock 404 → `exists == false && err == nil`; mock 500 → `exists == false && err != nil`

## Порядок реализации

1. Добавить `CollectionManager` в `internal/domain/interfaces.go` — это unblocks compile-time assertion
2. Реализовать методы в `chromadb.go`: `CreateCollection` (заменяет `createCollection`), `DeleteCollection`, `CollectionExists`; обновить `SearchWithMetadataFilter` для вызова `s.CreateCollection`; добавить compile-time assertion
3. Написать тесты в `chromadb_collection_test.go`

Шаги 1 и 2 зависимы (интерфейс нужен для assertion). Тесты пишутся после реализации.

## Риски

- ChromaDB 0.4.x возвращает 200, 0.5.x — 201 на создание коллекции
  Mitigation: существующий `createCollection` уже обрабатывает оба кода — `CreateCollection` наследует это поведение

- Существующий `SearchWithMetadataFilter` с autocreate: замена приватного `createCollection` на публичный `CreateCollection` может сломать путь 404-retry, если сигнатура изменится
  Mitigation: сигнатура не меняется, `CreateCollection(ctx)` — прямой drop-in

## Rollout и compatibility

Никаких специальных rollout-действий не требуется. Фича строго additive: новый интерфейс + новые методы. Существующий публичный API библиотеки (`pkg/draftrag/`) не затрагивается.

## Проверка

- `TestChromaCreateCollection_Idempotent` — mock сервер, два вызова, проверка `nil` (AC-001)
- `TestChromaDeleteCollection_HappyPath` — mock захватывает `DELETE /api/v1/collections/docs`, проверка nil (AC-002)
- `TestChromaDeleteCollection_HTTPError` — mock 404, проверка `err != nil` и наличия кода в строке (AC-003)
- `TestChromaCollectionExists_True` — mock 200, проверка `(true, nil)` (AC-004)
- `TestChromaCollectionExists_False` — mock 404, проверка `(false, nil)` (AC-005)
- `go build ./...` — подтверждает compile-time assertion (AC-006, DEC-001)
- `go vet ./...` — структурная проверка (конституция)
- Существующие тесты `chromadb_test.go` + `chromadb_delete_test.go` не должны сломаться (DEC-002)

## Соответствие конституции

- **Интерфейсная абстракция**: `CollectionManager` объявлен в domain-слое без импорта внешних пакетов — выполняется
- **Чистая архитектура**: domain не импортирует infrastructure; реализация — в infrastructure; зависимость направлена внутрь — выполняется
- **context.Context**: все три метода принимают `ctx context.Context` первым аргументом — выполняется
- **Тестовое покрытие ≥60% для infrastructure**: шесть новых тестов покрывают все новые методы — выполняется
- **Godoc-комментарии на русском**: все новые публичные типы и методы будут снабжены godoc — выполняется
- **Go 1.21+**: используются только стандартные конструкции, `any` вместо `interface{}` — выполняется
- Конфликтов нет
