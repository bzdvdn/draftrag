# Fuzz и Property Tests — Задачи

## Phase Contract

Inputs: plan.md, spec.md, domain/models.go, pkg/draftrag/search.go
Outputs: tasks.md
Stop if: нет — plan детален, AC привязаны.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/fuzz_test.go` | T1.1, T2.1, T3.1 |
| `pkg/draftrag/fuzz_test.go` | T1.2, T2.1, T3.1 |
| `pkg/draftrag/roundtrip_test.go` | T2.2, T3.1 |

## Implementation Context

- **Цель MVP**: 4 fuzz-функции (3 domain + 1 SearchBuilder) + 1 property roundtrip-тест для VectorStore.
- **Инварианты/семантика**:
  - Validate-функции не должны паниковать ни при каких входных данных
  - Validate возвращает детерминированный результат (ошибка или nil) — не паникует, не зависает
  - Roundtrip: Upsert(chunk) → Search(chunk.Embedding) → результат содержит chunk с тем же ID
- **Ошибки/коды**: sentinel-ошибки из `domain` (`ErrEmptyDocumentID`, `ErrEmptyChunkID`, `ErrEmptyQueryText`, `ErrInvalidQueryTopK`) — fuzz проверяет только отсутствие panic
- **Контракты/протокол**: fuzz-функции принимают примитивные Go-типы (string, int, []byte) через `f.Fuzz`
- **Границы scope**: не трогаем production-код, не тестируем HTTP-клиенты
- **Proof signals**: `go test -fuzz=FuzzValidateDocument -fuzztime=15s` без краша, `go test -run TestVectorStoreRoundtrip -count=100` PASS
- **References**: DEC-001 (native fuzzing), DEC-002 (примитивные типы), DEC-003 (regular test для roundtrip), DEC-004 (seed corpora)

## Фаза 1: Domain fuzz-тесты

Цель: fuzz-покрытие для Document.Validate, Chunk.Validate, Query.Validate.

- [x] T1.1 Создать `internal/domain/fuzz_test.go` с тремя fuzz-функциями:
  - `FuzzValidateDocument(f *testing.F)` — принимает `id string, content string`, конструирует `Document{ID: id, Content: content}`, вызывает `Validate()`. Seed corpora: пустые строки, unicode, null bytes, очень длинные строки.
  - `FuzzValidateChunk(f *testing.F)` — принимает `id, content, parentID string`, конструирует `Chunk{ID: id, Content: content, ParentID: parentID}`, вызывает `Validate()`. Seed corpora: те же.
  - `FuzzValidateQuery(f *testing.F)` — принимает `text string, topK int`, конструирует `Query{Text: text, TopK: topK}`, вызывает `Validate()`. Seed corpora: пустая строка, topK=0, topK=MinInt/MaxInt.
  Touches: `internal/domain/fuzz_test.go`

## Фаза 2: SearchBuilder fuzz + roundtrip

Цель: fuzz для SearchBuilder.validate, property roundtrip для VectorStore.

- [x] T1.2 Создать `pkg/draftrag/fuzz_test.go` с `FuzzSearchBuilderValidate(f *testing.F)`:
  - Принимает `question string, topK int`.
  - Создаёт `SearchBuilder` через `NewSearchBuilder(p, question).TopK(topK).Build()`.
  - Вызывает validate().
  - Проверяет: нет panic (recover не нужен — Go fuzzer ловит паники сам).
  - Seed corpora: пустая строка, unicode, пробелы, null bytes, topK=0/-1/MaxInt.
  Touches: `pkg/draftrag/fuzz_test.go`
- [x] T2.2 Создать `pkg/draftrag/roundtrip_test.go` с `TestVectorStoreRoundtrip(t *testing.T)`:
  - 100 итераций с rand-генерацией случайных Chunk (ID, Content, ParentID — rand strings; Embedding — []float64{rand}).
  - Upsert(ctx, chunk) → Search(ctx, chunk.Embedding, 10) → assert найден chunk с тем же ID.
  - Проверяет: нет ошибок, результат не пустой, ID совпадает.
  Touches: `pkg/draftrag/roundtrip_test.go`

## Фаза 3: Проверка

Цель: доказать, что fuzz и property тесты работают.

- [x] T3.1 Финальная проверка:
  - `go test -fuzz=FuzzValidateDocument -fuzztime=15s ./internal/domain/` — без паники.
  - `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=15s ./pkg/draftrag/` — без паники.
  - `go test -run TestVectorStoreRoundtrip -count=100 ./pkg/draftrag/` — PASS.
  - `go vet ./internal/domain/ ./pkg/draftrag/` — exit 0.
  Touches: `internal/domain/fuzz_test.go`, `pkg/draftrag/fuzz_test.go`, `pkg/draftrag/roundtrip_test.go`

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1
- AC-002 -> T1.2, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T1.1, T1.2 (seed corpora), T3.1
- AC-005 -> T3.1

## Заметки

- seed corpora задаются через `f.Add()` внутри каждой Fuzz-функции (не в init).
- Fuzz-функции для domain используют 2-3 примитивных аргумента (Go fuzzer ограничивает до ~8).
- Roundtrip-тест использует `math/rand` с фиксированным seed для воспроизводимости.
