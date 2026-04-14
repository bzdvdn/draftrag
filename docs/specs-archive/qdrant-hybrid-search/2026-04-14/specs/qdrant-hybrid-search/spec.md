# Qdrant Hybrid Search

## Scope Snapshot

- In scope: реализация HybridSearcher интерфейса для Qdrant через Query API с BM25 и semantic fusion
- Out of scope: поддержка других vectorstores (weaviate, chromadb, milvus), reranking с late interaction models

## Цель

Разработчики получают возможность использовать гибридный поиск (BM25 + semantic fusion) в Qdrant через интерфейс HybridSearcher. Реализация использует Query API Qdrant с Prefetch и Fusion.RRF для объединения результатов из sparse и dense векторов.

## Основной сценарий

1. Разработчик создаёт QdrantStore с индексированными sparse и dense векторами
2. Разработчик вызывает SearchHybrid с query, embedding и HybridConfig
3. QdrantStore использует Query API с Prefetch для параллельного поиска по sparse и dense векторам
4. QdrantStore объединяет результаты через Fusion.RRF и возвращает RetrievalResult

## Scope

- Реализация HybridSearcher интерфейса в internal/infrastructure/vectorstore/qdrant.go
- Реализация HybridSearcherWithFilters интерфейса с Query API
- Поддержка Query API с Prefetch для multi-vector retrieval
- Поддержка Fusion.RRF для объединения результатов
- Конфигурация через HybridConfig (аналогично pgvector)

## Контекст

- Qdrant поддерживает Query API с Prefetch и Fusion начиная с версии 1.10
- Sparse vectors (BM25) и dense vectors (semantic) индексируются отдельно
- Fusion.RRF (Reciprocal Rank Fusion) используется для объединения результатов
- Текущая реализация QdrantStore не реализует HybridSearcher интерфейс
- pgvector уже имеет реализацию гибридного поиска через RRF

## Требования

- RQ-001 QdrantStore ДОЛЖЕН реализовывать интерфейс HybridSearcher с методом SearchHybrid
- RQ-002 SearchHybrid ДОЛЖЕН использовать Query API с Prefetch для sparse и dense векторов
- RQ-003 SearchHybrid ДОЛЖЕН использовать Fusion.RRF для объединения результатов
- RQ-004 QdrantStore ДОЛЖЕН реализовывать интерфейс HybridSearcherWithFilters
- RQ-005 SearchHybridWithParentIDFilter ДОЛЖЕН использовать Query API с фильтрацией по ParentID
- RQ-006 SearchHybridWithMetadataFilter ДОЛЖЕН использовать Query API с фильтрацией по метаданным

## Вне scope

- Поддержка других vectorstores (weaviate, chromadb, milvus, memory)
- Reranking с late interaction models (ColBERT, SPLADE)
- Matryoshka embeddings и multi-step retrieval
- Другие fusion стратегии кроме RRF

## Критерии приемки

### AC-001 Реализация HybridSearcher интерфейса

- Почему это важно: обеспечивает совместимость с существующей архитектурой draftRAG и паритет с pgvector
- **Given** разработчик создаёт QdrantStore с индексированными sparse и dense векторами
- **When** разработчик вызывает SearchHybrid с query, embedding и HybridConfig
- **Then** метод возвращает RetrievalResult с объединёнными результатами из sparse и dense поиска
- Evidence: QdrantStore реализует метод SearchHybrid, соответствующий сигнатуре HybridSearcher

### AC-002 Использование Query API с Prefetch

- Почему это важно: Query API с Prefetch является нативным способом гибридного поиска в Qdrant
- **Given** QdrantStore выполняет гибридный поиск
- **When** формируется запрос к Query API
- **Then** запрос содержит Prefetch структуру для sparse и dense векторов с limit для каждого prefetch
- Evidence: код qdrant.go содержит Query API вызов с Prefetch для sparse и dense векторов

### AC-003 Использование Fusion.RRF

- Почему это важно: RRF является стандартной стратегией fusion для гибридного поиска
- **Given** Prefetch возвращает результаты из sparse и dense поиска
- **When** формируется финальный запрос fusion
- **Then** запрос содержит FusionQuery с Fusion.RRF для объединения результатов
- Evidence: код qdrant.go содержит FusionQuery с Fusion.RRF

### AC-004 Реализация HybridSearcherWithFilters

- Почему это важно: обеспечивает фильтрацию результатов гибридного поиска по ParentID и метаданным
- **Given** разработчик вызывает SearchHybridWithParentIDFilter или SearchHybridWithMetadataFilter
- **When** формируется запрос к Query API
- **Then** запрос содержит фильтр в Prefetch или fusion структуре
- Evidence: QdrantStore реализует методы SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter

### AC-005 Валидация HybridConfig

- Почему это важно: обеспечивает корректную конфигурацию гибридного поиска
- **Given** разработчик вызывает SearchHybrid с некорректной HybridConfig
- **When** метод валидирует конфигурацию
- **Then** метод возвращает ошибку с описанием проблемы
- Evidence: SearchHybrid вызывает config.Validate() и возвращает ошибку при неверной конфигурации

### AC-006 Обработка ошибок Query API

- Почему это важно: обеспечивает информативные сообщения при сбоях Qdrant
- **Given** Qdrant API возвращает ошибку Query API
- **When** SearchHybrid обрабатывает ответ
- **Then** метод возвращает ошибку с описанием проблемы
- Evidence: код обрабатывает HTTP ошибки Query API и возвращает ошибку

## Допущения

- Qdrant версия 1.10+ с поддержкой Query API
- Sparse векторы (BM25) уже индексированы в коллекции
- Dense векторы (semantic) уже индексированы в коллекции
- Qdrant API доступен и отвечает корректно

## Критерии успеха

none

## Краевые случаи

- Пустой query: возвращается ошибка валидации
- Отсутствие sparse векторов: fallback на чистый dense поиск
- Отсутствие dense векторов: fallback на чистый sparse поиск
- Пустой фильтр: фильтрация не применяется
- Network timeout: возвращается context error

## Открытые вопросы

none
