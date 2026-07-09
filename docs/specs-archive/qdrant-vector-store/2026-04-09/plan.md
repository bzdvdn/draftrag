# Qdrant vector store План

## Phase Contract

Inputs: spec qdrant-vector-store и минимальный контекст репозитория (существующие vectorstore реализации).
Outputs: plan.md, data-model.md.

## Цель

Реализация нового infrastructure-слоя — HTTP REST клиента для Qdrant, удовлетворяющего интерфейсам `VectorStore` и `VectorStoreWithFilters` из `internal/domain`. Клиент инкапсулирует все взаимодействия с Qdrant API, маппит доменные фильтры на payload-фильтры Qdrant, предоставляет factory-функции и миграции для управления коллекциями.

## Scope

- **Внутри**: `internal/infrastructure/vectorstore/qdrant.go` — новая реализация VectorStore
- **Внутри**: `internal/infrastructure/vectorstore/qdrant_test.go` — unit-тесты с HTTP mocks
- **Внутри**: `pkg/draftrag/qdrant.go` — публичные factory-функции и миграции
- **Внутри**: `pkg/draftrag/qdrant_test.go` — интеграционные тесты
- **Граница**: Qdrant REST API v1.7+ (порт 6333)
- **Не трогаем**: существующие pgvector, memory реализации; domain-интерфейсы

## Implementation Surfaces

| Surface | Статус | Почему участвует |
|---------|--------|------------------|
| `internal/infrastructure/vectorstore/qdrant.go` | Новая | Основная реализация VectorStore через Qdrant REST API |
| `internal/infrastructure/vectorstore/qdrant_test.go` | Новая | Unit-тесты с mock HTTP сервером |
| `pkg/draftrag/qdrant.go` | Новая | Публичный API: фабрики `NewQdrantStore`, миграции `CreateCollection`, `DeleteCollection` |
| `pkg/draftrag/qdrant_test.go` | Новая | Интеграционные тесты публичного API |
| `go.mod` | Изменяем | Добавление зависимости (если нужен HTTP client helper) или остаёмся на stdlib net/http |

## Влияние на архитектуру

- **Локальное**: Новый infrastructure пакет, не затрагивает существующие реализации
- **Интеграция**: Реализует существующие domain-интерфейсы без breaking changes
- **Clean Architecture**: Зависимость направлена внутрь — domain не знает о Qdrant
- **Compatibility**: Публичный API добавляется, существующий не меняется

## Acceptance Approach

| AC | Реализация | Surfaces | Proof |
|----|------------|----------|-------|
| AC-001 Базовый поиск | HTTP POST /collections/{name}/points/search, парсинг ответа | qdrant.go | Тест `TestQdrantStore_Search` проходит, возвращает RetrievedChunk с корректным Score |
| AC-002 Фильтр по ParentID | Маппинг `ParentIDFilter` на Qdrant payload filter `must: {key: "parent_id", match: {value: "..."}}` | qdrant.go | Тест `TestQdrantStore_SearchWithParentIDFilter` — только чанки с matching ParentID |
| AC-003 Фильтр по метаданным | Маппинг `MetadataFilter.Fields` на Qdrant `must: [{key: "metadata.k", match: {value: "v"}}, ...]` | qdrant.go | Тест `TestQdrantStore_SearchWithMetadataFilter` — все key=value совпадают |
| AC-004 Upsert/Delete | HTTP PUT /collections/{name}/points для upsert, POST .../points/delete для delete | qdrant.go | Тест `TestQdrantStore_UpsertDelete` — upsert добавляет, delete удаляет |
| AC-005 Миграции коллекций | HTTP PUT /collections/{name} с vector params (distance: Cosine), DELETE /collections/{name} | pkg/draftrag/qdrant.go | Тест `TestQdrantStore_Migrations` — коллекция создаётся с правильной конфигурацией |
| AC-006 Обработка ошибок | Проверка HTTP status codes, декодирование Qdrant error response | qdrant.go | Тесты на 404, 400 — ошибки содержат status и message |

## Данные и контракты

См. `data-model.md` для деталей маппинга доменных сущностей на Qdrant payload.

**API границы:**
- Исходящие: HTTP REST API Qdrant (localhost:6333 или заданный URL)
- Формат: JSON request/response
- Аутентификация: вне scope (базовая поддержка может быть добавлена позже)

**Контракты не требуются** — нет event-driven взаимодействия, только request-response.

## Стратегия реализации

### DEC-001 HTTP клиент на stdlib net/http

- **Why**: Qdrant REST API простой, не требует сложной retry-логики в MVP; stdlib даёт достаточный контроль над таймаутами через context
- **Tradeoff**: Без встроенного retry и connection pooling; при высокой нагрузке может потребоваться отдельный HTTP client
- **Affects**: `internal/infrastructure/vectorstore/qdrant.go` — структура store хранит `baseURL string` и использует `http.Client{Timeout: ...}`
- **Validation**: Unit-тесты с `httptest.Server` проходят, запросы содержат ожидаемые headers и body

### DEC-002 Плоский payload без nested objects

- **Why**: Соответствует допущению spec; Qdrant match-фильтры проще с плоской структурой
- **Tradeoff**: Невозможно фильтровать по вложенным структурам в metadata
- **Affects**: Маппинг `Chunk` на Qdrant point: `payload["parent_id"]=chunk.ParentID`, `payload["content"]=chunk.Content`, `payload["metadata.k"]=v` для каждого k,v
- **Validation**: Тесты фильтрации по metadata проходят

### DEC-003 UUID как ID чанка

- **Why**: Qdrant требует string ID для точек; domain Chunk.ID уже string
- **Tradeoff**: Нет проверки формата UUID внутри Qdrant store
- **Affects**: Прямое использование `chunk.ID` как point ID без трансформации
- **Validation**: Upsert с разными ID корректно создаёт разные точки

## Incremental Delivery

### MVP (Первая ценность)

- Файл `qdrant.go` с базовой структурой `QdrantStore`
- Реализация `Upsert`, `Delete`, `Search` (AC-001, AC-004 частично)
- Фабрика `NewQdrantStore` в `pkg/draftrag`
- Базовые unit-тесты с mock HTTP server

**Критерий готовности MVP**: `TestQdrantStore_Search` проходит, чанки сохраняются и находятся.

### Итеративное расширение

| Шаг | Что добавляем | AC | Валидация |
|-----|---------------|----|-----------|
| 2 | `SearchWithFilter` (ParentID) | AC-002 | Тест на фильтрацию проходит |
| 3 | `SearchWithMetadataFilter` | AC-003 | Тест на metadata фильтр проходит |
| 4 | `CreateCollection`, `DeleteCollection` | AC-005 | Миграции создают/удаляют коллекцию |
| 5 | Полная обработка ошибок | AC-006 | Тесты на ошибки Qdrant проходят |
| 6 | Дополнительные edge cases | — | Пустые фильтры, таймауты, пустые результаты |

## Порядок реализации

1. **Сначала**: Скелет `qdrant.go` — структура, конструктор, HTTP client setup
2. **Параллельно**: Базовые тесты с `httptest.Server` для проверки request/response формата
3. **Затем**: `Upsert` и `Search` (core функциональность)
4. **Затем**: `Delete` и фильтры (`SearchWithFilter`, `SearchWithMetadataFilter`)
5. **Затем**: Миграции в `pkg/draftrag/qdrant.go`
6. **Наконец**: Полный набор тестов на ошибки и edge cases

## Риски

| Риск | Mitigation |
|------|------------|
| Несовместимость с новыми версиями Qdrant API | Фиксируем поддержку версии 1.7+ в допущениях; pinned API endpoints |
| Размер payload превышает лимиты Qdrant | Runtime опция `MaxContentBytes`, валидация перед отправкой |
| Производительность при большом topK | Ограничение `MaxTopK` в runtime options; документирование лимитов |
| Отсутствие Qdrant для интеграционных тестов | Используем httptest mocks для unit; Docker-compose для ручной интеграции |

## Rollout и compatibility

- **Новый код**: Добавление только, существующие пользователи не затронуты
- **Zero-downtime**: Не применимо — новая реализация, не замена
- **Migration**: Не требуется, данные не переносятся
- **Feature flags**: Не требуются

## Проверка

| Что проверяем | Как | AC/DEC |
|---------------|-----|--------|
| Правильность HTTP запросов | Unit-тесты с `httptest.Server`, проверка URL, method, body | DEC-001 |
| Корректность маппинга фильтров | Тесты с разными `ParentIDFilter` и `MetadataFilter` | AC-002, AC-003 |
| Обработка ошибок API | Mock server возвращает 4xx/5xx, проверяем ошибки | AC-006 |
| Таймауты через context | Context with deadline, проверяем interruption | RQ-011 |
| Соответствие интерфейсам | Compile-time check `var _ domain.VectorStore = (*QdrantStore)(nil)` | RQ-001, RQ-002 |
| Покрытие тестами | `go test -cover`, цель ≥60% | SC-002 |

## Соответствие конституции

- **Interface abstraction**: Реализация удовлетворяет `VectorStore` и `VectorStoreWithFilters` — нет конфликтов
- **Clean Architecture**: Код в `internal/infrastructure/vectorstore/`, зависимость направлена к `internal/domain` — нет конфликтов
- **Context safety**: Все методы принимают `context.Context` — нет конфликтов
- **Testability**: HTTP client заменяется через тестовый server — нет конфликтов
- **Go 1.21+**: Используем stdlib + context — нет конфликтов

**Итог**: нет конфликтов с конституцией.
