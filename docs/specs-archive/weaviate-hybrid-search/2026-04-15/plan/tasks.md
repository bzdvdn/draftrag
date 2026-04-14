# Weaviate Hybrid Search Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для этой фичи.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/vectorstore/weaviate.go | T1.1, T2.1, T3.1 |
| internal/infrastructure/vectorstore/weaviate_test.go | T2.2, T4.1 |

## Фаза 1: Основа

Цель: добавить compile-time assertion для интерфейсов HybridSearcher и HybridSearcherWithFilters.

- [x] T1.1 Добавить compile-time assertion для HybridSearcher интерфейса — WeaviateStore реализует HybridSearcher (AC-001, DEC-003). Touches: internal/infrastructure/vectorstore/weaviate.go

## Фаза 2: Основная реализация

Цель: реализовать метод SearchHybrid с GraphQL API, BM25, nearVector и fusion.

- [x] T2.1 Реализовать SearchHybrid метод с GraphQL API — метод использует bm25, nearVector и fusion (AC-001, AC-002, AC-003, AC-005, AC-006, DEC-001, DEC-002). Touches: internal/infrastructure/vectorstore/weaviate.go

## Фаза 3: Проверка основной реализации

Цель: добавить unit-тесты для SearchHybrid.

- [x] T3.1 Добавить unit-тесты для SearchHybrid — тесты покрывают GraphQL запрос, валидацию HybridConfig и обработку ошибок (AC-002, AC-003, AC-005, AC-006). Touches: internal/infrastructure/vectorstore/weaviate_test.go

## Фаза 4: Реализация с фильтрацией

Цель: добавить compile-time assertion для HybridSearcherWithFilters и реализовать методы с фильтрацией.

- [x] T4.1 Реализовать SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter — методы выполняют hybrid search с фильтрацией в GraphQL запросе (AC-004, DEC-003). Touches: internal/infrastructure/vectorstore/weaviate.go

## Фаза 5: Проверка с фильтрацией

Цель: добавить unit-тесты для методов с фильтрацией.

- [x] T5.1 Добавить unit-тесты для методов с фильтрацией — тесты покрывают фильтрацию по ParentID и метаданным (AC-004). Touches: internal/infrastructure/vectorstore/weaviate_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1
- AC-002 -> T2.1, T3.1
- AC-003 -> T2.1, T3.1
- AC-004 -> T4.1, T5.1
- AC-005 -> T2.1, T3.1
- AC-006 -> T2.1, T3.1

## Заметки

- Порядок задач соответствует плану реализации
- Все задачи ссылаются на стабильные ID (AC-*, DEC-*)
- Surface Map обеспечивает batch-чтение файлов для implement-агента
