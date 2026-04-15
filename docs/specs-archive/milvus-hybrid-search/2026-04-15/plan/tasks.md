# Milvus Hybrid Search Задачи

## Phase Contract

Inputs: plan и минимальные supporting артефакты для этой фичи.
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: задачи получаются расплывчатыми или coverage не удается сопоставить.

## Surface Map

| Surface | Tasks |
|---------|-------|
| internal/infrastructure/vectorstore/milvus.go | T1.1, T2.1, T2.2, T3.1, T3.2 |
| internal/infrastructure/vectorstore/milvus_test.go | T4.1, T4.2, T4.3, T4.4 |

## Фаза 1: Основа

Цель: добавить compile-time assertions для интерфейсов HybridSearcher и HybridSearcherWithFilters.

- [x] T1.1 Добавить compile-time assertion для HybridSearcher интерфейса — MilvusStore реализует HybridSearcher (AC-001, DEC-001). Touches: internal/infrastructure/vectorstore/milvus.go
- [x] T1.2 Добавить compile-time assertion для HybridSearcherWithFilters интерфейса — MilvusStore реализует HybridSearcherWithFilters (AC-004, DEC-001). Touches: internal/infrastructure/vectorstore/milvus.go

## Фаза 2: Основная реализация

Цель: реализовать SearchHybrid метод с Multi-Vector Search API, валидацией и обработкой ошибок.

- [x] T2.1 Реализовать SearchHybrid метод с Multi-Vector Search API — метод использует AnnSearchRequest для BM25 и dense векторов, валидирует HybridConfig, вызывает hybrid_search() через POST /v2/vectordb/entities/hybrid_search (AC-001, AC-002, AC-003, AC-005, AC-006, DEC-001, DEC-002). Touches: internal/infrastructure/vectorstore/milvus.go
- [x] T2.2 Реализовать парсинг ответа hybrid_search() в domain.RetrievalResult — метод извлекает чанки и fusion score из ответа Milvus (AC-002, AC-006). Touches: internal/infrastructure/vectorstore/milvus.go

## Фаза 3: Реализация с фильтрацией

Цель: реализовать методы с фильтрацией через expr в AnnSearchRequest.

- [x] T3.1 Реализовать SearchHybridWithParentIDFilter — метод выполняет hybrid search с фильтрацией по parentId через expr (AC-004, DEC-003). Touches: internal/infrastructure/vectorstore/milvus.go
- [x] T3.2 Реализовать SearchHybridWithMetadataFilter — метод выполняет hybrid search с фильтрацией по metadata через expr (AC-004, DEC-003). Touches: internal/infrastructure/vectorstore/milvus.go

## Фаза 4: Проверка

Цель: добавить unit-тесты для всех новых методов.

- [x] T4.1 Добавить unit-тесты для SearchHybrid — тесты покрывают RRF fusion, weighted fusion, валидацию HybridConfig и обработку ошибок (AC-002, AC-003, AC-005, AC-006). Touches: internal/infrastructure/vectorstore/milvus_test.go
- [x] T4.2 Добавить unit-тест для SearchHybrid с пустыми результатами — тест проверяет поведение при пустой коллекции (AC-002). Touches: internal/infrastructure/vectorstore/milvus_test.go
- [x] T4.3 Добавить unit-тесты для SearchHybridWithParentIDFilter — тесты покрывают фильтрацию по parentId и делегирование при пустом фильтре (AC-004). Touches: internal/infrastructure/vectorstore/milvus_test.go
- [x] T4.4 Добавить unit-тесты для SearchHybridWithMetadataFilter — тесты покрывают фильтрацию по metadata и делегирование при пустом фильтре (AC-004). Touches: internal/infrastructure/vectorstore/milvus_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1
- AC-002 -> T2.1, T2.2, T4.1, T4.2
- AC-003 -> T2.1, T4.1
- AC-004 -> T1.2, T3.1, T3.2, T4.3, T4.4
- AC-005 -> T2.1, T4.1
- AC-006 -> T2.1, T2.2, T4.1

## Заметки

- Сохраняйте порядок задач согласованным с планом и переносите работу в поздние фазы только если она реально зависит от ранних
- Используйте phase-scoped task IDs в формате `T<phase>.<index>`
- Делайте каждую задачу конкретной, измеримой и исполнимой как один связный кусок работы
- Предпочитайте action verbs, связанные с наблюдаемым результатом: implement, add, migrate, validate, remove, backfill
- По возможности ссылайтесь в тексте задач на 1-2 стабильных ID (`AC-*`, `RQ-*`, `DEC-*`)
- Не прячьте proof внутри большой implementation-задачи, а выносите validation отдельно
- Отмечайте задачи выполненными по мере реализации и не оставляйте критерии приемки без покрытия задачами
- Явно укажите, если какая-то фаза осознанно пропущена, потому что фиче она не нужна
