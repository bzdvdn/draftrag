# Weaviate Hybrid Search

## Scope Snapshot

- In scope: реализация HybridSearcher интерфейса для Weaviate через GraphQL API с BM25 + semantic fusion
- Out of scope: поддержка других vectorstores, reranking с late interaction models, другие fusion стратегии

## Цель

Разработчики получают возможность использовать гибридный поиск (BM25 + semantic fusion) в Weaviate через интерфейс HybridSearcher. Это обеспечивает параллельное выполнение BM25 и dense поиска с объединением результатов через fusion-стратегию.

## Основной сценарий

1. Разработчик создаёт WeaviateStore через NewWeaviateStore
2. Разработчик вызывает SearchHybrid с query, embedding, topK и HybridConfig
3. Weaviate выполняет BM25 поиск по query и dense поиск по embedding через GraphQL API
4. Weaviate объединяет результаты через fusion-стратегию (RRF или weighted)
5. Метод возвращает RetrievalResult с объединёнными чанками и скорами от fusion

## Scope

- Реализация HybridSearcher интерфейса в internal/infrastructure/vectorstore/weaviate.go
- Реализация HybridSearcherWithFilters интерфейса с фильтрацией по ParentID и метаданным
- Использование Weaviate GraphQL API для hybrid search
- Поддержка BM25 + dense fusion через Weaviate native capabilities
- Валидация HybridConfig
- Обработка ошибок GraphQL API
- Unit-тесты для новых методов
- Явно остаётся нетронутым: другие vectorstores, reranking с late interaction models, другие fusion стратегии

## Контекст

- Weaviate поддерживает hybrid search через GraphQL API с BM25 и dense векторами
- Интерфейс HybridSearcher уже определён в domain/interfaces.go
- HybridSearcherWithFilters расширяет HybridSearcher возможностью фильтрации
- Qdrant уже реализует аналогичный функционал через Query API
- Weaviate использует GraphQL вместо REST API

## Требования

- RQ-001 WeaviateStore ДОЛЖНА реализовать метод SearchHybrid с использованием Weaviate GraphQL hybrid search API
- RQ-002 SearchHybrid ДОЛЖЕН использовать Weaviate GraphQL query с BM25 и nearVector для multi-vector retrieval
- RQ-003 SearchHybrid ДОЛЖЕН поддерживать fusion-стратегии: RRF (Reciprocal Rank Fusion) или weighted fusion
- RQ-004 WeaviateStore ДОЛЖНА реализовать методы SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией
- RQ-005 SearchHybrid ДОЛЖЕН валидировать HybridConfig через config.Validate() перед выполнением поиска
- RQ-006 Код ДОЛЖЕН обрабатывать GraphQL ошибки и возвращать информативные ошибки

## Вне scope

- Поддержка других vectorstores (chromadb, milvus, memory)
- Reranking с late interaction models (ColBERT, SPLADE)
- Matryoshka embeddings и multi-step retrieval
- Другие fusion стратегии кроме RRF и weighted
- Управление коллекциями Weaviate (создание/удаление) — это отдельная capability

## Критерии приемки

### AC-001 Реализация HybridSearcher интерфейса

- Почему это важно: обеспечивает совместимость с существующей архитектурой и позволяет использовать Weaviate в гибридном поиске
- **Given** WeaviateStore инициализирован с базовым URL, коллекцией и размерностью
- **When** вызывается метод SearchHybrid с query, embedding, topK и HybridConfig
- **Then** метод выполняет hybrid search через Weaviate GraphQL API и возвращает RetrievalResult
- Evidence: код weaviate.go содержит метод SearchHybrid и compile-time assertion для HybridSearcher

### AC-002 Использование GraphQL API с BM25 и nearVector

- Почему это важно: BM25 и nearVector являются стандартными методами hybrid search в Weaviate
- **Given** SearchHybrid вызван с query и embedding
- **When** формируется GraphQL запрос
- **Then** запрос содержит BM25 search по query и nearVector search по embedding
- Evidence: код weaviate.go содержит GraphQL запрос с bm25 и nearVector полями

### AC-003 Использование fusion-стратегии

- Почему это важно: fusion-стратегия определяет, как объединяются результаты из BM25 и dense поиска
- **Given** BM25 и nearVector возвращают результаты
- **When** формируется финальный запрос fusion
- **Then** запрос содержит fusion с типом rrf или weighted в зависимости от HybridConfig
- Evidence: код weaviate.go содержит fusion поле в GraphQL запросе

### AC-004 Реализация HybridSearcherWithFilters

- Почему это важно: обеспечивает фильтрацию результатов гибридного поиска по ParentID и метаданным
- **Given** WeaviateStore реализует HybridSearcher
- **When** вызываются методы SearchHybridWithParentIDFilter или SearchHybridWithMetadataFilter
- **Then** методы выполняют hybrid search с фильтрацией в GraphQL запросе
- Evidence: код weaviate.go содержит методы SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter и compile-time assertion для HybridSearcherWithFilters

### AC-005 Валидация HybridConfig

- Почему это важно: предотвращает выполнение поиска с некорректной конфигурацией
- **Given** SearchHybrid вызван с HybridConfig
- **When** config содержит невалидные значения (SemanticWeight вне [0,1], RRFK < 1)
- **Then** метод возвращает ошибку ErrInvalidHybridConfig
- Evidence: код weaviate.go вызывает config.Validate() в начале SearchHybrid

### AC-006 Обработка ошибок GraphQL API

- Почему это важно: обеспечивает информативные ошибки при проблемах с Weaviate API
- **Given** GraphQL запрос выполняется
- **When** Weaviate возвращает ошибку (network error, query error, invalid response)
- **Then** метод возвращает информативную ошибку с деталями
- Evidence: код weaviate.go обрабатывает ошибки GraphQL клиента и HTTP ошибки

## Допущения

- Weaviate версия >= 1.20 поддерживает hybrid search через GraphQL API
- BM25 индекс уже создан в коллекции Weaviate для текстовых полей
- Dense векторы уже индексированы в коллекции Weaviate
- GraphQL endpoint доступен по базовому URL Weaviate
- Поля для BM25 поиска и векторные поля настроены в схеме Weaviate

## Критерии успеха

- SC-001 Unit-тесты покрывают все методы SearchHybrid, SearchHybridWithParentIDFilter, SearchHybridWithMetadataFilter
- SC-002 Compile-time assertions проходят без ошибок для HybridSearcher и HybridSearcherWithFilters

## Краевые случаи

- Пустой query: метод возвращает ошибку ErrEmptyQueryText
- Пустой список ParentIDs или Fields в фильтрах: метод выполняет обычный SearchHybrid без фильтрации
- GraphQL API возвращает ошибку: метод возвращает информативную ошибку
- Timeout запроса: метод возвращает context cancellation error
- Некорректная размерность embedding: метод возвращает ErrEmbeddingDimensionMismatch

## Открытые вопросы

- none
