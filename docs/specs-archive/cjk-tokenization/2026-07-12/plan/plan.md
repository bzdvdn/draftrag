# CJK Tokenization — План

## Phase Contract

Inputs: spec.md, inspect.md (pass).
Outputs: plan.md, data-model.md.
Stop if: spec слишком расплывчата.

## Цель

Модифицировать `splitSentences` и `isSentenceBoundary` в SemanticChunker для поддержки CJK-пунктуации (`。`, `！`, `？`). BasicChunker не требует изменений (rune-based split уже корректен). Фича не добавляет публичного API и не меняет модель данных.

## MVP Slice

Добавить CJK-пунктуацию в `splitSentences` + CJK-границы в `isSentenceBoundary` + тесты.

Закрывает: AC-001, AC-002, AC-003, AC-004, AC-005, AC-006.

## First Validation Path

`go test ./internal/infrastructure/chunker/ -run TestCJK -v` — все CJK-тесты проходят. Затем `go test ./internal/infrastructure/chunker/ -run TestSplitSentences -v` — регрессии нет.

## Scope

- `internal/infrastructure/chunker/semantic.go` — модификация `splitSentences` и `isSentenceBoundary`
- `internal/infrastructure/chunker/cjk_test.go` — новый файл тестов
- `internal/domain/interfaces.go` — read-only reference
- `internal/infrastructure/chunker/basic.go` — без изменений

## Performance Budget

- `none` — добавление 3 проверок на CJK-символы не влияет на производительность (O(n) как и сейчас).

## Implementation Surfaces

| Surface | Изменение | Причина |
|---------|-----------|---------|
| `internal/infrastructure/chunker/semantic.go` | modify `splitSentences`, `isSentenceBoundary` | основной алгоритм сплиттинга |
| `internal/infrastructure/chunker/cjk_test.go` | new | тесты CJK-функциональности |

## Bootstrapping Surfaces

- `none` — все нужные файлы уже существуют.

## Влияние на архитектуру

- Локальное: только `internal/infrastructure/chunker/`.
- Интеграции: не затрагиваются (Chunker interface не меняется).
- Public API: не меняется.

## Acceptance Approach

- AC-001: `splitSentences` + new test `TestCJK_SplitSentences_Chinese`
- AC-002: `splitSentences` + new test `TestCJK_SplitSentences_Japanese`
- AC-003: `isSentenceBoundary` + new test `TestCJK_SentenceBoundary`
- AC-004: `splitSentences` + `isSentenceBoundary` + integration test через SemanticChunker
- AC-005: существующие тесты `TestSplitSentences*` без изменений — регрессия проверяется их pass
- AC-006: `BasicRuneChunker` (read-only) + new test `TestCJK_BasicChunker_RuneSplit`

## Данные и контракты

- Data model не меняется. См. `data-model.md`.

## Стратегия реализации

### DEC-001 Modify existing splitSentences

**Why**: CJK-пунктуация — это просто 3 дополнительных символа-разделителя в `splitSentences`. Создавать отдельный CJKChunker или дублировать логику — избыточно (простота > расширяемость).

**Tradeoff**: Если в будущем потребуется radically другой алгоритм для CJK (word segmentation), `splitSentences` придётся рефакторить. На данный момент это premature.

**Affects**: `internal/infrastructure/chunker/semantic.go`

**Validation**: AC-001, AC-002 проходят.

### DEC-002 CJK boundaries via rune category, not IsUpper

**Why**: `isSentenceBoundary` использует `unicode.IsUpper` для определения границы после пунктуации. Для CJK все символы — не uppercase. Нужна проверка `unicode.Is(unicode.Han, r)` или обобщение: any non-whitespace, non-punctuation character после CJK-пунктуации — граница.

**Tradeoff**: Обобщение на любой non-whitespace символ (вместо IsUpper) может давать ложные границы для аббревиатур в латинице. Решение: проверять CJK-пунктуацию отдельно от латинской, и для CJK-ветки использовать `!unicode.IsPunct(r) && !unicode.IsSpace(r)`.

**Affects**: `internal/infrastructure/chunker/semantic.go` — `isSentenceBoundary`

**Validation**: AC-003, AC-004, AC-005 проходят.

### DEC-003 BasicChunker unchanged — rune-based is correct for CJK

**Why**: Go `[]rune(content)` даёт каждый Unicode codepoint как отдельную руну. CJK-символ — один codepoint. Фиксированный split по рунам не ломает символы. Любое изменение здесь — over-engineering.

**Tradeoff**: Rune-based split не знает о word boundaries в CJK (нет пробелов), но это ограничение inherent в подходе, а не регрессия. Задокументировано в spec.

**Affects**: нет изменений кода

**Validation**: AC-006

## Incremental Delivery

### MVP

1. Add CJK punctuation to `splitSentences` (+ test)
2. Update `isSentenceBoundary` for CJK (+ test)
3. Integration test via SemanticChunker (+ test)
4. Regression check — existing tests unchanged
5. BasicChunker CJK test

### Итеративное расширение

- Нет — фича полностью закрывается MVP.

## Порядок реализации

1. `splitSentences` — добавить CJK-пунктуацию как разделители
2. `isSentenceBoundary` — добавить CJK-границы
3. Тесты — все 6 AC
4. `go vet ./...` + `go test ./internal/infrastructure/chunker/ -v`

Параллелизация: нет (зависимости последовательные).

## Риски

- **Регрессия латиницы**: `isSentenceBoundary` может сломаться, если обобщение boundary detection для CJK затронет латинский путь.
  Mitigation: AC-005 — существующие тесты `TestSplitSentences*` работают как регрессионный барьер.

- **Пропущенные CJK-разделители**: другие CJK-символы пунктуации (fullwidth period `．` U+FF0E, ideographic comma `、` U+3001, etc.) не включены в MVP.
  Mitigation: явно задокументировано в spec как осознанное out of scope.

## Rollout and compatibility

- Изменения только в `internal/` — публичный API не затронут.
- Feature не требует rollout steps, feature flags или миграций.

## Проверка

- `go test ./internal/infrastructure/chunker/ -run TestCJK -v` — 6+ новых тестов pass
- `go test ./internal/infrastructure/chunker/ -run TestSplitSentences -v` — 0 регрессий
- `go vet ./internal/infrastructure/chunker/` — clean
- `golangci-lint run ./internal/infrastructure/chunker/` — только pre-existing warnings

## Соответствие конституции

- нет конфликтов
