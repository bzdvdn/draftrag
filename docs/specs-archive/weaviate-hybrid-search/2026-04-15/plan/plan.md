# Weaviate Hybrid Search План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи.
Outputs: plan, data model и contracts при необходимости.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Добавить реализацию HybridSearcher и HybridSearcherWithFilters интерфейсов в WeaviateStore через GraphQL API с BM25 + semantic fusion. Работа сосредоточена в internal/infrastructure/vectorstore/weaviate.go с использованием существующего HTTP клиента.

## Scope

- Реализация методов SearchHybrid, SearchHybridWithParentIDFilter, SearchHybridWithMetadataFilter в WeaviateStore
- Использование Weaviate GraphQL API для hybrid search с bm25 и nearVector
- Поддержка fusion-стратегий (RRF, weighted) через HybridConfig
- Валидация HybridConfig и обработка ошибок GraphQL API
- Unit-тесты для новых методов в weaviate_test.go
- Явно остаётся нетронутым: существующие методы VectorStore и VectorStoreWithFilters, другие vectorstores

## Implementation Surfaces

- internal/infrastructure/vectorstore/weaviate.go — существующая поверхность, добавляются методы SearchHybrid, SearchHybridWithParentIDFilter, SearchHybridWithMetadataFilter и compile-time assertions для HybridSearcher и HybridSearcherWithFilters
- internal/infrastructure/vectorstore/weaviate_test.go — существующая поверхность, добавляются unit-тесты для новых методов
- internal/domain/interfaces.go — уже содержит определения HybridSearcher и HybridSearcherWithFilters (не меняется)
- internal/domain/models.go — уже содержит HybridConfig и методы валидации (не меняется)

## Влияние на архитектуру

- Локальное влияние на WeaviateStore: добавление трёх методов для гибридного поиска
- Нет влияния на интеграции или границы между частями системы — интерфейсы уже определены в domain слое
- Нет migration или compatibility последствий — это новая capability, не меняющая существующий API

## Acceptance Approach

- AC-001 -> добавить compile-time assertion var _ domain.HybridSearcher = (*WeaviateStore)(nil) в weaviate.go, наблюдается через успешную компиляцию
- AC-002 -> реализовать GraphQL запрос с полями bm25 и nearVector в SearchHybrid, наблюдается через код weaviate.go и unit-тесты
- AC-003 -> реализовать fusion поле в GraphQL запросе с типом rrf или weighted в зависимости от HybridConfig.UseRRF, наблюдается через код weaviate.go и unit-тесты
- AC-004 -> добавить compile-time assertion var _ domain.HybridSearcherWithFilters = (*WeaviateStore)(nil) и реализовать методы SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией в GraphQL запросе, наблюдается через код weaviate.go и unit-тесты
- AC-005 -> вызвать config.Validate() в начале SearchHybrid, наблюдается через код weaviate.go и unit-тесты с невалидной конфигурацией
- AC-006 -> обрабатывать ошибки GraphQL клиента и HTTP ошибки, возвращать информативные ошибки, наблюдается через код weaviate.go и unit-тесты с ошибками API

## Данные и контракты

- Ссылка на AC-001, AC-004: compile-time assertions для интерфейсов
- Ссылка на AC-002, AC-003: GraphQL запрос с bm25, nearVector и fusion
- Ссылка на AC-005: валидация HybridConfig через существующий domain.HybridConfig.Validate()
- Ссылка на AC-006: обработка ошибок GraphQL и HTTP
- Изменения data model не требуются — HybridConfig и связанные типы уже определены в domain/models.go
- API contracts не меняются — это добавление новых методов в существующую реализацию, не меняющая публичный интерфейс пакета

## Стратегия реализации

- DEC-001 Использование GraphQL API вместо REST для hybrid search
  Why: Weaviate поддерживает hybrid search только через GraphQL API с bm25 и nearVector в одном запросе, REST API не предоставляет аналогичной возможности
  Tradeoff: требует формирования GraphQL запросов вручную вместо JSON, но это единственный способ реализовать hybrid search в Weaviate
  Affects: internal/infrastructure/vectorstore/weaviate.go
  Validation: unit-тесты проверяют структуру GraphQL запроса и ответа

- DEC-002 Повторное использование существующего HybridConfig из domain
  Why: HybridConfig уже определён в domain/models.go с валидацией, не нужно дублировать логику
  Tradeoff: нет tradeoff, это повторное использование существующего domain типа
  Affects: internal/infrastructure/vectorstore/weaviate.go (использует domain.HybridConfig)
  Validation: unit-тесты проверяют валидацию через config.Validate()

- DEC-003 Аналогичная структура реализации как в QdrantStore
  Why: QdrantStore уже реализует HybridSearcher с аналогичной логикой, повторение паттерна упрощает review и поддержку
  Tradeoff: нет tradeoff, это следование существующему паттерну
  Affects: internal/infrastructure/vectorstore/weaviate.go
  Validation: unit-тесты проверяют аналогичное поведение

## Incremental Delivery

### MVP (Первая ценность)

- Реализация SearchHybrid с GraphQL API, BM25, nearVector и fusion
- Compile-time assertion для HybridSearcher
- Unit-тесты для SearchHybrid
- Критерий готовности MVP: AC-001, AC-002, AC-003, AC-005, AC-006 покрыты

### Итеративное расширение

- Реализация SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией
- Compile-time assertion для HybridSearcherWithFilters
- Unit-тесты для методов с фильтрацией
- Критерий готовности: AC-004 покрыта

## Порядок реализации

- Сначала добавить compile-time assertion для HybridSearcher (T1.1)
- Затем реализовать SearchHybrid метод (T2.1)
- Добавить unit-тесты для SearchHybrid (T2.2)
- Добавить compile-time assertion для HybridSearcherWithFilters и реализовать методы с фильтрацией (T3.1)
- Добавить unit-тесты для методов с фильтрацией (T4.1)

## Риски

- Риск 1: GraphQL API Weaviate может отличаться от документации или требовать специфической структуры запроса
  Mitigation: использовать unit-тесты с mock сервером для проверки структуры запроса и обработки ответа

- Риск 2: Fusion-стратегии в Weaviate могут отличаться от ожидаемых (например, параметры RRF)
  Mitigation: проверить документацию Weaviate для fusion параметров и добавить fallback в код

## Rollout и compatibility

- Специальных rollout-действий не требуется — это новая capability в существующей реализации
- Нет migration или compatibility последствий
- Нет feature flags — изменение добавляется сразу

## Проверка

- Добавить unit-тесты для SearchHybrid с mock GraphQL сервером (проверка bm25, nearVector, fusion)
- Добавить unit-тесты для валидации HybridConfig (проверка ошибок при невалидных значениях)
- Добавить unit-тесты для обработки ошибок GraphQL API (проверка HTTP ошибок и ошибок парсинга)
- Добавить unit-тесты для SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter (проверка фильтрации)
- Подтверждает AC-001, AC-002, AC-003, AC-004, AC-005, AC-006 и DEC-001, DEC-002, DEC-003

## Соответствие конституции

- Интерфейсная абстракция: реализация следует существующим интерфейсам HybridSearcher и HybridSearcherWithFilters из domain/interfaces.go (CONST-001)
- Чистая архитектура: работа в infrastructure слое, не меняет domain или application слои (CONST-002)
- Контекстная безопасность: все методы принимают context.Context как первый параметр (CONST-003)
- Тестируемость: добавляются unit-тесты с mock сервером для GraphQL API (CONST-004)
- Язык реализации: Go 1.23+, соответствует ограничению (CONST-005)
- Язык документации и комментариев: русский, соответствует языковой политике (CONST-006)
