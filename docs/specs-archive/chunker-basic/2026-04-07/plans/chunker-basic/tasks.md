# BasicChunker для draftRAG — Задачи

## Phase Contract

Inputs: `.speckeep/plans/chunker-basic/plan.md`, `.speckeep/plans/chunker-basic/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/errors.go | T1.1 |
| pkg/draftrag/basic_chunker.go | T1.2, T2.1 |
| internal/infrastructure/chunker/basic.go | T2.2 |
| internal/infrastructure/chunker/basic_test.go | T3.1 |
| pkg/draftrag/basic_chunker_test.go | T3.2 |
| domain.Chunker | T1.2, T2.2 |
| errors.Is | T1.1, T2.1, T3.2 |

## Фаза 1: Публичные ошибки и API каркас

Цель: зафиксировать публичный контракт ошибок и фабрику Chunker.

- [x] T1.1 Обновить `pkg/draftrag/errors.go` — добавить `ErrInvalidChunkerConfig` (sentinel) для проверок через `errors.Is`. Touches: pkg/draftrag/errors.go
- [x] T1.2 Создать `pkg/draftrag/basic_chunker.go` — options struct (`ChunkSize`, `Overlap`, `MaxChunks`) и фабрика `NewBasicChunker(opts) Chunker`; godoc на русском; ошибки конфигурации возвращаются из `Chunk` через `ErrInvalidChunkerConfig`. Touches: pkg/draftrag/basic_chunker.go

## Фаза 2: Infrastructure реализация (rune-based + overlap + MaxChunks)

Цель: реализовать алгоритм чанкинга и уважение контекста.

- [x] T2.1 Реализовать `Chunk(ctx, doc)` в публичном chunker: `ctx != nil` (panic), ранний `ctx.Err()`, валидация документа (`doc.Validate()`), валидация options (`ChunkSize > 0`, `Overlap >= 0`, `Overlap < ChunkSize`, `MaxChunks >= 0`), `errors.Is` для `ErrInvalidChunkerConfig`, делегирование в infra. Touches: pkg/draftrag/basic_chunker.go
- [x] T2.2 Создать `internal/infrastructure/chunker/basic.go` — реализация rune-based чанкинга: `ChunkSize`/`Overlap`/`MaxChunks`, `TrimSpace` и пропуск пустых чанков, детерминированные поля (`ParentID`, `Position`, `ID = fmt.Sprintf(\"%s:%d\", doc.ID, position)`), уважение `ctx.Done()` в цикле, best-effort обрезка по `MaxChunks`. Touches: internal/infrastructure/chunker/basic.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC через unit-тесты.

- [x] T3.1 Создать `internal/infrastructure/chunker/basic_test.go` — unit-тесты: AC-002 (детерминизм/поля), AC-003 (overlap), AC-004 (ctx cancel/deadline ≤ 100мс), AC-006 (MaxChunks ограничивает количество). Touches: internal/infrastructure/chunker/basic_test.go
- [x] T3.2 Создать `pkg/draftrag/basic_chunker_test.go` — тесты публичного API: AC-001 (compile-time assertion), AC-005 (errors.Is для ErrInvalidChunkerConfig). Touches: pkg/draftrag/basic_chunker_test.go

## Покрытие критериев приемки

- AC-001 -> T1.2, T3.2
- AC-002 -> T2.2, T3.1
- AC-003 -> T2.2, T3.1
- AC-004 -> T2.2, T3.1
- AC-005 -> T1.1, T1.2, T2.1, T3.2
- AC-006 -> T2.2, T3.1

## Заметки

- Все тесты должны проходить без внешней сети.
- Валидация конфигурации должна быть детерминированной и проверяемой через `errors.Is`.
