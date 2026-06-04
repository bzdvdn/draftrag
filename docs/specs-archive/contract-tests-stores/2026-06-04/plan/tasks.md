# Контрактные тесты VectorStore — Задачи

## Phase Contract

Inputs: plan.md, spec.md, domain/interfaces.go, domain/models.go
Outputs: tasks.md
Stop if: нет — plan детален, AC привязаны.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/vectorstore/contract_test.go` | T1.1, T2.1, T2.2, T3.1, T3.2 |
| `internal/infrastructure/vectorstore/memory_contract_test.go` | T3.1 |
| `internal/infrastructure/vectorstore/qdrant_contract_test.go` | T4.1 |

## Implementation Context

- **Цель MVP**: contract suite с 15 parameterized сценариями, MemoryStore проходит все, QdrantStore — через HTTP mock.
- **Инварианты/семантика**:
  - `StoreFactory func() domain.VectorStore` создаёт чистое (пустое) хранилище на каждый вызов
  - Тесты не используют t.Parallel (каждый тест получает свежий store)
  - Search на пустой коллекции → пустой RetrievalResult, не ошибка
  - Delete несуществующего ID → idempotent (nil error)
  - Upsert существующего ID → перезапись, не дублирование
  - topK ≤ 0 → ошибка (внутренняя или `ErrInvalidQueryTopK`)
- **Ошибки/коды**: `domain.ErrInvalidQueryTopK`, `domain.ErrEmbeddingDimensionMismatch`
- **Контракты/протокол**: `StoreFactory func() domain.VectorStore` — точка расширения для любого store
- **Границы scope**: не делаем HybridSearcher/DocumentStore/CollectionManager (P2), не трогаем существующие тесты store
- **Proof signals**: `go test -run TestContract -v -count=1` ≥15 PASS, `go vet` clean
- **References**: DEC-001 (функциональный Suite), DEC-002 (StoreFactory), DEC-003 (15 сценариев), DEC-004 (HTTP mock)

## Фаза 1: Основа

Цель: Suite-каркас с StoreFactory, типами сценариев и пустыми телами тестов.

- [x] T1.1 Создать `contract_test.go`: тип `StoreFactory`, константы сценариев, Suite-функции-заглушки для 8 VectorStore + 7 VectorStoreWithFilters сценариев, сгруппированные под `TestContract`. Touches: `internal/infrastructure/vectorstore/contract_test.go`

## Фаза 2: MVP VectorStore (Upsert, Delete, Search)

Цель: 8 contract-сценариев для базового VectorStore работают через MemoryStore.

- [x] T2.1 Реализовать 8 VectorStore contract-сценариев: Upsert+Search (базовый), Upsert+Search topK clipping, Search на пустой коллекции, Delete и Search, Delete несуществующего ID, Upsert дублирующего ID (перезапись), nil embedding в Search, topK=0/отрицательный. Touches: `internal/infrastructure/vectorstore/contract_test.go`
- [x] T2.2 Добавить `memory_contract_test.go`: регистрация MemoryStore через `StoreFactory`, привязка к `TestContract/memory`. `go test -run TestContract_VectorStore/memory` — 8 PASS. Touches: `internal/infrastructure/vectorstore/memory_contract_test.go`

## Фаза 3: MVP VectorStoreWithFilters (SearchWithFilter + SearchWithMetadataFilter)

Цель: 7 contract-сценариев для фильтров работают через MemoryStore.

- [x] T3.1 Реализовать 7 VectorStoreWithFilters contract-сценариев: SearchWithFilter по одному ParentID, SearchWithFilter по нескольким ParentID, SearchWithFilter пустой фильтр (делегат Search), SearchWithMetadataFilter точное совпадение, SearchWithMetadataFilter множественные поля, SearchWithMetadataFilter no match, SearchWithMetadataFilter пустой фильтр. Touches: `internal/infrastructure/vectorstore/contract_test.go`
- [x] T3.2 Полный прогон MemoryStore: `go test -run TestContract_/memory -v -count=1` — 15+ сценариев PASS. Touches: `internal/infrastructure/vectorstore/memory_contract_test.go`, `internal/infrastructure/vectorstore/contract_test.go`

## Фаза 4: HTTP mock prototype + проверка

Цель: QdrantStore через HTTP mock, финальная валидация AC-004 + AC-005.

- [x] T4.1 Создать `qdrant_contract_test.go`: HTTP mock сервер (`httptest.NewServer`) с минимальными хендлерами для Upsert/Delete/Search Qdrant API, регистрация QdrantStore через StoreFactory, `go test -run TestContract_/qdrant` PASS. Touches: `internal/infrastructure/vectorstore/qdrant_contract_test.go`
- [x] T4.2 Финальная проверка: `go vet ./internal/infrastructure/vectorstore/` — exit 0. Touches: `internal/infrastructure/vectorstore/`

## Покрытие критериев приемки

- AC-001 -> T2.1, T2.2
- AC-002 -> T3.1, T3.2
- AC-003 -> T2.2, T3.2
- AC-004 -> T4.1
- AC-005 -> T4.2

## Заметки

- Фаза 1 минимальна (только каркас) — основные сценарии в Фазах 2-3.
- T4.2 не требует изменений кода — только прогон инструментов.
- MemoryStore не меняется; contract-тесты пишутся против её текущего поведения.
