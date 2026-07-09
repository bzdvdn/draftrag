# VectorStore: поддержка Weaviate

## Scope Snapshot

- In scope: реализация `VectorStore` и `VectorStoreWithFilters` для Weaviate; публичный конструктор `NewWeaviateStore`; хелперы управления коллекцией (`CreateCollection`, `DeleteCollection`, `CollectionExists`).
- Out of scope: гибридный BM25+semantic поиск через Weaviate (отдельная фича); использование Weaviate-модулей для генерации эмбеддингов (embedder всегда внешний); многотенантность Weaviate.

## Цель

Пользователь, работающий с Weaviate как основным векторным хранилищем, должен подключить его к draftRAG одной строкой — как это уже работает для Qdrant и pgvector — получив поиск по вектору, фильтрацию по ParentID и фильтрацию по произвольным метаданным без изменения остального кода pipeline.

## Основной сценарий

1. Пользователь создаёт коллекцию в Weaviate (`draftrag.CreateWeaviateCollection`) и получает `VectorStore` через `draftrag.NewWeaviateStore(opts)`.
2. Подключает store к `NewPipeline` — остальной код pipeline не меняется.
3. `pipeline.Index` / `pipeline.IndexBatch` сохраняет чанки в Weaviate через `Upsert`.
4. `pipeline.Search(...).Cite(ctx)` выполняет `Search` → Weaviate возвращает top-K чанков по cosine similarity.
5. При использовании `.Filter(f)` pipeline вызывает `SearchWithMetadataFilter` — Weaviate возвращает только чанки с совпадающими свойствами.
6. Если Weaviate недоступен — `NewWeaviateStore` возвращает ошибку конфигурации; операции возвращают сетевые ошибки, совместимые с `RetryEmbedder`/`RetryLLMProvider`.

## Scope

- `internal/infrastructure/vectorstore/weaviate.go` — реализация `VectorStore` + `VectorStoreWithFilters`
- `internal/infrastructure/vectorstore/weaviate_test.go` — unit-тесты с mock HTTP-сервером
- `pkg/draftrag/weaviate.go` — публичный API: `WeaviateOptions`, `NewWeaviateStore`, `CreateWeaviateCollection`, `DeleteWeaviateCollection`, `WeaviateCollectionExists`

## Контекст

- Все существующие VectorStore реализованы в `internal/infrastructure/vectorstore/` и экспортированы через тонкую обёртку в `pkg/draftrag/`.
- `VectorStoreWithFilters` расширяет `VectorStore` двумя методами: `SearchWithFilter` (ParentID) и `SearchWithMetadataFilter` (произвольные поля).
- `Chunk` содержит поля `ID`, `Content`, `ParentID`, `Embedding`, `Position`, `Metadata map[string]string` — все должны сохраняться и восстанавливаться через Weaviate.
- Weaviate Go client v4 (`github.com/weaviate/weaviate-client-go/v4`) — актуальная версия с gRPC-транспортом.
- Weaviate хранит объекты как "objects" внутри "collection" (ранее "class"). UUID объекта = ID чанка (детерминированная генерация из строки).
- Existing stores (Qdrant, pgvector) реализуют `VectorStoreWithFilters`; Weaviate должен делать то же самое для паритета.

## Требования

- RQ-001 `NewWeaviateStore` ДОЛЖЕН принимать `WeaviateOptions` (host, scheme, collection name, опциональный API key) и возвращать `(VectorStore, error)`.
- RQ-002 `Upsert` ДОЛЖЕН сохранять `ID`, `Content`, `ParentID`, `Position`, `Metadata` как свойства объекта и `Embedding` как вектор.
- RQ-003 `Search` ДОЛЖЕН выполнять near-vector поиск и возвращать top-K чанков с score (certainty или distance), отсортированных по убыванию score.
- RQ-004 `SearchWithFilter` ДОЛЖЕН ограничивать результаты объектами с `parentId` из переданного списка.
- RQ-005 `SearchWithMetadataFilter` ДОЛЖЕН ограничивать результаты объектами, у которых каждое поле `filter.Fields` совпадает с соответствующим свойством объекта.
- RQ-006 `Delete` ДОЛЖЕН удалять объект по ID; если объект не существует — не возвращать ошибку.
- RQ-007 `CreateWeaviateCollection` ДОЛЖЕН создать коллекцию со схемой, пригодной для хранения чанков; если коллекция уже существует — не возвращать ошибку.

## Вне scope

- Гибридный BM25+vector поиск через Weaviate (`HybridSearcher` интерфейс) — отдельная фича.
- Использование Weaviate-модулей (`text2vec-*`) для генерации эмбеддингов — embedder всегда внешний.
- Многотенантность Weaviate (`multi-tenancy`).
- Weaviate Cloud OIDC/OAuth аутентификация — только API key.
- Автоматическая миграция схемы при изменении полей чанка.
- Поддержка cross-references и named vectors.

## Критерии приемки

### AC-001 Базовый round-trip: Upsert → Search

- Почему это важно: основная функция store — сохранить и найти.
- **Given** Weaviate-коллекция создана, `NewWeaviateStore` вернул store без ошибки
- **When** выполнен `Upsert(chunk)`, затем `Search(chunk.Embedding, topK=1)`
- **Then** возвращается `RetrievalResult` с одним чанком, у которого `ID`, `Content`, `ParentID`, `Position`, `Metadata` совпадают с исходным чанком; `Score > 0`
- Evidence: тест с mock Weaviate HTTP server проходит; `go test ./internal/infrastructure/vectorstore/... -run TestWeaviate` зелёный

### AC-002 SearchWithFilter по ParentID

- Почему это важно: паритет с Qdrant и pgvector — без этого фильтрация по документу не работает.
- **Given** в коллекции есть чанки с разными `ParentID`
- **When** вызван `SearchWithFilter(embedding, topK, ParentIDFilter{ParentIDs: ["doc-A"]})`
- **Then** все возвращённые чанки имеют `ParentID == "doc-A"`
- Evidence: тест проверяет, что чанки с другим ParentID не попадают в результат

### AC-003 SearchWithMetadataFilter

- Почему это важно: единственный способ фильтрации по категории/тегу без изменения кода pipeline.
- **Given** в коллекции есть чанки с разными значениями в `Metadata`
- **When** вызван `SearchWithMetadataFilter(embedding, topK, MetadataFilter{Fields: {"category": "go"}})`
- **Then** все возвращённые чанки имеют `Metadata["category"] == "go"`
- Evidence: тест с mock сервером проверяет структуру where-фильтра в запросе

### AC-004 Delete идемпотентен

- Почему это важно: pipeline вызывает Delete при reindex; несуществующий объект не должен прерывать поток.
- **Given** store подключён к Weaviate
- **When** вызван `Delete(id)` для несуществующего ID, затем для существующего
- **Then** оба вызова возвращают `nil`
- Evidence: тест проверяет оба случая

### AC-005 PublicAPI: NewWeaviateStore доступен из pkg/draftrag

- Почему это важно: пользователь работает только с `pkg/draftrag`, не с `internal`.
- **Given** корректный `WeaviateOptions`
- **When** вызван `draftrag.NewWeaviateStore(opts)`
- **Then** возвращается `(VectorStore, error)` без паники; `go build ./...` проходит
- Evidence: `go build ./...` ok; тест на `ErrInvalidConfig` при пустом host

## Допущения

- Weaviate Go client v4 (`github.com/weaviate/weaviate-client-go/v4`) используется как единственная зависимость для взаимодействия с Weaviate — паттерн как у Qdrant с его HTTP-клиентом.
- UUID объекта генерируется детерминированно из строки `Chunk.ID` (UUID v5 или аналог) — это позволяет `Delete` и повторный `Upsert` работать без хранения маппинга.
- Метаданные `Metadata map[string]string` сериализуются как отдельные свойства объекта с префиксом или как вложенный `additionalProperties` — конкретная схема фиксируется в plan.
- Score нормализован: `certainty` (0–1, Weaviate cosine) конвертируется в `Score` поля `RetrievedChunk`.
- Тесты используют mock HTTP-сервер (httptest.Server), не реальный Weaviate — аналогично существующим тестам ChromaDB.

## Краевые случаи

- `WeaviateOptions.Host` пустой → `NewWeaviateStore` возвращает ошибку конфигурации до сетевого вызова.
- `Search` при пустой коллекции → возвращает `RetrievalResult{}` без ошибки.
- `Upsert` одного чанка дважды (тот же ID) → идемпотентен, второй вызов обновляет объект.
- `MetadataFilter.Fields` пустой → `SearchWithMetadataFilter` ведёт себя как `Search`.

## Открытые вопросы

- none
