# Milvus Hybrid Search

## Scope Snapshot

- In scope: реализация HybridSearcher интерфейса для Milvus через Multi-Vector Search API с BM25 + dense vector fusion
- Out of scope: поддержка других vectorstores, reranking с custom strategies, другие fusion стратегии кроме RRF/weighted

## Цель

Разработчики получают возможность использовать гибридный поиск (BM25 + semantic fusion) в Milvus через интерфейс HybridSearcher. Это обеспечивает параллельное выполнение BM25 и dense поиска с объединением результатов через fusion-стратегию для улучшения качества релевантности в RAG-сценариях.

## Основной сценарий

1. Разработчик создаёт MilvusStore через NewMilvusStore
2. Разработчик вызывает SearchHybrid с query, embedding, topK и HybridConfig
3. Milvus выполняет BM25 поиск по query (sparse vector) и dense поиск по embedding через AnnSearchRequest
4. Milvus объединяет результаты через hybrid_search() с fusion-стратегией (RRF или weighted)
5. Метод возвращает RetrievalResult с объединёнными чанками и скорами от fusion

## Scope

- Реализация HybridSearcher интерфейса в internal/infrastructure/vectorstore/milvus.go
- Реализация HybridSearcherWithFilters интерфейса с фильтрацией через expr
- Использование Milvus Multi-Vector Search API с AnnSearchRequest
- Поддержка BM25 (sparse) + dense fusion через Milvus native capabilities
- Валидация HybridConfig
- Обработка ошибок Milvus API
- Unit-тесты для новых методов
- Явно остаётся нетронутым: другие vectorstores, reranking с custom strategies, другие fusion стратегии

## Контекст

- Milvus поддерживает Multi-Vector Hybrid Search через AnnSearchRequest и hybrid_search()
- Интерфейс HybridSearcher уже определён в domain/interfaces.go
- HybridSearcherWithFilters расширяет HybridSearcher возможностью фильтрации
- PGVectorStore и QdrantStore уже реализуют аналогичный функционал
- Milvus использует AnnSearchRequest для каждого векторного поля (dense и sparse)
- Milvus поддерживает BM25 через sparse vectors и full-text search
- Fusion в Milvus выполняется через reranking strategies (RRF, weighted)

## Требования

- RQ-001 MilvusStore ДОЛЖНА реализовать метод SearchHybrid с использованием Milvus Multi-Vector Search API
- RQ-002 SearchHybrid ДОЛЖЕН использовать AnnSearchRequest для BM25 (sparse) и dense векторного поиска
- RQ-003 SearchHybrid ДОЛЖЕН поддерживать fusion-стратегии: RRF (Reciprocal Rank Fusion) или weighted fusion через rerank strategy
- RQ-004 MilvusStore ДОЛЖНА реализовать методы SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией через expr
- RQ-005 SearchHybrid ДОЛЖЕН валидировать HybridConfig через config.Validate() перед выполнением поиска
- RQ-006 Код ДОЛЖЕН обрабатывать ошибки Milvus API и возвращать информативные ошибки

## Вне scope

- Поддержка других vectorstores (chromadb, memory)
- Reranking с custom strategies или late interaction models
- Другие fusion стратегии кроме RRF и weighted
- Изменение существующих методов MilvusStore (Upsert, Search, Delete)
- Миграция существующих данных или schema changes

## Критерии приемки

### AC-001 MilvusStore реализует HybridSearcher интерфейс

- Почему это важно: обеспечивает совместимость с domain-интерфейсом для гибридного поиска
- **Given** MilvusStore инициализирован с валидными параметрами подключения
- **When** разработчик проверяет compile-time assertion для HybridSearcher интерфейса
- **Then** компиляция успешна без ошибок, MilvusStore реализует все методы HybridSearcher
- Evidence: `var _ domain.HybridSearcher = (*MilvusStore)(nil)` в milvus.go без ошибок компиляции

### AC-002 SearchHybrid использует BM25 и dense векторы через AnnSearchRequest

- Почему это важно: обеспечивает multi-vector retrieval для улучшения релевантности
- **Given** MilvusStore инициализирован, коллекция содержит чанки с dense и sparse векторами
- **When** разработчик вызывает SearchHybrid с query, embedding, topK и HybridConfig
- **Then** метод создаёт два AnnSearchRequest (один для sparse/BM25, один для dense) и вызывает hybrid_search()
- Evidence: код создаёт AnnSearchRequest для text_sparse и text_dense полей и вызывает hybrid_search() с rerank strategy

### AC-003 SearchHybrid поддерживает fusion-стратегии RRF и weighted

- Почему это важно: позволяет настраивать баланс между BM25 и semantic search
- **Given** MilvusStore инициализирован, HybridConfig с UseRRF=true или UseRRF=false
- **When** разработчик вызывает SearchHybrid с HybridConfig.UseRRF=true
- **Then** hybrid_search() использует RRF rerank strategy; при UseRRF=false использует weighted fusion
- Evidence: код передаёт соответствующую rerank strategy в hybrid_search() в зависимости от HybridConfig.UseRRF

### AC-004 MilvusStore реализует HybridSearcherWithFilters с фильтрацией

- Почему это важно: обеспечивает гибридный поиск с возможностью фильтрации по ParentID и метаданным
- **Given** MilvusStore инициализирован, коллекция содержит чанки с parentId и metadata
- **When** разработчик вызывает SearchHybridWithParentIDFilter или SearchHybridWithMetadataFilter
- **Then** методы добавляют expr фильтр в AnnSearchRequest и выполняют hybrid search с фильтрацией
- Evidence: код добавляет expr параметр в AnnSearchRequest для фильтрации по parentId или metadata

### AC-005 SearchHybrid валидирует HybridConfig перед выполнением

- Почему это важно: предотвращает выполнение поиска с невалидной конфигурацией
- **Given** MilvusStore инициализирован
- **When** разработчик вызывает SearchHybrid с невалидной HybridConfig (SemanticWeight вне [0,1], RRFK < 1)
- **Then** метод возвращает ошибку валидации без вызова Milvus API
- Evidence: код вызывает config.Validate() и возвращает ошибку, если проверка не прошла

### AC-006 Код обрабатывает ошибки Milvus API информативно

- Почему это важно: позволяет разработчикам быстро диагностировать проблемы с Milvus
- **Given** MilvusStore инициализирован, Milvus API возвращает ошибку (timeout, invalid request, connection error)
- **When** разработчик вызывает SearchHybrid и Milvus API возвращает ошибку
- **Then** метод возвращает ошибку с описанием причины (статус код, сообщение от Milvus)
- Evidence: код проверяет ошибки от Milvus API и оборачивает их в информативные error messages

## Допущения

- Milvus версия 2.5+ с поддержкой Multi-Vector Hybrid Search
- Коллекция Milvus уже настроена с полями для dense и sparse векторов
- Milvus API доступен и отвечает в разумные сроки
- Sparse векторы для BM25 уже вычислены и хранятся в коллекции
- Dense векторы для semantic search уже вычислены и хранятся в коллекции
- HybridConfig структура уже определена в domain/models.go и имеет метод Validate()

## Критерии успеха

- none

## Краевые случаи

- Пустая коллекция: hybrid search возвращает пустой RetrievalResult без ошибки
- Пустой query: SearchHybrid возвращает ErrEmptyQueryText
- Невалидный topK: SearchHybrid возвращает ErrInvalidQueryTopK
- Milvus API недоступен: SearchHybrid возвращает ошибку с описанием connection error
- Пустой фильтр в SearchHybridWithParentIDFilter: делегирует в SearchHybrid без WHERE
- Пустой фильтр в SearchHybridWithMetadataFilter: делегирует в SearchHybrid без WHERE
- Context cancellation: SearchHybrid возвращает context error без вызова Milvus API

## Открытые вопросы

- none
