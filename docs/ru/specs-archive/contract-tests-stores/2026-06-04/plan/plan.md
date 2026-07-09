# Контрактные тесты VectorStore — План

## Phase Contract

Inputs: spec.md, domain/interfaces.go, domain/models.go, vectorstore/memory.go, vectorstore/memory_test.go
Outputs: plan.md, data-model.md
Stop if: нет — spec достаточно детальна.

## MVP Slice

- `contract_test.go` — Suite + 15 parameterized contract-тестов для `VectorStore` (8) + `VectorStoreWithFilters` (7)
- `memory_contract_test.go` — регистрация `InMemoryStore` через `StoreFactory`
- `qdrant_contract_test.go` — prototype HTTP-mock регистрации для QdrantStore (AC-004)
- AC-001, AC-002, AC-003, AC-004, AC-005

## First Validation Path

`go test ./internal/infrastructure/vectorstore/ -run TestContract -v -count=1` — 15+ сценариев PASS.

## Scope

- Новый файл `contract_test.go` с Suite и телами тестов
- Новый файл `memory_contract_test.go` для регистрации MemoryStore
- Новый файл `qdrant_contract_test.go` для AC-004 prototype
- Никаких изменений в существующих файлах store-реализаций
- P2 (HybridSearcher, DocumentStore, CollectionManager) — вне scope

## Implementation Surfaces

| Surface | Почему участвует | Новая/сущ. |
|---------|-----------------|------------|
| `internal/infrastructure/vectorstore/contract_test.go` | Suite + parameterized contract body | Новая |
| `internal/infrastructure/vectorstore/memory_contract_test.go` | Регистрация InMemoryStore | Новая |
| `internal/infrastructure/vectorstore/qdrant_contract_test.go` | Prototype HTTP-mock регистрации (AC-004) | Новая |
| `internal/domain/interfaces.go` | VectorStore + VectorStoreWithFilters — источник контрактов | Сущ. |
| `internal/domain/models.go` | Chunk, RetrievalResult, MetadataFilter — типы данных | Сущ. |
| `internal/infrastructure/vectorstore/memory.go` | Reference implementation | Сущ., без изменений |
| `internal/infrastructure/vectorstore/qdrant.go` | HTTP store для mock-prototype | Сущ., без изменений |

## Bootstrapping Surfaces

`none` — все нужные структуры в репозитории уже есть.

## Влияние на архитектуру

- Локальное: только vectorstore-пакет, один новый test-файл
- На интеграции: не влияет
- Migration/compatibility: не требуется

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|----|--------|----------|------------|
| AC-001 | VectorStore contract тесты с `StoreFactory` → MemoryStore | `contract_test.go`, `memory_contract_test.go` | `go test -run TestContract_VectorStore` PASS |
| AC-002 | VectorStoreWithFilters contract тесты с `StoreFactory` → MemoryStore | `contract_test.go`, `memory_contract_test.go` | `go test -run TestContract_VectorStoreWithFilters` PASS |
| AC-003 | MemoryStore проходит все 15+ сценариев suite | `contract_test.go`, `memory_contract_test.go` | `go test -run TestContract_/memory -v` ≥15 PASS |
| AC-004 | QdrantStore с `httptest.NewServer` регистрируется и проходит | `qdrant_contract_test.go`, `contract_test.go` | `go test -run TestContract_/qdrant` PASS (mock) |
| AC-005 | `go vet` и `golangci-lint` без ошибок | Все | `go vet ./... && golangci-lint run` exit 0 |

## Данные и контракты

- Data model не меняется — см. `data-model.md` (stub: no-change)
- API/event контракты не меняются
- `StoreFactory func() domain.VectorStore` — единственный новый контракт, живёт только в test-файлах

## Стратегия реализации

### DEC-001 Функциональный Suite без testify/ginkgo

Why: стандартный `testing.T` + subtests — zero dependencies, соответствует текущему стилю тестов (memory_test.go), проще для понимания.
Tradeoff: немного более Verbose чем testify/suite, но без external dependency.
Affects: `contract_test.go`
Validation: 15 тестов через `t.Run` subtests.

### DEC-002 StoreFactory как `func() domain.VectorStore`

Why: минимальный контракт, легко замыкать store с предконфигурацией, без интерфейсов.
Tradeoff: каждый тест создаёт чистый store; фабрика не параметризована (нельзя передать конфиг).
Affects: `contract_test.go`, `*_contract_test.go`
Validation: MemoryStore фабрика = `func() domain.VectorStore { return NewInMemoryStore() }`.

### DEC-003 15 сценариев как отдельные функции, grouped в TestContract

Why: каждый сценарий — изолированный `func(StoreFactory)`, запускается через `t.Run` в двух группах: `VectorStore` и `VectorStoreWithFilters`. Легко читать, легко добавлять.
Tradeoff: 15 функций вместо табличного теста; табличный тест сложнее параметризовать для разных групп.
Affects: `contract_test.go`
Validation: `go test -run TestContract` = 15 subtests.

### DEC-004 HTTP mock для QdrantStore через `httptest.NewServer` + handler map

Why: минимальная имитация Qdrant API (`/collections/{name}/points`). Позволяет пройти contract-тесты без поднятия Docker.
Tradeoff: хрупкий — при изменении Qdrant API mock сломается.
Affects: `qdrant_contract_test.go`, `qdrant.go` не меняется.
Validation: `go test -run TestContract_/qdrant` — PASS.

## Incremental Delivery

### MVP (Первая ценность)

1. `contract_test.go` — Suite с 15 сценариями, `StoreFactory` тип
2. `memory_contract_test.go` — регистрация MemoryStore
3. `go test -run TestContract_/memory` — 15 PASS
4. `go vet ./internal/infrastructure/vectorstore/` — PASS

### Итеративное расширение

5. `qdrant_contract_test.go` — HTTP-mock для QdrantStore, проверка AC-004
6. `golangci-lint run ./internal/infrastructure/vectorstore/` — PASS

## Порядок реализации

1. `contract_test.go` — framework (StoreFactory, Suite-функции) — без тел тестов
2. Наполнение 8 сценариев VectorStore
3. Наполнение 7 сценариев VectorStoreWithFilters
4. `memory_contract_test.go` — регистрация + прогон
5. Валидация AC-001, AC-002, AC-003
6. `qdrant_contract_test.go` — HTTP mock + регистрация
7. Валидация AC-004
8. `go vet + golangci-lint` — AC-005
9. Итоговый прогон всех тестов

Параллелить: шаги 2 и 3 можно писать параллельно; шаг 6 независим от 4.

## Риски

- **Риск 1**: Qdrant HTTP mock не покрывает все эндпоинты, которые вызывает QdrantStore.
  Mitigation: mock только для методов, используемых в VectorStore (Upsert, Delete, Search — базовые CRUD точки). Если QdrantStore вызывает `PUT /collections`, mock обрабатывает и это.
- **Риск 2**: `golangci-lint` имеет pre-existing проблемы с pgx/otel (установлено ранее).
  Mitigation: AC-005 применяется только к vectorstore-пакету; pre-existing errors в других пакетах не блокируют.

## Rollout и compatibility

Специальных rollout-действий не требуется. Новые файлы только в test package.

## Проверка

| Шаг | Check | AC |
|-----|-------|----|
| Пакет собирается | `go build ./internal/infrastructure/vectorstore/` | — |
| VectorStore contract | `go test -run TestContract_VectorStore` PASS | AC-001 |
| Filter contract | `go test -run TestContract_VectorStoreWithFilters` PASS | AC-002 |
| MemoryStore полный прогон | `go test -run TestContract_/memory -v` ≥15 PASS | AC-003 |
| Qdrant mock | `go test -run TestContract_/qdrant` PASS | AC-004 |
| Линтер | `go vet ./internal/infrastructure/vectorstore/` + `golangci-lint run ./internal/infrastructure/vectorstore/` PASS | AC-005 |

## Соответствие конституции

Нет конфликтов.
