# Поддержка Milvus как векторного хранилища

## Scope Snapshot

- In scope: реализация `MilvusStore`, удовлетворяющей `domain.VectorStore` и `domain.VectorStoreWithFilters`, через Milvus REST API v2 (без официального SDK).
- Out of scope: поддержка Milvus gRPC API, управление коллекциями/схемами (создание/удаление), интеграция с Milvus Managed Cloud (Zilliz).

## Цель

Разработчики, использующие draftRAG, смогут подключить Milvus как векторное хранилище наравне с Qdrant, ChromaDB и Weaviate. Успех — когда `MilvusStore` проходит те же операции (Upsert, Delete, Search, SearchWithFilter, SearchWithMetadataFilter), что и другие реализации, без добавления тяжёлых зависимостей в `go.mod`.

## Основной сценарий

1. Стартовая точка: у пользователя запущен Milvus 2.3+ и существует коллекция с нужной схемой.
2. Основное действие: пользователь создаёт `NewMilvusStore(host, collection, token)` и передаёт его как `domain.VectorStore` в RAG-пайплайн.
3. Результат: чанки успешно индексируются (`Upsert`), удаляются (`Delete`, `DeleteByParentID`) и возвращаются при семантическом поиске (`Search`, `SearchWithFilter`, `SearchWithMetadataFilter`).
4. Fallback: если Milvus недоступен или возвращает HTTP-ошибку, метод возвращает описательную `error` без паники.

## Scope

- Новый файл `internal/infrastructure/vectorstore/milvus.go` со структурой `MilvusStore`
- Новый файл `internal/infrastructure/vectorstore/milvus_test.go` с unit-тестами (мок-сервер)
- Compile-time assertions на `domain.VectorStore`, `domain.VectorStoreWithFilters`, `domain.DocumentStore`
- Использование только стандартной библиотеки Go + уже существующих зависимостей (`encoding/json`, `net/http`)

## Контекст

- Все существующие реализации (`weaviate.go`, `qdrant.go`, `chromadb.go`) используют raw HTTP без официального SDK — этот же паттерн ДОЛЖЕН быть применён для Milvus (Milvus REST API v2, доступен с версии 2.3).
- Milvus REST API v2 базируется на `POST /v2/vectordb/entities/...`; аутентификация через заголовок `Authorization: Bearer <token>`.
- Конституция запрещает добавление зависимостей без явной необходимости; gRPC SDK (`milvus-sdk-go`) является тяжёлой зависимостью и явно вне scope.
- Domain-интерфейс `VectorStore` и `VectorStoreWithFilters` стабилен и не требует изменений.
- Схема коллекции Milvus (поля `id`, `text`, `parent_id`, `metadata`, `vector`) считается существующей на стороне пользователя.

## Требования

- RQ-001 `MilvusStore` ДОЛЖЕН реализовывать `domain.VectorStore` (Upsert, Delete, Search).
- RQ-002 `MilvusStore` ДОЛЖЕН реализовывать `domain.VectorStoreWithFilters` (SearchWithFilter по ParentID, SearchWithMetadataFilter).
- RQ-003 `MilvusStore` ДОЛЖЕН реализовывать `domain.DocumentStore` (DeleteByParentID).
- RQ-004 Взаимодействие с Milvus ДОЛЖНО происходить исключительно через Milvus REST API v2 (`/v2/vectordb/entities/`), без использования gRPC или официального SDK.
- RQ-005 `NewMilvusStore(baseURL, collection, token string) *MilvusStore` — публичный конструктор с HTTP-клиентом (timeout 10s).
- RQ-006 При HTTP-ошибке или ненулевом `code` в теле ответа метод ДОЛЖЕН возвращать `error` с описанием кода и сообщения из API.
- RQ-007 Все публичные типы и функции ДОЛЖНЫ иметь godoc-комментарии на русском языке.
- RQ-008 Тестовое покрытие `milvus.go` ДОЛЖНО быть ≥60% (infrastructure-слой, конституция §Ключевые измерения качества).

## Вне scope

- Создание/удаление коллекций Milvus — управление схемой остаётся на пользователе.
- Поддержка Milvus < 2.3 (нет REST API v2).
- Интеграция с Zilliz Cloud или любыми управляемыми сервисами помимо стандартного Milvus.
- gRPC-транспорт и официальный `milvus-sdk-go`.
- Гибридный поиск (BM25 + semantic) — `HybridSearcher` не входит в эту фичу.
- Пагинация результатов сверх `topK`.

## Критерии приемки

### AC-001 Upsert сохраняет чанк

- Почему это важно: без Upsert индексирование документов невозможно.
- **Given** `MilvusStore` сконфигурирован с корректным baseURL и collection
- **When** вызывается `Upsert(ctx, chunk)` с валидным `Chunk`
- **Then** отправляется POST на `/v2/vectordb/entities/upsert`, функция возвращает `nil`
- Evidence: unit-тест с мок-HTTP-сервером проверяет тело запроса и возвращает `nil` ошибку.

### AC-002 Delete удаляет чанк по ID

- Почему это важно: корректное удаление чанков необходимо для поддержания актуальности индекса.
- **Given** `MilvusStore` подключён к Milvus
- **When** вызывается `Delete(ctx, id)`
- **Then** отправляется POST на `/v2/vectordb/entities/delete` с фильтром `id == "<id>"`, возвращается `nil`
- Evidence: unit-тест с мок-сервером проверяет фильтр-выражение в теле запроса.

### AC-003 Search возвращает TopK результатов

- Почему это важно: поиск — ключевая операция RAG retrieval.
- **Given** в коллекции есть векторы
- **When** вызывается `Search(ctx, embedding, topK)`
- **Then** отправляется POST на `/v2/vectordb/entities/search`, возвращается `RetrievalResult` с ≤topK чанками
- Evidence: unit-тест с мок-сервером, возвращающим тестовые данные; результат корректно десериализован.

### AC-004 SearchWithFilter фильтрует по ParentID

- Почему это важно: позволяет ограничить поиск чанками одного документа.
- **Given** `MilvusStore` реализует `VectorStoreWithFilters`
- **When** вызывается `SearchWithFilter(ctx, embedding, topK, ParentIDFilter{ParentIDs: ["doc1"]})`
- **Then** в запросе к Milvus присутствует фильтр-выражение `parent_id in ["doc1"]`
- Evidence: unit-тест проверяет тело запроса.

### AC-005 SearchWithMetadataFilter фильтрует по метаданным

- Почему это важно: обеспечивает гибкую фильтрацию по произвольным полям метаданных.
- **Given** `MilvusStore` реализует `VectorStoreWithFilters`
- **When** вызывается `SearchWithMetadataFilter(ctx, embedding, topK, MetadataFilter{Fields: {"source": "wiki"}})`
- **Then** в запросе присутствует фильтр `metadata["source"] == "wiki"` (или аналогичное выражение Milvus)
- Evidence: unit-тест проверяет наличие фильтра в теле запроса.

### AC-006 DeleteByParentID удаляет все чанки документа

- Почему это важно: позволяет удалять документ целиком без перебора отдельных чанков.
- **Given** `MilvusStore` реализует `domain.DocumentStore`
- **When** вызывается `DeleteByParentID(ctx, "doc1")`
- **Then** отправляется POST на `/v2/vectordb/entities/delete` с фильтром `parent_id == "doc1"`
- Evidence: unit-тест с мок-сервером проверяет фильтр-выражение.

### AC-007 Compile-time assertions

- Почему это важно: гарантирует соответствие интерфейсам без запуска тестов.
- **Given** пакет компилируется
- **When** выполняется `go build ./...`
- **Then** compile-time assertions `var _ domain.VectorStore = (*MilvusStore)(nil)` и т.д. не вызывают ошибок
- Evidence: `go build ./...` завершается без ошибок.

### AC-008 Ошибки HTTP/API обёрнуты

- Почему это важно: явные ошибки помогают пользователям диагностировать проблемы конфигурации.
- **Given** Milvus возвращает HTTP 4xx/5xx или `{"code": <non-zero>}`
- **When** вызывается любой публичный метод
- **Then** метод возвращает не-nil `error` с кодом и сообщением; паники нет
- Evidence: unit-тест проверяет error-path.

## Допущения

- Коллекция Milvus уже создана пользователем с полями: `id` (VARCHAR PK), `text` (VARCHAR), `parent_id` (VARCHAR), `metadata` (JSON), `vector` (FLOAT_VECTOR).
- Milvus версии ≥ 2.3 с включённым REST API v2.
- Токен аутентификации передаётся как Bearer-токен; пустая строка означает отсутствие аутентификации.
- Размерность вектора определяется коллекцией на стороне Milvus; `MilvusStore` не управляет размерностью.
- `MetadataFilter.Fields` — `map[string]string`; реализация конвертирует в Milvus-выражение через `AND`-конъюнкцию.

## Критерии успеха

- SC-001 `go build ./...` завершается без ошибок и новых предупреждений `go vet`.
- SC-002 `go test ./internal/infrastructure/vectorstore/...` — все тесты проходят; coverage для `milvus.go` ≥ 60%.
- SC-003 В `go.mod` не появляются новые зависимости (только stdlib).

## Краевые случаи

- Пустой `ParentIDFilter.ParentIDs`: метод `SearchWithFilter` вызывает `Search` без фильтра.
- Пустой `MetadataFilter.Fields` (nil или len==0): метод `SearchWithMetadataFilter` вызывает `Search` без фильтра.
- Milvus возвращает пустой массив `data` при Search: возвращается `RetrievalResult` с пустым слайсом чанков, ошибки нет.
- Timeout HTTP-клиента (10s): возвращается ошибка сети, завёрнутая через `fmt.Errorf`.
- `id` чанка содержит спецсимволы: значение экранируется в JSON-строке стандартным `encoding/json`.

## Открытые вопросы

- none
