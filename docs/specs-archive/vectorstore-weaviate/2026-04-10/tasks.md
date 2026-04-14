# VectorStore: Weaviate — Задачи

## Phase Contract

Inputs: plan.md, data-model.md.
Outputs: упорядоченные исполнимые задачи с покрытием всех 5 AC.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/vectorstore/weaviate.go` | T1.1, T1.2, T1.3, T2.1, T2.2, T2.3 |
| `internal/infrastructure/vectorstore/weaviate_test.go` | T4.1 |
| `pkg/draftrag/weaviate.go` | T3.1 |

## Фаза 1: Инфраструктура — структура и CRUD

Цель: создать файл, базовую структуру, детерминированный UUID и методы `Upsert`/`Delete` — минимальный набор для AC-001 и AC-004.

- [x] T1.1 Создать `WeaviateStore` + `uuidFromID` + compile-time assertions — файл компилируется, `var _ domain.VectorStore = (*WeaviateStore)(nil)` проходит
  Touches: internal/infrastructure/vectorstore/weaviate.go
  AC: DEC-001, DEC-002

- [x] T1.2 Реализовать `Upsert` — PUT `/v1/objects/{class}/{id}`, если 404 → POST `/v1/objects`; тело включает `chunkId`, `content`, `parentId`, `position`, `chunkMetadata` (JSON) и `meta_{key}` для каждого ключа Metadata; вектор в поле `vector`
  Touches: internal/infrastructure/vectorstore/weaviate.go
  AC: AC-001, RQ-002, DEC-003, DEC-005

- [x] T1.3 Реализовать `Delete` — `DELETE /v1/objects/{class}/{uuid}`; HTTP 204 и 404 оба возвращают nil
  Touches: internal/infrastructure/vectorstore/weaviate.go
  AC: AC-004, RQ-006

## Фаза 2: Поиск — Search и фильтры

Цель: реализовать все три search-метода с GraphQL-запросами к `POST /v1/graphql`.

- [x] T2.1 Реализовать `Search` — GraphQL near-vector запрос, limit=topK, properties: `chunkId content parentId position chunkMetadata _additional{id certainty}`; парсинг ответа восстанавливает все поля Chunk; Score = certainty
  Touches: internal/infrastructure/vectorstore/weaviate.go
  AC: AC-001, RQ-003, DEC-004

- [x] T2.2 Реализовать `SearchWithFilter` — вызов `Search` с добавлением WHERE-блока `path:["parentId"] operator:Equal/ContainsAny`; пустой ParentIDs → делегировать в Search без where-клаузы
  Touches: internal/infrastructure/vectorstore/weaviate.go
  AC: AC-002, RQ-004

- [x] T2.3 Реализовать `SearchWithMetadataFilter` — вызов Search с WHERE `{operator:And, operands:[{path:["meta_{key}"], operator:Equal, valueText:"{val}"}...]}`; пустой filter.Fields → делегировать в Search
  Touches: internal/infrastructure/vectorstore/weaviate.go
  AC: AC-003, RQ-005

## Фаза 3: Публичный API

Цель: открыть `WeaviateStore` через `pkg/draftrag/weaviate.go`; `go build ./...` проходит.

- [x] T3.1 Создать `pkg/draftrag/weaviate.go` — `WeaviateOptions{Host, Scheme, Collection, APIKey}`; `NewWeaviateStore` возвращает `(VectorStore, error)`, при пустом `Host` → `ErrInvalidVectorStoreConfig`; `CreateWeaviateCollection` → `POST /v1/schema`; `DeleteWeaviateCollection` → `DELETE /v1/schema/{class}`; `WeaviateCollectionExists` → `GET /v1/schema/{class}`
  Touches: pkg/draftrag/weaviate.go
  AC: AC-005, RQ-001, RQ-007

## Фаза 4: Тесты

Цель: доказать корректность всех AC с mock HTTP server; `go test ./internal/infrastructure/vectorstore/... -run TestWeaviate` зелёный.

- [x] T4.1 Добавить `weaviate_test.go` с mock HTTP server — покрыть: `TestWeaviateUpsertSearch` (AC-001), `TestWeaviateSearchWithFilter` (проверить `parentId` в WHERE, AC-002), `TestWeaviateSearchWithMetadataFilter` (проверить `meta_` prefix в path, AC-003), `TestWeaviateDeleteIdempotent` (AC-004), `TestWeaviateNewStore_InvalidConfig` (AC-005)
  Touches: internal/infrastructure/vectorstore/weaviate_test.go
  AC: AC-001, AC-002, AC-003, AC-004, AC-005

## Покрытие критериев приемки

- AC-001 -> T1.2, T2.1, T4.1
- AC-002 -> T2.2, T4.1
- AC-003 -> T2.3, T4.1
- AC-004 -> T1.3, T4.1
- AC-005 -> T3.1, T4.1
