# Qdrant Hybrid Search План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи
Outputs: plan, data model
Stop if: spec слишком расплывчата для безопасного планирования

## Цель

Реализовать HybridSearcher интерфейс для Qdrant через Query API с Prefetch и Fusion.RRF. Добавить методы SearchHybrid, SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter в QdrantStore, используя Qdrant Query API для параллельного поиска по sparse и dense векторам с объединением результатов через RRF.

## Scope

- Реализация HybridSearcher интерфейса в internal/infrastructure/vectorstore/qdrant.go
- Реализация HybridSearcherWithFilters интерфейса с Query API
- Добавление compile-time assertions для новых интерфейсов
- Поддержка Query API с Prefetch для multi-vector retrieval
- Поддержка Fusion.RRF для объединения результатов
- Валидация HybridConfig
- Обработка ошибок Query API
- Unit-тесты для новых методов
- Явно остаётся нетронутым: другие vectorstores, reranking с late interaction models, другие fusion стратегии

## Implementation Surfaces

- internal/infrastructure/vectorstore/qdrant.go — существующая поверхность, добавление методов SearchHybrid, SearchHybridWithParentIDFilter, SearchHybridWithMetadataFilter и compile-time assertions
- internal/infrastructure/vectorstore/qdrant_test.go — существующая поверхность, добавление unit-тестов для гибридного поиска
- domain/interfaces.go — существующая поверхность, интерфейсы HybridSearcher и HybridSearcherWithFilters уже определены (не меняются)

## Влияние на архитектуру

- Локальное влияние: QdrantStore получает новые методы для гибридного поиска, реализует HybridSearcher и HybridSearcherWithFilters интерфейсы
- Влияние на интеграции: нет новых интеграций, использование существующего Qdrant API
- Migration/compatibility: не требуется, это additive change к существующему QdrantStore

## Acceptance Approach

- AC-001 -> добавить методы SearchHybrid в QdrantStore с использованием Query API Prefetch для sparse и dense векторов, compile-time assertion для HybridSearcher
- AC-002 -> формировать Query API запрос с Prefetch структурой для sparse и dense векторов с limit для каждого prefetch
- AC-003 -> формировать финальный Query API запрос с FusionQuery и Fusion.RRF для объединения результатов
- AC-004 -> добавить методы SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией в Prefetch структуре, compile-time assertion для HybridSearcherWithFilters
- AC-005 -> валидировать HybridConfig через config.Validate() в начале SearchHybrid, возвращать ошибку при неверной конфигурации
- AC-006 -> обрабатывать HTTP ошибки Query API и возвращать информативные ошибки

## Данные и контракты

- Эта фича не вводит новых persisted сущностей или state transitions
- Эта фича не вводит новых API или event boundaries
- Использует существующий интерфейс HybridSearcher из domain/interfaces.go
- Использует существующий HybridConfig из domain/interfaces.go
- Использует Query API Qdrant (внешний контракт)

## Стратегия реализации

### DEC-001 Использование Query API с Prefetch для multi-vector retrieval

- Why: Query API с Prefetch является нативным способом гибридного поиска в Qdrant, обеспечивает параллельный поиск по sparse и dense векторам
- Tradeoff: Query API требует Qdrant версии 1.10+, но это совместимо с современными инсталляциями
- Affects: internal/infrastructure/vectorstore/qdrant.go
- Validation: unit-тесты проверяют формирование Prefetch структуры и вызов Query API

### DEC-002 Использование Fusion.RRF для объединения результатов

- Why: RRF является стандартной стратегией fusion для гибридного поиска, обеспечивает корректное объединение рангов из sparse и dense поиска
- Tradeoff: RRF может быть менее точным чем weighted fusion для специфических use cases, но это компенсируется гибкостью HybridConfig
- Affects: internal/infrastructure/vectorstore/qdrant.go
- Validation: unit-тесты проверяют формирование FusionQuery с Fusion.RRF

## Incremental Delivery

### MVP (Первая ценность)

- Реализация SearchHybrid с Query API Prefetch и Fusion.RRF
- Compile-time assertion для HybridSearcher
- Unit-тесты для SearchHybrid
- Критерий готовности MVP: AC-001, AC-002, AC-003 покрыты

### Итеративное расширение

- Реализация SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter
- Compile-time assertion для HybridSearcherWithFilters
- Unit-тесты для методов с фильтрацией
- Критерий готовности: AC-004 покрыта

## Порядок реализации

1. Добавить compile-time assertion для HybridSearcher в qdrant.go
2. Реализовать SearchHybrid с Query API Prefetch и Fusion.RRF
3. Добавить unit-тесты для SearchHybrid
4. Добавить compile-time assertion для HybridSearcherWithFilters
5. Реализовать SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter
6. Добавить unit-тесты для методов с фильтрацией

## Риски

- Риск 1: Qdrant версия < 1.10 не поддерживает Query API
  Mitigation: документация в spec указывает требование Qdrant 1.10+, ошибка валидации при неподдерживаемой версии
- Риск 2: Sparse векторы не индексированы в коллекции
  Mitigation: fallback на чистый dense поиск или информативная ошибка, документация в spec указывает допущение
- Риск 3: Query API возвращает ошибку при некорректной конфигурации
  Mitigation: валидация HybridConfig и обработка ошибок Query API

## Rollout и compatibility

- Специальных rollout-действий не требуется, это additive change
- Обратная совместимость сохраняется, существующие методы QdrantStore не меняются
- Monitoring: пользователи могут использовать HybridConfig для контроля поведения

## Проверка

- Unit-тесты в internal/infrastructure/vectorstore/qdrant_test.go для SearchHybrid
- Unit-тесты в internal/infrastructure/vectorstore/qdrant_test.go для SearchHybridWithParentIDFilter
- Unit-тесты в internal/infrastructure/vectorstore/qdrant_test.go для SearchHybridWithMetadataFilter
- Compile-time assertions для HybridSearcher и HybridSearcherWithFilters
- Тесты подтверждают AC-001, AC-002, AC-003, AC-004, AC-005, AC-006

## Соответствие конституции

- [CONST-ARCH] Чистая архитектура: реализация в infrastructure слое, зависит только от domain интерфейсов
- [CONST-LANG] Язык Go 1.23+: код использует стандартную библиотеку и http.Client
- [CONST-INTERFACE] Интерфейсная абстракция: QdrantStore реализует HybridSearcher интерфейс
- [CONST-CONTEXT] Контекстная безопасность: SearchHybrid принимает context.Context и поддерживает отмену
- [CONST-TEST] Тестируемость: unit-тесты с mock HTTP сервером для Query API
