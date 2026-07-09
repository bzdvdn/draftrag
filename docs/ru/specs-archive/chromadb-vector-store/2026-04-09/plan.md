# ChromaDB vector store — План

## Phase Contract

Inputs: spec `chromadb-vector-store`, inspect report, контекст репозитория (существующие реализации pgvector, Qdrant, memory).
Outputs: plan.md, data-model.md.

## Цель

Создать реализацию `ChromaStore` в `internal/infrastructure/vectorstore/` — HTTP-клиент для ChromaDB REST API, поддерживающий базовые операции VectorStore и фильтрацию по метаданным. Форма реализации аналогична существующим `QdrantStore` и `PgVectorStore`.

## Scope

- **Зона реализации**: `internal/infrastructure/vectorstore/chromadb.go` — новая реализация
- **Зона реализации**: `internal/infrastructure/vectorstore/chromadb_test.go` — unit-тесты
- **Не трогаем**: domain-интерфейсы (уже определены), другие реализации VectorStore

## Implementation Surfaces

| Surface | Тип | Изменение | Почему |
|---------|-----|-----------|--------|
| `internal/infrastructure/vectorstore/chromadb.go` | Новый файл | Создание | Реализация ChromaStore |
| `internal/infrastructure/vectorstore/chromadb_test.go` | Новый файл | Создание | Unit-тесты для AC |
| `internal/infrastructure/vectorstore/qdrant.go` | Существующий | Reference | Паттерны для HTTP-клиента и структуры кода |

## Влияние на архитектуру

- **Локальное**: новый файл в infrastructure-слое, не затрагивает domain или application
- **Integration**: добавляется альтернативная реализация VectorStore — пользователи могут выбирать между pgvector, Qdrant, ChromaDB
- **Rollout**: нет breaking changes, новая функциональность опциональна

## Acceptance Approach

| AC | Подход к реализации | Surfaces | Observable Proof |
|----|---------------------|----------|----------------|
| AC-001 Успешный upsert | HTTP POST `/api/v1/collections/{name}/upsert` с embedding + metadata | chromadb.go | Тест: upsert → GET point по ID через API |
| AC-002 Поиск по эмбеддингу | HTTP POST `/api/v1/collections/{name}/query` с query_embeddings | chromadb.go | Тест: search возвращает чанки с distances → score |
| AC-003 Фильтрация по метаданным | where-фильтр в JSON формате ChromaDB при query | chromadb.go | Тест: query с where={"source": "doc1"} возвращает только matching |
| AC-004 Удаление чанка | HTTP POST `/api/v1/collections/{name}/delete` с ids массивом | chromadb.go | Тест: delete → search не возвращает deleted ID |
| AC-005 Валидация размерности | Проверка `len(chunk.Embedding) == s.dimension` перед upsert | chromadb.go | Тест: ErrEmbeddingDimensionMismatch при mismatch |
| AC-006 Context cancellation | `http.NewRequestWithContext` + проверка `ctx.Err()` | chromadb.go | Тест: timeout 1ms → DeadlineExceeded |
| AC-007 Автосоздание коллекции | При отсутствии коллекции → POST `/api/v1/collections` | chromadb.go | Тест: delete collection → upsert → GET collection exists |

## Данные и контракты

См. `data-model.md`. Кратко:
- Нет новых domain-сущностей — используется существующая `domain.Chunk`
- ChromaDB как external state: collections, embeddings, metadata
- API boundary: ChromaDB REST API v1 (формат JSON)
- Нет event contracts

## Стратегия реализации

### DEC-001 HTTP клиент без внешних SDK

**Why**: ChromaDB предоставляет REST API, стандартный `net/http` достаточен. SDK для Go (chromem-go) добавляет зависимость без значимых преимуществ для базовых операций.

**Tradeoff**: Ручная сериализация JSON-запросов/ответов; нужно следить за изменениями API.

**Affects**: `chromadb.go` — ручная маршалинг/unmarshaling структур ChromaDB.

**Validation**: Тесты с httptest.Server проверяют корректность запросов.

### DEC-002 Плоское хранение метаданных

**Why**: ChromaDB требует where-фильтры по полям metadata; плоская структура `{"key": "value"}` позволяет прямую фильтрацию без nested queries.

**Tradeoff**: Потенциальный конфликт ключей если Chunk имеет поле с тем же именем что metadata.

**Affects**: Маппинг `chunk.Metadata` на ChromaDB metadata в upsert; where-фильтры в search.

**Validation**: AC-003 — тест фильтрации по metadata.

### DEC-003 Автосоздание коллекции при первом доступе

**Why**: Упрощает первый запуск без ручной настройки ChromaDB; consistent с поведением других реализаций.

**Tradeoff**: Латентность на первый запрос; race condition если несколько goroutines одновременно проверяют коллекцию.

**Affects**: Lazy-инициализация внутри методов при отсутствии коллекции.

**Validation**: AC-007 — тест автосоздания.

### DEC-004 ID как строка (не UUID)

**Why**: `Chunk.ID` уже является строкой; ChromaDB принимает произвольные string IDs. Нет необходимости в UUID-конверсии.

**Tradeoff**: Пользователь отвечает за уникальность ID.

**Affects**: Прямое использование `chunk.ID` в ChromaDB point ID.

**Validation**: AC-001 — тест upsert с конкретным ID.

## Incremental Delivery

### MVP (Первая ценность)

1. Базовая структура `ChromaStore` с HTTP клиентом
2. Реализация `Upsert` без автосоздания коллекции
3. Реализация `Search` без фильтров
4. Базовые тесты для AC-001, AC-002

**Критерий готовности MVP**: тесты проходят с mock ChromaDB server.

### Итеративное расширение

| Итерация | Что добавляется | AC покрываются |
|----------|-----------------|----------------|
| 2 | `Delete`, `SearchWithFilter` (ParentID) | AC-004 |
| 3 | `SearchWithMetadataFilter`, автосоздание коллекции | AC-003, AC-007 |
| 4 | Валидация размерности, context cancellation | AC-005, AC-006 |

## Порядок реализации

1. **Сначала**: структура ChromaStore, NewChromaStore, compile-time проверки интерфейсов
2. **Параллельно (после базовой структуры)**: Upsert + Search базовые версии
3. **Затем**: Delete, SearchWithFilter
4. **Затем**: SearchWithMetadataFilter, автосоздание коллекции
5. **Наконец**: полные тесты для всех AC, edge cases

## Риски

| Риск | Mitigation |
|------|------------|
| ChromaDB API изменится в новой версии | Абстракция через `ChromaStore` — изменения локализованы; фиксируем tested version в комментариях |
| Различия в distance metric (cosine vs euclidean) | Явно используем cosine как default при создании коллекции; документируем в godoc |
| Performance с большими metadata | Ограничиваем размер metadata в валидации; документируем limits |
| Concurrent autocreate коллекции | `sync.Once` или проверка existence перед create с обработкой conflict |

## Rollout и compatibility

- **Backfill**: не требуется, новая функциональность
- **Feature flag**: не требуется, опциональная реализация интерфейса
- **Monitoring**: базовая observability через `Hooks` (если настроен)
- **Compatibility**: Go 1.21+, ChromaDB 0.4.x+ с HTTP API v1

## Проверка

| Проверка | Что подтверждает | AC/DEC |
|----------|------------------|--------|
| Unit-тест `TestChromaStore_Upsert` | Upsert сохраняет чанк | AC-001 |
| Unit-тест `TestChromaStore_Search` | Search возвращает результаты | AC-002 |
| Unit-тест `TestChromaStore_SearchWithMetadataFilter` | Фильтрация работает | AC-003 |
| Unit-тест `TestChromaStore_Delete` | Удаление работает | AC-004 |
| Unit-тест `TestChromaStore_DimensionMismatch` | Валидация размерности | AC-005 |
| Unit-тест `TestChromaStore_ContextCancellation` | Context уважается | AC-006 |
| Unit-тест `TestChromaStore_AutocreateCollection` | Автосоздание работает | AC-007 |
| `go vet ./...` | Нет статических ошибок | — |
| Покрытие ≥60% | Достаточно тестов | SC-001 |

## Соответствие конституции

| Ограничение конституции | Статус | Пояснение |
|------------------------|--------|-----------|
| **Интерфейсная абстракция** | ✓ | `ChromaStore` реализует `VectorStore` и `VectorStoreWithFilters` |
| **Clean Architecture** | ✓ | Реализация в infrastructure-слое, зависимость направлена внутрь к domain |
| **Контекстная безопасность** | ✓ | Все операции принимают `context.Context` первым параметром |
| **Тестируемость** | ✓ | Unit-тесты с mock HTTP server; compile-time проверки интерфейсов |
| **Минимальная конфигурация** | ✓ | Разумные defaults для endpoint, таймаутов, размерности |
| **Языковая политика** | ✓ | Godoc на русском, код на английском |

Конфликтов нет.
