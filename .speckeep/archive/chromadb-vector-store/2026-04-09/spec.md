# ChromaDB vector store

## Scope Snapshot

- In scope: реализация `ChromaStore` — backend для работы с ChromaDB через HTTP API, поддерживающий базовые операции VectorStore и метаданные
- Out of scope: гибридный поиск (BM25), persistence-состояние коллекций между перезапусками, distributed ChromaDB

## Цель

Пользователи библиотеки draftRAG получают возможность использовать ChromaDB в качестве векторного хранилища — популярное решение для прототипирования и Python-экосистемы. Реализация позволяет быстро разворачивать RAG-системы без необходимости сложной инфраструктуры.

## Основной сценарий

1. Пользователь запускает ChromaDB (локально или через Docker)
2. Создаёт `ChromaStore` через `NewChromaStore(baseURL, collection, dimension)` с указанием endpoint и размерности эмбеддингов
3. Использует `Upsert()` для индексации чанков с эмбеддингами и метаданными
4. Выполняет `Search()` для семантического поиска похожих чанков
5. При необходимости использует `SearchWithMetadataFilter()` для фильтрации по метаданным

## Scope

- Реализация `ChromaStore` — структура с HTTP клиентом для ChromaDB API
- Поддержка базовых операций `domain.VectorStore`: `Upsert`, `Delete`, `Search`
- Реализация `domain.VectorStoreWithFilters`: `SearchWithFilter`, `SearchWithMetadataFilter`
- Поддержка метаданных чанков при upsert и фильтрации при поиске
- Автоматическое создание коллекции с указанной размерностью при первом использовании
- Runtime-опции для таймаутов и лимитов (аналогично QdrantStore)
- Валидация размерности эмбеддингов
- Поддержка `context.Context` для cancellation и timeout

## Контекст

- ChromaDB использует HTTP REST API для операций с коллекциями и точками
- Интерфейс `VectorStore` определён в `internal/domain/interfaces.go`
- Модель `Chunk` содержит поля: ID, Content, ParentID, Embedding, Position, Metadata
- ChromaDB поддерживает where-фильтры для метаданных в формате JSON
- Clean Architecture: реализация находится в `internal/infrastructure/vectorstore/`
- Существующие реализации (pgvector, Qdrant) служат reference для структуры кода
- Go 1.21+ с поддержкой `context`, `net/http`, `encoding/json`

## Требования

- RQ-001 `ChromaStore` ДОЛЖЕН реализовывать интерфейс `domain.VectorStore`
- RQ-002 `ChromaStore` ДОЛЖЕН реализовывать интерфейс `domain.VectorStoreWithFilters`
- RQ-003 `Upsert` ДОЛЖЕН сохранять чанк с эмбеддингом и метаданными в ChromaDB
- RQ-004 `Delete` ДОЛЖЕН удалять чанк по ID из коллекции
- RQ-005 `Search` ДОЛЖЕН выполнять similarity search и возвращать `RetrievalResult` с отсортированными по score чанками
- RQ-006 `SearchWithMetadataFilter` ДОЛЖЕН применять where-фильтр ChromaDB для точного совпадения по метаданным
- RQ-007 При отсутствии коллекции система ДОЛЖНА автоматически создавать её с указанной размерностью
- RQ-008 Валидация размерности эмбеддинга ДОЛЖНА возвращать `ErrEmbeddingDimensionMismatch` при несоответствии
- RQ-009 Все операции ДОЛЖНЫ уважать `context.Context` для cancellation и timeout
- RQ-010 Клиент ДОЛЖЕН иметь разумные значения по умолчанию для таймаутов (Search: 2s, Upsert/Delete: 5s)

## Вне scope

- Гибридный поиск (BM25 + semantic) — требует отдельной интеграции
- Поддержка Streaming API (ChromaDB не предоставляет streaming для поиска)
- Persistence настроек коллекции между перезапусками
- Поддержка multi-tenant сценариев с tenant/workspace изоляцией
- Автоматическая миграция схемы данных при изменениях
- Retry и circuit breaker логика (будет покрыта отдельной обёрткой)
- Batch-операции для массового upsert/delete

## Критерии приемки

### AC-001 Успешный upsert чанка

- Почему это важно: базовая операция индексации должна работать корректно
- **Given** валидный `Chunk` с эмбеддингом и метаданными
- **When** вызывается `ChromaStore.Upsert(ctx, chunk)`
- **Then** чанк сохраняется в ChromaDB с корректным ID, эмбеддингом и payload
- Evidence: тест выполняет upsert и подтверждает запись через прямой GET из ChromaDB

### AC-002 Поиск по эмбеддингу возвращает релевантные результаты

- Почему это важно: семантический поиск — ключевая функция RAG
- **Given** индексированные чанки с разными эмбеддингами
- **When** вызывается `ChromaStore.Search(ctx, queryEmbedding, topK=3)`
- **Then** возвращаются top-3 наиболее похожих чанка с корректными score > 0
- Evidence: тест с фиксированными эмбеддингами проверяет порядок и значения score

### AC-003 Фильтрация по метаданным работает корректно

- Почему это важно: пользователи должны фильтровать поиск по полям документа
- **Given** чанки с метаданными `{source: "doc1"}` и `{source: "doc2"}`
- **When** вызывается `SearchWithMetadataFilter` с фильтром `source=doc1`
- **Then** возвращаются только чанки с matching метаданными
- Evidence: тест создаёт чанки с разными source, фильтрует и проверяет результат

### AC-004 Удаление чанка по ID

- Почему это важно: необходима возможность удалять устаревшие данные
- **Given** существующий чанк с ID в коллекции
- **When** вызывается `ChromaStore.Delete(ctx, id)`
- **Then** чанк удаляется из коллекции, последующий поиск не возвращает его
- Evidence: тест удаляет чанк и проверяет отсутствие через поиск

### AC-005 Валидация размерности эмбеддинга

- Почему это важно: предотвращает silent corruption данных
- **Given` `ChromaStore` с dimension=384, чанк с embedding длины 512
- **When** вызывается `Upsert(ctx, chunk)`
- **Then** возвращается ошибка `ErrEmbeddingDimensionMismatch`
- Evidence: тест проверяет возврат специфической ошибки при несоответствии размерностей

### AC-006 Context cancellation прерывает операцию

- Почему это важно: пользователь должен контролировать время выполнения
- **Given** `context.Context` с timeout=1ms и медленная операция upsert
- **When** timeout истекает во время HTTP-запроса
- **Then** операция прерывается и возвращается `context.DeadlineExceeded`
- Evidence: тест с mock server и коротким timeout проверяет cancellation

### AC-007 Автосоздание коллекции

- Почему это важно: упрощает первый запуск без ручной настройки
- **Given` несуществующая коллекция `test_collection`
- **When** выполняется первый `Upsert` или `Search`
- **Then` коллекция автоматически создаётся с dimension из конфигурации store
- Evidence: тест удаляет коллекцию, выполняет операцию, проверяет создание через API

## Допущения

- ChromaDB доступен по HTTP (по умолчанию `http://localhost:8000` для Chroma 0.4.x+)
- ChromaDB версии 0.4.x или совместимой с HTTP API v1
- Коллекция создаётся с distance metric cosine (default для ChromaDB)
- Метаданные маппятся на flat JSON-структуру в ChromaDB payload
- One-to-one маппинг между `Chunk.ID` и ChromaDB point ID (строковый UUID или произвольный ID)
- Thread-safe доступ обеспечивается через HTTP-клиент (нет shared mutable state в структуре)
- Размерность эмбеддинга фиксируется при создании коллекции и не меняется динамически

## Критерии успеха

- SC-001 Покрытие unit-тестами ≥60% для ChromaStore (основные пути и ошибки)
- SC-002 Время ответа на `Search` с topK=10 не превышает 500ms для коллекции с 10k записей (локальный ChromaDB)
- SC-003 Все операции корректно обрабатывают HTTP-ошибки (4xx, 5xx) с понятными сообщениями

## Краевые случаи

- Пустая коллекция: Search возвращает пустой RetrievalResult без ошибки
- Пустой MetadataFilter: поведение идентично обычному Search
- Несуществующий ID при Delete: возвращается nil (идempotent удаление)
- Nil context: panic с понятным сообщением (consistent с другими реализациями)
- Очень длинные метаданные: ограничения ChromaDB на размер payload
- Специальные символы в ID чанка: должны корректно эскейпиться для URL

## Открытые вопросы

- none
