# Fuzz и Property Tests

## Scope Snapshot

- In scope: fuzz-тесты (Go 1.18+ `testing.F`) для критических путей обработки пользовательского ввода в domain и pipeline, property-based roundtrip-тесты для VectorStore.
- Out of scope: fuzz-тесты для HTTP-клиентов (qdrant, chromadb, etc.), интеграционные fuzz-тесты с реальными бекендами, property-тесты для LLM/Embedder (внешние зависимости).

## Цель

Разработчик, меняющий логику валидации в Pipeline или SearchBuilder, сейчас не защищён от регрессий в обработке краевых случаев ввода. После внедрения fuzz-тестов `go test -fuzz=FuzzValidate ./pkg/draftrag/` находит паники и некорректные состояния за <1 минуты. Успех измеряется: ≥4 fuzz-корпуса, ≥2 property-теста, 0 panics на случайных данных.

## Основной сценарий

1. Стартовая точка: fuzz-тест генерирует случайные строки/комбинации флагов.
2. Основное действие: тест передаёт их в `SearchBuilder.validate()`, `Document.Validate()`, `Chunk.Validate()`, `Pipeline.Index`/`Query`.
3. Результат: функция не паникует, возвращает детерминированную ошибку или успех.
4. Дополнительно: roundtrip-тест upsert → search подтверждает, что данные не теряются.

## User Stories

- P1 (MVP): fuzz-тесты для domain-валидации (Document, Chunk, Query) + SearchBuilder.validate + pickRoute с random флагами.
- P2: property-based roundtrip-тесты для VectorStore (upsert → search → result содержит chunk).

## MVP Slice

- `internal/domain/fuzz_test.go` — FuzzValidateDocument, FuzzValidateChunk, FuzzValidateQuery
- `pkg/draftrag/fuzz_test.go` — FuzzSearchBuilderValidate
- Каждый fuzz-тест проверяет: нет паники, возврат детерминированного результата (ошибка или успех).
- 2 property-теста (roundtrip VectorStore).

## First Deployable Outcome

- `go test -fuzz=FuzzValidateDocument -fuzztime=10s ./internal/domain/` — 0 panics.
- `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=10s ./pkg/draftrag/` — 0 panics.

## Scope

- `internal/domain/fuzz_test.go` — fuzz-тесты для Document.Validate, Chunk.Validate, Query.Validate
- `pkg/draftrag/fuzz_test.go` — fuzz-тесты для SearchBuilder.validate + property roundtrip
- Существующие Validate-функции не меняются

## Контекст

- Go 1.23+ — fuzzing встроен в `testing.F`, не требует внешних зависимостей
- SearchBuilder имеет 6 маршрутов и 7 output-методов — комбинаторный взрыв, fuzzing:
  - случайная строка question (пустая, unicode, только пробелы, null bytes)
  - topK: 0, отрицательный, огромный
  - ParentIDs: nil, пустой, с пустыми строками
  - HybridConfig: nil, нулевые поля
- Document/Chunk validation: пустые ID, unicode, очень длинные строки (near memory limit)
- Property roundtrip: InMemoryStore — upsert случайного Chunk → Search → chunk найден, ID совпадает

## Требования

- RQ-001 FuzzValidateDocument ДОЛЖЕН принимать `[]byte` как случайный документ, парсить в Document, вызывать Validate.
- RQ-002 FuzzValidateChunk ДОЛЖЕН принимать `[]byte` как случайный чанк, парсить в Chunk, вызывать Validate.
- RQ-003 FuzzValidateQuery ДОЛЖЕН принимать случайную строку и int topK, вызывать Query.Validate.
- RQ-004 FuzzSearchBuilderValidate ДОЛЖЕН принимать случайный question и комбинацию флагов (строку-конфиг), вызывать validate.
- RQ-005 Property roundtrip VectorStore ДОЛЖЕН: Upsert(random chunk) → Search(embedding) → результат содержит chunk с тем же ID.
- RQ-006 Все fuzz-тесты ДОЛЖНЫ использовать `testing.F` (Go native).

## Вне scope

- Fuzz-тесты для HTTP-клиентов (qdrant, chromadb, weaviate, milvus, pgvector) — требуют моков.
- Fuzz-тесты для chunker (basic_chunker.go) — зависит от Document, покрывается через FuzzValidateDocument.
- Parallel fuzzing (Go fuzzer сам управляет параллелизмом).

## Критерии приемки

### AC-001 Fuzz-тесты для domain-валидации

- Почему это важно: Document/Chunk/Query — базовые типы, их Validate вызывается на каждом входе.
- **Given** `internal/domain/fuzz_test.go` с FuzzValidateDocument, FuzzValidateChunk, FuzzValidateQuery
- **When** `go test -fuzz=FuzzValidate -fuzztime=15s ./internal/domain/`
- **Then** ни один seed не вызывает panic, все возвращают детерминированную ошибку или nil
- Evidence: fuzztime=15s без краша

### AC-002 Fuzz-тест для SearchBuilder

- Почему это важно: SearchBuilder — центральная точка обработки запросов.
- **Given** `pkg/draftrag/fuzz_test.go` с FuzzSearchBuilderValidate
- **When** `go test -fuzz=FuzzSearchBuilderValidate -fuzztime=15s ./pkg/draftrag/`
- **Then** ни один seed не вызывает panic, validate возвращает детерминированный результат
- Evidence: fuzztime=15s без краша

### AC-003 Property roundtrip для VectorStore

- Почему это важно: гарантирует, что Upsert+Search работают как ожидается для любых данных.
- **Given** InMemoryStore
- **When** случайный Chunk генерируется, Upsert → Search(его embedding)
- **Then** результат содержит chunk с тем же ID
- Evidence: `go test -run TestVectorStoreRoundtrip -count=100` — 0 failures

### AC-004 Fuzz-тесты не падают на существующих seed-корпусах

- Почему это важно: seed-корпуса должны покрывать базовые случаи.
- **Given** fuzz-тесты с seed-корпусами (пустая строка, unicode, null byte)
- **When** `go test -run TestFuzzSeedCorpora ./internal/domain/ ./pkg/draftrag/`
- **Then** все seed-корпуса проходят
- Evidence: `go test -run TestFuzzSeedCorpora` PASS

### AC-005 go vet + build без errors

- **Given** все изменения завершены
- **When** `go vet ./internal/domain/ ./pkg/draftrag/`
- **Then** exit code 0
- Evidence: `go vet` PASS

## Допущения

- Fuzz-тесты используют Go native fuzzer без external библиотек (rapid/gofuzz).
- InMemoryStore — reference для roundtrip; property-тесты не переносятся на production store.
- fuzztime=15s достаточно для базового покрытия; полное fuzz-тестирование может занимать часы.

## Критерии успеха

- SC-001 ≥4 fuzz-функций (`FuzzValidateDocument`, `FuzzValidateChunk`, `FuzzValidateQuery`, `FuzzSearchBuilderValidate`).
- SC-002 ≥2 property-теста для VectorStore roundtrip.
- SC-003 0 panics за 15s fuzzing на каждом корпусе.

## Краевые случаи

- Пустая строка: question="", doc.ID="" → ожидаемая sentinel-ошибка, не panic.
- Unicode: кириллица, иероглифы, эмодзи, RTL — Validate должен корректно обрабатывать.
- Null bytes: `\x00` в строке — валидация должна вернуть ошибку или пропустить (не panic).
- Огромные строки: 10MB+ — Validate может быть дорогим, но не паниковать.
- topK = MinInt / MaxInt — переполнение int, корректная обработка.
- ParentIDs с дубликатами и пустыми строками — SearchBuilder не должен паниковать.

## Открытые вопросы

- Нужен ли corpus-директорий для seed-значений (testdata/fuzz/)? Нет, seed задаются через `f.Add()`.
