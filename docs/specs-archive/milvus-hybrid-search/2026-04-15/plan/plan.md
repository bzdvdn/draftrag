# Milvus Hybrid Search План

## Phase Contract

Inputs: spec и минимальный контекст репозитория для этой фичи.
Outputs: plan, data model и contracts при необходимости.
Stop if: spec слишком расплывчата для безопасного планирования.

## Цель

Реализовать HybridSearcher и HybridSearcherWithFilters интерфейсы в MilvusStore через Milvus Multi-Vector Search API с AnnSearchRequest для BM25 (sparse) и dense векторов, поддерживая fusion-стратегии RRF и weighted через rerank strategy.

## Scope

- internal/infrastructure/vectorstore/milvus.go - добавление методов SearchHybrid, SearchHybridWithParentIDFilter, SearchHybridWithMetadataFilter
- internal/infrastructure/vectorstore/milvus_test.go - добавление unit-тестов для новых методов
- Явно остаётся нетронутым: существующие методы Upsert, Delete, Search, SearchWithFilter, SearchWithMetadataFilter

## Implementation Surfaces

- internal/infrastructure/vectorstore/milvus.go - существующая поверхность, расширение методами HybridSearcher
- Добавление compile-time assertions для HybridSearcher и HybridSearcherWithFilters
- Реализация SearchHybrid с использованием Milvus Multi-Vector Search API (POST /v2/vectordb/entities/hybrid_search)
- Реализация SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией через expr

## Влияние на архитектуру

- Локальное влияние: MilvusStore получает три новых метода без изменения существующих интерфейсов
- Нет влияния на интеграции или границы между частями системы
- Нет migration, compatibility или rollout-последствий (чистое расширение существующего store)

## Acceptance Approach

- AC-001 -> добавление compile-time assertion `var _ domain.HybridSearcher = (*MilvusStore)(nil)` в milvus.go
- AC-002 -> реализация SearchHybrid с созданием двух AnnSearchRequest (text_sparse для BM25, text_dense для semantic) и вызовом hybrid_search() через POST /v2/vectordb/entities/hybrid_search
- AC-003 -> реализация выбора rerank strategy в зависимости от HybridConfig.UseRRF (RRF или weighted) в теле запроса hybrid_search()
- AC-004 -> реализация SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с добавлением expr фильтра в AnnSearchRequest
- AC-005 -> вызов config.Validate() в начале SearchHybrid с возвратом ошибки при невалидной конфигурации
- AC-006 -> оборачивание ошибок от doRequest и JSON parsing в информативные error messages с описанием причины

## Данные и контракты

- Эта фича не вводит новых сущностей или изменений в data model (использует существующие domain.Chunk, domain.RetrievalResult, domain.HybridConfig)
- Эта фича не вводит новых API или event boundaries (расширение существующего domain.HybridSearcher интерфейса)
- Изменения ограничены internal/infrastructure/vectorstore/milvus.go и milvus_test.go

## Стратегия реализации

- DEC-001 Использование Milvus Multi-Vector Search API через REST (без SDK)
  Why: MilvusStore уже использует raw HTTP для всех операций (Upsert, Search, Delete), сохранение паттерна уменьшает сложность и не вводит новых зависимостей
  Tradeoff: отсутствие типобезопасности SDK против простоты и согласованности с существующим кодом
  Affects: internal/infrastructure/vectorstore/milvus.go (новый метод searchHybrid через doRequest)
  Validation: unit-тесты проверяют корректность формирования запроса и парсинга ответа

- DEC-002 Использование AnnSearchRequest для BM25 (sparse) и dense векторов
  Why: Milvus Multi-Vector Search требует создания AnnSearchRequest для каждого векторного поля; это канонический подход для hybrid search в Milvus
  Tradeoff: дополнительная сложность формирования запроса против гибкости и соответствия документации Milvus
  Affects: internal/infrastructure/vectorstore/milvus.go (SearchHybrid создаёт два AnnSearchRequest)
  Validation: unit-тесты проверяют, что AnnSearchRequest создаются для text_sparse и text_dense полей

- DEC-003 Выбор rerank strategy на основе HybridConfig.UseRRF
  Why: Milvus поддерживает RRF и weighted fusion через rerank strategy; HybridConfig.UseRRF определяет стратегию
  Tradeoff: ограничение двумя стратегиями против простоты и соответствия существующей HybridConfig
  Affects: internal/infrastructure/vectorstore/milvus.go (SearchHybrid передаёт rerank strategy в запрос)
  Validation: unit-тесты проверяют RRF и weighted fusion с разными HybridConfig

- DEC-004 Фильтрация через expr в AnnSearchRequest
  Why: Milvus поддерживает фильтрацию через expr параметр в AnnSearchRequest; это канонический подход
  Tradeoff: строковые выражения вместо типобезопасного builder против простоты и соответствия существующему паттерну фильтрации в SearchWithFilter
  Affects: internal/infrastructure/vectorstore/milvus.go (SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter добавляют expr)
  Validation: unit-тесты проверяют фильтрацию по parentId и metadata через expr

## Incremental Delivery

### MVP (Первая ценность)

- Реализация SearchHybrid с BM25 + dense fusion через RRF
- Критерий готовности MVP: AC-001, AC-002, AC-003, AC-005, AC-006 покрыты unit-тестами

### Итеративное расширение

- Реализация SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией
- Критерий готовности итерации: AC-004 покрыт unit-тестами

## Порядок реализации

- Добавление compile-time assertions для HybridSearcher и HybridSearcherWithFilters
- Реализация SearchHybrid с валидацией HybridConfig, созданием AnnSearchRequest и вызовом hybrid_search()
- Реализация парсинга ответа hybrid_search() в domain.RetrievalResult
- Реализация SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией через expr
- Добавление unit-тестов для всех новых методов

## Риски

- Риск 1: Milvus API для hybrid_search() может отличаться от документации
  Mitigation: unit-тесты с mock сервером проверяют корректность запроса и ответа; при необходимости адаптируем формат запроса

- Риск 2: Формирование expr для фильтрации может быть сложным для metadata с nested структурами
  Mitigation: ограничиваемся flat metadata (map[string]string) как в существующем SearchWithMetadataFilter

## Rollout и compatibility

- Специальных rollout-действий не требуется (чистое расширение существующего store)
- Нет backward compatibility concerns (новые методы не меняют существующее поведение)
- Нет monitoring или operational follow-up (изменения только в library коде)

## Проверка

- Unit-тесты для SearchHybrid с RRF fusion (AC-002, AC-003)
- Unit-тесты для SearchHybrid с weighted fusion (AC-003)
- Unit-тесты для валидации HybridConfig (AC-005)
- Unit-тесты для обработки ошибок Milvus API (AC-006)
- Unit-тесты для SearchHybridWithParentIDFilter (AC-004)
- Unit-тесты для SearchHybridWithMetadataFilter (AC-004)
- Compile-time assertion для HybridSearcher (AC-001)
- Compile-time assertion для HybridSearcherWithFilters (AC-004)

## Соответствие конституции

- [CONST-LANG] Язык документации: русский (соответствует конституции)
- [CONST-ARCH] Clean Architecture: implementation в infrastructure слое, использование domain интерфейсов (соответствует конституции)
- [CONST-GO] Только Go 1.21+, нет bindings для других языков (соответствует конституции)
- [CONST-NO-SERVER] Нет встроенного HTTP-сервера или CLI (соответствует конституции)
- [CONST-INTERFACES] Все внешние зависимости через Go-интерфейсы (соответствует конституции, используем domain.HybridSearcher)
- [CONST-CONTEXT] Контекст (context.Context) во всех публичных операциях (соответствует конституции)
- [CONST-TESTS] Unit-тесты для всех новых функций (соответствует конституции)
