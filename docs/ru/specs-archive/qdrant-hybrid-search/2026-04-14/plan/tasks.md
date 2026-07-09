# Qdrant Hybrid Search Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для этой фичи
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/vectorstore/qdrant.go | T1.1, T2.1, T3.1 |
| internal/infrastructure/vectorstore/qdrant_test.go | T2.2, T4.1 |

## Фаза 1: Основа

Цель: подготовить compile-time assertions для новых интерфейсов.

- [x] T1.1 Добавить compile-time assertion для HybridSearcher — QdrantStore реализует интерфейс. Touches: internal/infrastructure/vectorstore/qdrant.go — AC-001, DEC-001

## Фаза 2: Основная реализация

Цель: реализовать SearchHybrid с Query API Prefetch и Fusion.RRF.

- [x] T2.1 Реализовать SearchHybrid метод — Query API с Prefetch и Fusion.RRF, валидация HybridConfig. Touches: internal/infrastructure/vectorstore/qdrant.go — AC-001, AC-002, AC-003, AC-005, AC-006, DEC-001, DEC-002
- [x] T2.2 Добавить unit-тесты для SearchHybrid — тесты для Query API Prefetch, Fusion.RRF, валидации и ошибок. Touches: internal/infrastructure/vectorstore/qdrant_test.go — AC-002, AC-003, AC-005, AC-006

## Фаза 3: Расширение

Цель: реализовать методы с фильтрацией.

- [x] T3.1 Добавить compile-time assertion для HybridSearcherWithFilters и реализовать методы — SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter с фильтрацией. Touches: internal/infrastructure/vectorstore/qdrant.go — AC-004

## Фаза 4: Проверка

Цель: добавить unit-тесты для методов с фильтрацией.

- [x] T4.1 Добавить unit-тесты для методов с фильтрацией — тесты для SearchHybridWithParentIDFilter и SearchHybridWithMetadataFilter. Touches: internal/infrastructure/vectorstore/qdrant_test.go — AC-004

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1
- AC-002 -> T2.1, T2.2
- AC-003 -> T2.1, T2.2
- AC-004 -> T3.1, T4.1
- AC-005 -> T2.1, T2.2
- AC-006 -> T2.1, T2.2

## Заметки

- Порядок задач соответствует плану: сначала compile-time assertions, затем MVP (SearchHybrid), затем расширение (методы с фильтрацией)
- Unit-тесты добавляются после реализации каждого метода
