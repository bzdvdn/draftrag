# CJK Tokenization — Задачи

## Phase Contract

Inputs: plan.md, data-model.md, spec.md.
Outputs: исполнимые задачи с покрытием AC-001–AC-006.
Stop if: задачи расплывчаты или coverage не сопоставляется.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/chunker/semantic.go` | T1.1, T1.2 |
| `internal/infrastructure/chunker/cjk_test.go` | T2.1, T2.2, T2.3, T2.4 |
| `internal/infrastructure/chunker/semantic_test.go` | T3.1 (read-only regression) |
| `internal/infrastructure/chunker/basic_test.go` | T3.1 (read-only regression) |

## Implementation Context

- **Цель MVP:** добавить CJK-пунктуацию (`。`, `！`, `？`) в `splitSentences` + CJK-границы в `isSentenceBoundary` + тесты.
- **Инварианты/семантика:**
  - `splitSentences` остаётся stateless; CJK-символы добавляются как дополнительные разделители наравне с `.`, `!`, `?`
  - `isSentenceBoundary` для CJK-пунктуации: граница если следующий символ — не пунктуация и не пробел (вместо `unicode.IsUpper`)
  - BasicChunker не меняется — `[]rune` уже корректен для CJK
- **Ошибки/коды:** нет новых sentinel-ошибок
- **Контракты/протокол:** Chunker interface (`internal/domain/interfaces.go:121`) не меняется
- **Границы scope:** не добавляем word segmentation; не меняем модель данных; не меняем публичное API
- **Proof signals:** `go test ./internal/infrastructure/chunker/ -run TestCJK -v` pass; `go test ./internal/infrastructure/chunker/ -run TestSemanticChunker -v` pass (регрессия)
- **References:** DEC-001 (modify splitSentences), DEC-002 (CJK boundaries), DEC-003 (BasicChunker unchanged)

## Фаза 1: Основа

Цель: модифицировать существующие функции `splitSentences` и `isSentenceBoundary` для поддержки CJK.

- [x] T1.1 Добавить CJK-пунктуацию (`。` U+3002, `！` U+FF01, `？` U+FF1F) в качестве sentence-ending разделителей в `splitSentences`. Touches: `internal/infrastructure/chunker/semantic.go`
- [x] T1.2 Обновить `isSentenceBoundary` для корректного определения границы после CJK-пунктуации: когда следующий символ — не пунктуация и не whitespace (CJK-иероглифы, буквы, цифры). Touches: `internal/infrastructure/chunker/semantic.go`

## Фаза 2: Тесты

Цель: unit-тесты для всех AC + интеграционный тест через SemanticChunker.

- [x] T2.1 Добавить unit-тесты `splitSentences` с CJK: AC-001 (китайский `。`), AC-002 (японский `！`/`？`), AC-005 (латиница без регрессии). Touches: `internal/infrastructure/chunker/cjk_test.go`
- [x] T2.2 Добавить unit-тест `isSentenceBoundary` для CJK: AC-003. Touches: `internal/infrastructure/chunker/cjk_test.go`
- [x] T2.3 Добавить интеграционный тест SemanticChunker с CJK-документом: AC-004 — 3+ предложения → несколько чанков. Touches: `internal/infrastructure/chunker/cjk_test.go`
- [x] T2.4 Добавить тест BasicChunker с CJK: AC-006 — rune-level split не ломает CJK. Touches: `internal/infrastructure/chunker/cjk_test.go`
- [x] T3.1 Запустить `go vet ./internal/infrastructure/chunker/`, `golangci-lint run ./internal/infrastructure/chunker/`, и все тесты (`go test ./internal/infrastructure/chunker/ -v`) — без ошибок. Touches: нет изменений кода

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1
- AC-002 -> T1.1, T2.1
- AC-003 -> T1.2, T2.2
- AC-004 -> T1.1, T1.2, T2.3
- AC-005 -> T1.1, T2.1 (регрессия проверяется существующими `TestSemanticChunker_*`)
- AC-006 -> T2.4 (BasicChunker read-only)
