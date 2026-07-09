# Контрактные тесты VectorStore

## Scope Snapshot

- In scope: parameterized contract test suite, который гарантирует одинаковое поведение всех VectorStore-реализаций (memory, pgvector, qdrant, chromadb, weaviate, milvus) для каждого метода core-интерфейсов.
- Out of scope: интеграционные тесты, требующие running backend (pgvector, qdrant, chromadb, weaviate, milvus); тестирование HybridSearcher, CollectionManager, TransactionalDocumentStore capability-интерфейсов; рефакторинг существующих per-store тестов.

## Цель

Разработчик, добавляющий новую VectorStore-реализацию, сейчас пишет тесты с нуля, копируя паттерны из существующих. После внедрения contract suite достаточно зарегистрировать store в suite и получить гарантию, что Upsert/Delete/Search/SearchWithFilter/SearchWithMetadataFilter работают идентично другим реализациям. Успех измеряется: contract suite покрывает ≥20 сценариев, все 6 store реализаций могут быть зарегистрированы (memory — сразу, остальные — через HTTP mock).

## Основной сценарий

1. Стартовая точка: каждая VectorStore-реализация имеет собственные тесты с разными паттернами и неполным покрытием edge cases.
2. Добавляется `contract_test.go` с Suite-структурой, параметризованной `StoreFactory func() VectorStore`.
3. Каждый store-пакет регистрирует Suite через TestMain или TestStore* функцию.
4. Результат: единый прогон `go test ./internal/infrastructure/vectorstore/ -run TestContract` покрывает ≥20 контрактных сценариев для зарегистрированных реализаций.

## User Stories

- P1 (MVP): contract suite для `VectorStore` (Upsert, Delete, Search) + `VectorStoreWithFilters` (SearchWithFilter, SearchWithMetadataFilter). Покрытие: 6 реализаций × 15 сценариев = 90 тестов.
- P2: contract suite для capability-интерфейсов `HybridSearcher`, `DocumentStore`, `CollectionManager`.

## MVP Slice

| Интерфейс | Методы | Число сценариев |
|-----------|--------|-----------------|
| VectorStore | Upsert, Delete, Search | 8 |
| VectorStoreWithFilters | SearchWithFilter, SearchWithMetadataFilter | 7 |
| **Итого** | | **15 сценариев × 6 store** |

## First Deployable Outcome

- `internal/infrastructure/vectorstore/contract_test.go` — Suite + parameterized tests.
- MemoryStore регистрируется в suite и все 15 сценариев проходят.
- `go test ./internal/infrastructure/vectorstore/ -run TestContract_/memory` — PASS.

## Scope

- `internal/infrastructure/vectorstore/contract_test.go` — новый файл: Suite, contract тесты, StoreFactory.
- `internal/infrastructure/vectorstore/memory_contract_test.go` — регистрация InMemoryStore.
- Каждый store может опционально добавить `*_contract_test.go` для регистрации через HTTP mock.

## Контекст

- Все 6 store имеют compile-time assertions на `VectorStore` и `VectorStoreWithFilters`.
- InMemoryStore — эталон: не требует внешних зависимостей, подходит как reference implementation.
- MemoryStore используется как test double в `internal/application/` — контрактные тесты гарантируют, что memory ведёт себя как production store.
- HTTP-based store (pgvector/qdrant/chromadb/weaviate/milvus) могут регистрироваться через `httptest.NewServer`.

## Требования

- RQ-001 Suite ДОЛЖЕН определять `StoreFactory func() VectorStore` как точку расширения.
- RQ-002 Каждый contract test ДОЛЖЕН принимать `*Suite` и вызывать `suite.Store()` для получения экземпляра.
- RQ-003 Suite ДОЛЖЕН тестировать все методы core `VectorStore` (Upsert, Delete, Search).
- RQ-004 Suite ДОЛЖЕН тестировать все методы `VectorStoreWithFilters` (SearchWithFilter, SearchWithMetadataFilter).
- RQ-005 Suite ДОЛЖЕН покрывать edge cases: пустая коллекция, nil embedding, невалидный topK, удаление несуществующего ID, дублирующий Upsert, embedding dimension mismatch.

## Вне scope

- Тестирование HybridSearcher, DocumentStore, TransactionalDocumentStore, CollectionManager (P2).
- Модификация существующих per-store тестовых файлов.
- Интеграционные тесты с реальными backend (docker-compose).
- Добавление новых VectorStore-реализаций.

## Критерии приемки

### AC-001 Contract suite для VectorStore (Upsert + Delete + Search)

- Почему это важно: базовый контракт, без которого RAG pipeline не работает.
- **Given** Suite с MemoryStore
- **When** вызываются все contract-тесты VectorStore
- **Then** все тесты PASS
- Evidence: `go test ./internal/infrastructure/vectorstore/ -run TestContract_VectorStore -count=1`

### AC-002 Contract suite для VectorStoreWithFilters (SearchWithFilter + SearchWithMetadataFilter)

- Почему это важно: фильтрация — ключевая capability для production RAG.
- **Given** Suite с MemoryStore
- **When** вызываются все contract-тесты VectorStoreWithFilters
- **Then** все тесты PASS
- Evidence: `go test ./internal/infrastructure/vectorstore/ -run TestContract_VectorStoreWithFilters -count=1`

### AC-003 MemoryStore проходит все сценарии suite

- Почему это важно: MemoryStore — reference implementation и test double для application-тестов.
- **Given** Suite зарегистрирован с MemoryStore
- **When** прогоняются все contract-тесты
- **Then** ≥15 сценариев PASS
- Evidence: `go test ./internal/infrastructure/vectorstore/ -run TestContract_/memory -v -count=1`

### AC-004 Suite расширяем для HTTP-based store

- Почему это важно: production store должны верифицироваться тем же suite.
- **Given** Suite-регистрация с StoreFactory, создающей QdrantStore с `httptest.NewServer`
- **When** прогоняются все contract-тесты
- **Then** тесты проходят (сервер обрабатывает запросы корректно)
- Evidence: prototype-регистрация QdrantStore через mock HTTP в `qdrant_contract_test.go`

### AC-005 go vet + golangci-lint без errors

- Почему это важно: код должен быть идиоматичным.
- **Given** все изменения завершены
- **When** `go vet ./internal/infrastructure/vectorstore/... && golangci-lint run ./internal/infrastructure/vectorstore/...`
- **Then** exit code 0

## Допущения

- StoreFactory создаёт "чистое" хранилище (пустая коллекция) на каждый вызов.
- InMemoryStore — reference implementation; расхождение поведения других store с MemoryStore = bug в store, не в suite.
- Тесты не используют t.Parallel (каждый тест создаёт свой store через StoreFactory).

## Критерии успеха

- SC-001 Contract suite покрывает ≥15 сценариев для core + filters.
- SC-002 Все 6 store (memory, pgvector, qdrant, chromadb, weaviate, milvus) регистрируются и проходят suite (memory — реально, остальные — через HTTP mock).

## Краевые случаи

- Пустая коллекция: Search возвращает пустой RetrievalResult, не ошибку.
- nil/пустой embedding: зависит от store — одни возвращают ошибку, другие игнорируют. Suite тестирует поведение MemoryStore как эталон.
- topK = 0 или отрицательный: возврат ErrInvalidQueryTopK или пустой результат.
- Delete несуществующего ID: должен быть idempotent (не ошибка).
- Upsert с существующим ID: перезаписывает, не дублирует.
- Upsert с dimension mismatch: ошибка валидации.

## Открытые вопросы

- Нужно ли тестировать Concurrent access (race) в contract suite? Да, но в P2.
- StoreFactory должна создавать store с предустановленными данными (seed) или пустой? Пустой — каждый тест сам наполняет.
