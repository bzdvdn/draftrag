# Fuzz и Property Tests План

## Phase Contract

Inputs: spec.md, domain/models.go, pkg/draftrag/search.go, pkg/draftrag/search_builder_test.go
Outputs: plan.md, data-model.md
Stop if: нет — spec детальна.

## MVP Slice

- `internal/domain/fuzz_test.go` — FuzzValidateDocument, FuzzValidateChunk, FuzzValidateQuery
- `pkg/draftrag/fuzz_test.go` — FuzzSearchBuilderValidate
- `pkg/draftrag/roundtrip_test.go` — TestVectorStoreRoundtrip (property-based)
- AC-001, AC-002, AC-003, AC-004, AC-005

## First Validation Path

`go test -fuzz=FuzzValidateDocument -fuzztime=15s ./internal/domain/` и `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=15s ./pkg/draftrag/` — 0 panics.

## Scope

- `internal/domain/fuzz_test.go` — 3 fuzz-функции для domain-валидации
- `pkg/draftrag/fuzz_test.go` — 1 fuzz-функция для SearchBuilder
- `pkg/draftrag/roundtrip_test.go` — 1 property-тест VectorStore roundtrip
- Никаких изменений в production-коде

## Implementation Surfaces

| Surface | Почему участвует | Новая/сущ. |
|---------|-----------------|------------|
| `internal/domain/fuzz_test.go` | FuzzValidateDocument, FuzzValidateChunk, FuzzValidateQuery | Новая |
| `pkg/draftrag/fuzz_test.go` | FuzzSearchBuilderValidate | Новая |
| `pkg/draftrag/roundtrip_test.go` | TestVectorStoreRoundtrip — roundtrip property | Новая |
| `internal/domain/models.go` | Document.Validate, Chunk.Validate, Query.Validate — цели fuzzing | Сущ., без изменений |
| `pkg/draftrag/search.go` | SearchBuilder.validate — цель fuzzing | Сущ., без изменений |
| `internal/infrastructure/vectorstore/memory.go` | InMemoryStore — reference для roundtrip | Сущ., без изменений |

## Bootstrapping Surfaces

`none` — все нужные структуры в репозитории уже есть.

## Влияние на архитектуру

- Локальное: только test-файлы, production-код не меняется
- На интеграции: не влияет
- Migration/compatibility: не требуется

## Acceptance Approach

| AC | Подход | Surfaces | Наблюдение |
|----|--------|----------|------------|
| AC-001 | 3 fuzz-функции в domain, fuzztime=15s без паники | `internal/domain/fuzz_test.go` | `go test -fuzz=FuzzValidate -fuzztime=15s ./internal/domain/` |
| AC-002 | 1 fuzz-функция для SearchBuilder.validate, fuzztime=15s без паники | `pkg/draftrag/fuzz_test.go` | `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=15s ./pkg/draftrag/` |
| AC-003 | Roundtrip: random chunk → Upsert → Search → ID совпадает | `pkg/draftrag/roundtrip_test.go` | `go test -run TestVectorStoreRoundtrip -count=100` PASS |
| AC-004 | Seed корпуса через f.Add() для базовых edge cases | `internal/domain/fuzz_test.go`, `pkg/draftrag/fuzz_test.go` | `go test -run TestFuzzSeedCorpora` PASS |
| AC-005 | `go vet` exit 0 | — | `go vet ./internal/domain/ ./pkg/draftrag/` |

## Данные и контракты

- Data model не меняется — см. `data-model.md` (stub: no-change)
- API/event контракты не меняются

## Стратегия реализации

### DEC-001 Go native fuzzing (`testing.F`) без external библиотек

Why: Go 1.23+ имеет встроенный fuzzer, zero dependencies. Соответствует текущему стилю тестов.
Tradeoff: ограниченная генерация структур (нужна ручная парсинг из []byte или примитивных типов).
Affects: `internal/domain/fuzz_test.go`, `pkg/draftrag/fuzz_test.go`
Validation: `go test -fuzz=FuzzValidateDocument -fuzztime=5s` работает.

### DEC-002 Fuzz-функции принимают примитивные типы, парсят структуры вручную

Why: Go fuzzer поддерживает только примитивные типы ([]byte, string, int, bool, float64). Для сложных структур (Document, Chunk) передаём string/int и конструируем объект внутри fuzz-функции.
Tradeoff: ручной парсинг, но это стандартная практика для Go fuzzing.
Affects: `internal/domain/fuzz_test.go` — FuzzValidateDocument берёт 2 string (ID, Content), FuzzValidateChunk — 3 string (ID, Content, ParentID), FuzzValidateQuery — string + int.
Validation: fuzz-функции вызывают Validate без паники.

### DEC-003 Property roundtrip как regular test с `testing/quick` или rand-генерацией

Why: roundtrip не требует fuzzer-а — достаточно 100+ случайных итераций внутри одного test. Проще, быстрее, детерминированнее.
Tradeoff: не использует fuzzer corpus, но roundtrip-инвариант простой и не требует умного поиска seed-ов.
Affects: `pkg/draftrag/roundtrip_test.go`
Validation: `go test -run TestVectorStoreRoundtrip -count=100` — 0 failures.

### DEC-004 Seed corpora через `f.Add()` в `init()` или в `FuzzXxx` первой строкой

Why: Go fuzzer использует seed-значения как базовые точки поиска. Добавляем пустые строки, unicode, null byte, очень длинные строки.
Tradeoff: небольшой boilerplate, но обязателен для покрытия известных edge cases.
Affects: `internal/domain/fuzz_test.go`, `pkg/draftrag/fuzz_test.go`
Validation: seed-корпуса проходят без ошибок.

## Incremental Delivery

### MVP (Первая ценность)

1. `internal/domain/fuzz_test.go` — 3 fuzz-функции с seed corpora
2. `pkg/draftrag/fuzz_test.go` — FuzzSearchBuilderValidate с seed corpora
3. `pkg/draftrag/roundtrip_test.go` — TestVectorStoreRoundtrip
4. `go vet` + fuzztime=15s прогоны

## Порядок реализации

1. domain fuzz: FuzzValidateDocument, FuzzValidateChunk, FuzzValidateQuery
2. SearchBuilder fuzz: FuzzSearchBuilderValidate
3. Property roundtrip: TestVectorStoreRoundtrip
4. Итоговый прогон + vet

Параллелить: шаги 1-3 независимы.

## Риски

- **Риск 1**: Fuzz-тесты на CI могут быть медленными (fuzztime=15s × 4 = 1 минута).
  Mitigation: fuzz-тесты запускаются отдельно от regular тестов; на CI достаточно `-fuzztime=5s`.
- **Риск 2**: SearchBuilder fuzz может найти реальный баг (например, panic на null bytes).
  Mitigation: это цель fuzzing, а не проблема. Баги фиксируются по мере обнаружения.

## Rollout и compatibility

Специальных rollout-действий не требуется. Новые файлы только в test package.

## Проверка

| Шаг | Check | AC |
|-----|-------|----|
| Build | `go build ./internal/domain/ ./pkg/draftrag/` | — |
| Domain fuzz short | `go test -fuzz=FuzzValidateDocument -fuzztime=5s ./internal/domain/` | AC-001 |
| SearchBuilder fuzz short | `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=5s ./pkg/draftrag/` | AC-002 |
| Roundtrip | `go test -run TestVectorStoreRoundtrip -count=100` | AC-003 |
| Seed corpora | `go test -run TestFuzzSeedCorpora` PASS | AC-004 |
| Vet | `go vet ./internal/domain/ ./pkg/draftrag/` | AC-005 |

## Соответствие конституции

Нет конфликтов.
