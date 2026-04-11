# VectorStore: Weaviate — План

## Phase Contract

Inputs: spec, inspect report, узкий контекст репозитория (chromadb.go, qdrant.go, pkg/draftrag/chromadb.go).
Outputs: plan.md, data-model.md.
Stop if: spec слишком расплывчата для архитектурных решений.

## Цель

Добавить три новых файла, реализующих `VectorStore` + `VectorStoreWithFilters` для Weaviate через Weaviate REST API v1 (raw HTTP, без официального SDK), по образцу существующих QdrantStore и ChromaStore. Основное решение — хранить `Chunk.Metadata` дважды: как отдельные `meta_*`-свойства (для server-side WHERE-фильтра) и как JSON-строку `chunkMetadata` (для надёжного восстановления при поиске). UUID объекта генерируется детерминированно через UUID v5 из `chunk.ID` без внешних зависимостей.

## Scope

- **Новый**: `internal/infrastructure/vectorstore/weaviate.go` — `WeaviateStore`, реализует `VectorStore` + `VectorStoreWithFilters`
- **Новый**: `internal/infrastructure/vectorstore/weaviate_test.go` — unit-тесты с `httptest.Server`
- **Новый**: `pkg/draftrag/weaviate.go` — `WeaviateOptions`, `NewWeaviateStore`, `CreateWeaviateCollection`, `DeleteWeaviateCollection`, `WeaviateCollectionExists`
- **Не затрагивается**: `internal/domain/` — интерфейсы и модели не меняются; `internal/application/` — pipeline не меняется; все существующие store-реализации

## Implementation Surfaces

- `internal/infrastructure/vectorstore/weaviate.go` — новая surface. Структура `WeaviateStore {baseURL, collection, apiKey string; client *http.Client}`. Пять методов: `Upsert`, `Delete`, `Search`, `SearchWithFilter`, `SearchWithMetadataFilter`. Два вспомогательных: `uuidFromID(id string) string`, `parseSearchResponse(body []byte) (domain.RetrievalResult, error)`. Compile-time assertions: `var _ domain.VectorStore = (*WeaviateStore)(nil)` и `var _ domain.VectorStoreWithFilters = (*WeaviateStore)(nil)`.
- `internal/infrastructure/vectorstore/weaviate_test.go` — новая surface. Тесты используют `httptest.NewServer`, регистрируют mock-handler на каждый endpoint (`POST /v1/objects`, `DELETE /v1/objects/...`, `POST /v1/graphql`, `POST /v1/schema`). Проверяют структуру запросов и возвращают предустановленные ответы.
- `pkg/draftrag/weaviate.go` — новая surface. Тонкая обёртка над `vectorstore.WeaviateStore`. `NewWeaviateStore` возвращает `(VectorStore, error)`; при пустом `opts.Host` — `ErrInvalidVectorStoreConfig`. `CreateWeaviateCollection` делает `POST /v1/schema`; `DeleteWeaviateCollection` — `DELETE /v1/schema/{collection}`; `WeaviateCollectionExists` — `GET /v1/schema/{collection}`.

## Влияние на архитектуру

- Только additive: три новых файла, zero изменений в существующих.
- `go.mod` не меняется — UUID v5 реализован внутри через `crypto/sha1` (stdlib).
- Новая зависимость не появляется: raw HTTP, стандартная библиотека Go.
- Обратная совместимость: полная; pipeline-код пользователей не меняется.

## Acceptance Approach

- **AC-001** (Upsert → Search round-trip): `Upsert` → `POST /v1/objects` с телом `{class, id (UUID v5), vector, properties:{chunkId, content, parentId, position, chunkMetadata}}`. `Search` → `POST /v1/graphql` с near-vector запросом, возвращает `_additional{id, certainty}` + фиксированные properties. Парсинг GraphQL-ответа восстанавливает все поля чанка. Score = `certainty`. Тест: mock HTTP server, проверяем ID/Content/ParentID/Metadata в результате, Score > 0.
- **AC-002** (SearchWithFilter, ParentID): `SearchWithFilter` добавляет в GraphQL-запрос `where: {path:["parentId"], operator: Equal/ContainsAny, valueText/valueTextArray: [...]}`. Тест: mock сервер проверяет наличие `where`-блока в теле GraphQL-запроса.
- **AC-003** (SearchWithMetadataFilter): `SearchWithMetadataFilter` строит `where: {operator: And, operands: [{path:["meta_key"], operator: Equal, valueText: "val"}...]}`. Тест: mock сервер проверяет наличие `meta_`-префикса в `path`.
- **AC-004** (Delete идемпотентен): `Delete` → `DELETE /v1/objects/{collection}/{uuid}`. При 204 — nil. При 404 — nil (не ошибка). Тест: два вызова, оба возвращают nil.
- **AC-005** (PublicAPI): `pkg/draftrag.NewWeaviateStore` при `opts.Host == ""` возвращает `ErrInvalidVectorStoreConfig`. `go build ./...` проходит. Тест: проверяем ошибку на пустом host.

## Данные и контракты

Фича вводит Weaviate-специфичный data model для хранения `Chunk`. Детали в `data-model.md`.

- Изменения domain-моделей: нет.
- API contracts: нет (Weaviate — внешняя система, не публичный API библиотеки).
- `contracts/` не создаётся: фича не проходит через API boundary библиотеки.

Эта фича затрагивает data model потому что вводит конкретную схему Weaviate-коллекции и Chunk↔объект маппинг, требующий явной фиксации.

## Стратегия реализации

- **DEC-001 Raw HTTP вместо официального Weaviate Go client v4**
  Why: Все существующие stores (Qdrant, ChromaDB) используют raw `net/http`. Официальный клиент v4 использует gRPC-транспорт, несовместимый с `httptest.Server` для unit-тестов. Паттерн репозитория — raw HTTP.
  Tradeoff: GraphQL-запросы придётся конструировать вручную как строки; нет type-safety от клиента.
  Affects: `weaviate.go`, `weaviate_test.go`
  Validation: `TestWeaviate*` проходят с mock HTTP server; `go build ./...` ok.

- **DEC-002 UUID v5 через stdlib (crypto/sha1), без внешних зависимостей**
  Why: Конституция требует минимального набора зависимостей. UUID v5 = SHA-1(namespace + name), реализуется в ~15 строк. `go.mod` не растёт.
  Tradeoff: Собственная реализация вместо battle-tested `github.com/google/uuid`; покрывается unit-тестом.
  Affects: `weaviate.go` (функция `uuidFromID`)
  Validation: Тест идемпотентности Upsert: один и тот же chunk → один и тот же UUID → второй Upsert обновляет, не дублирует.

- **DEC-003 Metadata: dual-write (flat meta_\* + JSON chunkMetadata)**
  Why: Weaviate GraphQL возвращает только явно перечисленные properties. Динамические `meta_*` ключи невозможно перечислить в запросе без introspection. Решение: хранить полный `Metadata` как JSON-строку `chunkMetadata` (для надёжного чтения) И как отдельные `meta_{key}` свойства (для server-side WHERE-фильтра в SearchWithMetadataFilter). GraphQL-запрос перечисляет только фиксированные properties + `chunkMetadata`.
  Tradeoff: Дублирование данных при Upsert; удваивает хранимый объём metadata; Weaviate auto-schema добавляет `meta_*` динамически.
  Affects: `Upsert` (dual-write), `Search`/`SearchWithFilter`/`SearchWithMetadataFilter` (чтение из `chunkMetadata`), `SearchWithMetadataFilter` (WHERE на `meta_*`).
  Validation: AC-003 — тест проверяет `path: ["meta_category"]` в WHERE-блоке запроса; AC-001 — `chunk.Metadata` корректно восстанавливается из `chunkMetadata`.

- **DEC-004 Score = certainty из `_additional`**
  Why: Weaviate возвращает `certainty` (0–1) для cosine similarity — естественный аналог `Score` в `domain.RetrievedChunk`. Альтернатива `distance` потребовала бы инверсии (1 − distance), что менее читаемо.
  Tradeoff: `certainty` специфичен для cosine metric; при смене метрики нужна замена.
  Affects: `parseSearchResponse` в `weaviate.go`.
  Validation: AC-001 — Score > 0 в результате поиска.

- **DEC-005 Upsert = попытка PUT → 404 → POST**
  Why: `PUT /v1/objects/{class}/{id}` заменяет существующий объект (idempotent update); если 404 — создаём через `POST /v1/objects`. Один сетевой round-trip для обновления, два для первичной записи. Это корректнее чем POST+422 handling, т.к. `PUT` семантически — replace.
  Tradeoff: Два HTTP-запроса для первого Upsert нового объекта.
  Affects: `Upsert` в `weaviate.go`.
  Validation: AC-001 — Upsert дважды с одним ID не возвращает ошибку; тест mock сервера обрабатывает оба endpoint.

## Порядок реализации

1. **Сначала**: `weaviate.go` — структура + `uuidFromID` + `Upsert` + `Delete` + `Search`. Это MVP для AC-001 и AC-004.
2. **Затем**: `SearchWithFilter` и `SearchWithMetadataFilter` — строятся поверх `Search` с добавлением WHERE-блока. AC-002 и AC-003.
3. **Параллельно с шагом 2**: `weaviate_test.go` — mock HTTP server тесты для всех методов.
4. **Последним**: `pkg/draftrag/weaviate.go` — публичные обёртки + коллекция-хелперы. AC-005. Зависит от готового `WeaviateStore`.

## Риски

- **GraphQL-конструирование вручную**: синтаксические ошибки в строках-запросах сложно отловить без реального Weaviate.
  Mitigation: Тесты с mock HTTP server проверяют структуру запроса (parse JSON body, проверить `query` field). Добавить минимальный GraphQL-парсер на уровне проверки присутствия ключевых слов.

- **auto-schema может быть отключён в Weaviate**: если пользователь отключил auto-schema, `meta_*` свойства не добавятся автоматически при Upsert.
  Mitigation: Документировать в godoc `Upsert`, что Weaviate должен иметь auto-schema включённым (по умолчанию). `CreateWeaviateCollection` объявляет только фиксированные свойства; `meta_*` добавляются динамически.

- **UUID v5 коллизии**: теоретически возможны, но вероятность пренебрежимо мала при строковых chunk.ID.
  Mitigation: Покрыть `uuidFromID` unit-тестом с несколькими входами; задокументировать ограничение.

## Rollout и compatibility

Специальных rollout-действий не требуется: чисто additive, новые файлы, нет изменений в существующем public API. `go build ./...` как единственная проверка совместимости.

## Проверка

- `TestWeaviateUpsertSearch` — AC-001: mock server для `PUT`, `POST`, `POST /v1/graphql`; проверить ID/Content/ParentID/Metadata/Score.
- `TestWeaviateSearchWithFilter` — AC-002: проверить наличие `parentId` WHERE-блока в GraphQL body.
- `TestWeaviateSearchWithMetadataFilter` — AC-003: проверить `meta_` prefix в WHERE-path.
- `TestWeaviateDeleteIdempotent` — AC-004: mock 404 и 204, оба вызова → nil.
- `TestWeaviateNewStore_InvalidConfig` — AC-005: пустой host → ErrInvalidVectorStoreConfig.
- `go build ./...` — AC-005 (build passes).
- `go test ./internal/infrastructure/vectorstore/... -run TestWeaviate` — все проходят.

## Соответствие конституции

- **Интерфейсная абстракция**: `WeaviateStore` реализует `domain.VectorStore` и `domain.VectorStoreWithFilters` через compile-time assertions. Пользователь видит только интерфейс `VectorStore` из `pkg/draftrag`. ✓
- **Чистая архитектура**: implementation в `internal/infrastructure/vectorstore/`, public API в `pkg/draftrag/`; domain-слой не меняется и не импортирует внешние пакеты. ✓
- **Контекстная безопасность**: все методы принимают `context.Context` как первый параметр; nil-context вызывает panic (паттерн репозитория). ✓
- **Тестируемость**: mock HTTP server вместо реального Weaviate; compile-time interface assertions. ✓
- **Минимальные зависимости**: no new `go.mod` dependency; UUID v5 через stdlib. ✓
- **Язык документации**: godoc-комментарии на русском языке. ✓
- **Тестовое покрытие**: ≥60% для infrastructure (unit-тесты с mock server покрывают все публичные методы). ✓
