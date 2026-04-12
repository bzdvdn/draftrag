# BasicChunker для draftRAG — План

## Phase Contract

Inputs: `.speckeep/specs/chunker-basic/spec.md`, `.speckeep/specs/chunker-basic/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно зафиксировать детерминированный контракт чанкинга и лимитирования без расплывчатости.

## Цель

Добавить базовую реализацию `Chunker`, которая разбивает `Document.Content` на чанки фиксированного размера по рунам с overlap и лимитом `MaxChunks`. Реализация должна быть:
- доступна пользователю через `pkg/draftrag` (без импорта `internal/...`),
- контекстно-безопасна (ctx cancel/deadline),
- детерминированна по результату (IDs/Position/ParentID),
- полностью тестируема без внешней сети,
- с детерминированной валидацией options через sentinel-ошибку.

## Scope

- Public API: options + фабрика `NewBasicChunker(opts) Chunker` + sentinel `ErrInvalidChunkerConfig`.
- Infrastructure: реализация алгоритма чанкинга (stdlib).
- Testing: unit-тесты на корректность, overlap, MaxChunks, контекст, валидацию.

## Implementation Surfaces

- `pkg/draftrag/errors.go` — добавить `ErrInvalidChunkerConfig` (sentinel) (T1.1).
- `pkg/draftrag/basic_chunker.go` — публичная фабрика, options, валидация конфигурации и метод `Chunk` (T1.2, T2.1).
- `internal/infrastructure/chunker/basic.go` — реализация rune-based чанкинга + overlap + MaxChunks, уважение контекста (T2.2).
- `internal/infrastructure/chunker/basic_test.go` — unit-тесты на алгоритм (детерминизм, overlap, MaxChunks) и контекст (T3.1).
- `pkg/draftrag/basic_chunker_test.go` — тесты публичного API (compile-time, errors.Is) (T3.2).
- `domain.Chunker` — доменный интерфейс, который реализуется basic chunker (T1.2, T2.2).

## Влияние на архитектуру

- Clean Architecture сохраняется: интерфейс `Chunker` остаётся в domain; реализация — в infrastructure; публичный доступ — `pkg/draftrag`.
- Зависимости: только стандартная библиотека.
- Никаких migration/rollout действий: библиотечное аддитивное API.

## Acceptance Approach

- AC-001 -> фабрика в `pkg/draftrag/basic_chunker.go` + compile-time assertion в `pkg/draftrag/basic_chunker_test.go`.
- AC-002 -> unit-тесты в `internal/infrastructure/chunker/basic_test.go`: проверка `ParentID`, `Position`, `ID` и непустых `Content`.
- AC-003 -> unit-тест на известной строке: соседние чанки перекрываются на `Overlap` рун (в пределах границ). Surface: `internal/infrastructure/chunker/basic_test.go`.
- AC-004 -> unit-тесты `context.WithCancel` / `context.WithTimeout`: возврат `context.Canceled`/`context.DeadlineExceeded` не позже 100мс (тестовый сценарий). Surface: `internal/infrastructure/chunker/basic_test.go`.
- AC-005 -> ошибки конфигурации через `errors.Is(err, ErrInvalidChunkerConfig)` (вызов `Chunk`). Surfaces: `pkg/draftrag/basic_chunker.go`, `pkg/draftrag/basic_chunker_test.go`, `pkg/draftrag/errors.go`.
- AC-006 -> unit-тест на лимит `MaxChunks`: количество чанков не превышает лимит. Surface: `internal/infrastructure/chunker/basic_test.go`.

## Данные и контракты

- Data model: только options и вычисляемые поля результата (`Chunk.ID`, `Chunk.Position`, `Chunk.ParentID`). Persisted состояние отсутствует (см. `data-model.md`).
- Контракты: публичный контракт — `Chunker.Chunk(ctx, doc) ([]Chunk, error)`, плюс детерминированность output и sentinel-ошибка конфигурации.
- Внешние API/ивенты не меняются.

## Стратегия реализации

- DEC-001 Чанкинг по рунам (а не по байтам и не по токенам)
  Why: корректно для Unicode и соответствует минимальному v1 контракту без внешних зависимостей.
  Tradeoff: не коррелирует с “токенами” LLM и может давать чанки разной “стоимости”.
  Affects: `internal/infrastructure/chunker/basic.go`
  Validation: unit-тесты на детерминизм/overlap/MaxChunks.

- DEC-002 Фабрика без error; конфиг-ошибки возвращаются из `Chunk`
  Why: единообразие с `NewOpenAICompatibleEmbedder`/`NewOpenAICompatibleLLM` — фабрики возвращают интерфейс без error.
  Tradeoff: ошибки проявляются при первом вызове.
  Affects: `pkg/draftrag/basic_chunker.go`
  Validation: unit-тесты AC-005.

- DEC-003 Лимит `MaxChunks` как best-effort (обрезка без ошибки)
  Why: safety-by-default против взрыва числа чанков; при этом пайплайн продолжает работать.
  Tradeoff: пользователь может не заметить потери хвоста документа без явного мониторинга.
  Affects: `internal/infrastructure/chunker/basic.go`, `pkg/draftrag/basic_chunker.go`
  Validation: unit-тест AC-006.

- DEC-004 Детеминированный ID: `fmt.Sprintf("%s:%d", doc.ID, position)`
  Why: простое и предсказуемое соответствие “документ+позиция”.
  Tradeoff: ID не устойчив к изменению разбиения при смене опций; это ожидаемо.
  Affects: `internal/infrastructure/chunker/basic.go`
  Validation: unit-тест AC-002.

## Incremental Delivery

### MVP (Первая ценность)

- Реализация `Chunk` с `ChunkSize` и `Overlap` + валидация options + базовые unit-тесты AC-001..AC-005.
- Готовность: `go test ./...` проходит, unit-тесты подтверждают детерминизм и контекст.

### Итеративное расширение

- Добавить `MaxChunks` ограничение (если не входит в MVP) + тест AC-006.
- (Out of scope) Позже: sentence/markdown chunking, token-based chunking.

## Порядок реализации

1. Sentinel-ошибка `ErrInvalidChunkerConfig` (чтобы зафиксировать контракт).
2. Публичный wrapper `pkg/draftrag/basic_chunker.go` (options + validation + timeout/ctx contract).
3. Infrastructure реализация алгоритма.
4. Unit-тесты infra и pkg.

## Риски

- Риск 1: бесконечный цикл при overlap (если `Overlap >= ChunkSize`).
  Mitigation: строгая валидация `Overlap < ChunkSize` + тесты.
- Риск 2: слишком большие документы приводят к большим затратам.
  Mitigation: `MaxChunks` (best-effort) + ранний `ctx.Err()` в цикле.

## Rollout и compatibility

- Rollout не требуется: аддитивная библиотечная функциональность.
- Compatibility: добавление новых фабрик и ошибок без изменения существующего поведения.

## Проверка

- Automated:
  - `go test ./...`
  - Unit-тесты: AC-001..AC-006
- Manual:
  - `go doc` на публичные symbols (godoc на русском) для `NewBasicChunker` и `BasicChunkerOptions`.

## Соответствие конституции

- нет конфликтов: интерфейсы сохраняются, зависимости минимальны, `context.Context` первым параметром, тестируемость обеспечена.
