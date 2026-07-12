# CJK Tokenization — Поддержка CJK в чанкере

## Scope Snapshot

- In scope: адаптация встроенных чанкеров (BasicChunker, SemanticChunker) для корректной обработки текста на китайском, японском и корейском (CJK) языках.
- Out of scope: полноценный word segmentation/tokenizer (Jieba, Kuromoji, etc.), добавление внешних NLP-зависимостей, поддержка вертикальных/двунаправленных текстов.

## Цель

Разработчики, индексирующие CJK-документы, получают осмысленные чанки вместо одного гигантского (SemanticChunker) или разорванных произвольно (BasicChunker). Успех измеряется корректным sentence-level split для CJK-текста и отсутствием регрессии для латиницы.

## Основной сценарий

1. Пользователь создаёт Pipeline с BasicChunker или SemanticChunker.
2. В Pipeline попадает документ, содержащий CJK-текст (китайский, японский или корейский).
3. BasicChunker разбивает по рунам — не ломает mid-word (CJK-символы — каждый отдельная руна, деление по рунам не хуже латиницы).
4. SemanticChunker определяет границы предложений по CJK-пунктуации (`。`, `！`, `？`) и собирает чанки по семантической близости.
5. Для латинских текстов поведение не меняется.

## User Stories

- P1: SemanticChunker корректно разбивает CJK-текст на предложения по CJK-пунктуации.
- P2: BasicChunker чётко задокументирован как rune-based (уже корректно работает с CJK).
- P3: Mixed CJK/Latin текст обрабатывается без ошибок и потерь контента.

## MVP Slice

Добавить CJK-пунктуацию (`。`, `！`, `？`) в `splitSentences` + CJK-границы в `isSentenceBoundary`. Это закрывает AC-001, AC-002, AC-003, AC-004.

## First Deployable Outcome

`go test ./internal/infrastructure/chunker/ -run TestCJK` показывает, что CJK-текст разбивается на несколько предложений, а не возвращается как один чанк.

## Scope

- `internal/infrastructure/chunker/semantic.go` — модификация `splitSentences` и `isSentenceBoundary`
- `internal/infrastructure/chunker/basic.go` — без изменений кода (уже CJK-совместим)
- `internal/infrastructure/chunker/cjk_test.go` — новые тесты
- `pkg/draftrag/` — без изменений публичного API (изменения только внутри `internal/`)
- `internal/domain/interfaces.go` — read-only reference (Chunker interface не меняется)

## Контекст

- `BasicRuneChunker` использует `[]rune(content)` — все Unicode codepoints, включая CJK, обрабатываются корректно.
- `splitSentences` разбивает только по `.`, `!`, `?`. CJK использует `。` (U+3002), `！` (U+FF01), `？` (U+FF1F) как sentence-ending punctuation.
- `isSentenceBoundary` проверяет `unicode.IsUpper` после разделителя — для CJK это всегда false, что ломает определение границы.
- CJK-текст не использует пробелы между словами — split по пробелам неприменим; разделение по символам (рунам) — минимально корректная стратегия.
- Чанкинг должен оставаться stateless и без внешних NLP-зависимостей (constraint из конституции: простота > расширяемость).

## Зависимости

- `none` — фича не добавляет внешних зависимостей.

## Требования

- RQ-001 `splitSentences` ДОЛЖЕН распознавать CJK-пунктуацию (`。`, `！`, `？`) как конец предложения.
- RQ-002 `isSentenceBoundary` ДОЛЖЕН корректно определять границу после CJK-пунктуации без опоры на uppercase.
- RQ-003 CJK-текст, переданный в SemanticChunker, ДОЛЖЕН разбиваться на несколько чанков, а не возвращаться как один.
- RQ-004 BasicChunker ДОЛЖЕН оставаться CJK-совместимым (rune-based split уже корректен).
- RQ-005 Поведение для латинского текста НЕ ДОЛЖНО измениться после добавления CJK-поддержки.
- RQ-006 `isAbbreviation` НЕ ДОЛЖЕН ложно срабатывать на CJK-символах (точка `。` не является аббревиатурой).

## Вне scope

- Word segmentation (Jieba, Kuromoji, MeCab) — требует внешнюю библиотеку, violates constraint «простота > расширяемость».
- Добавление нового типа Chunker (CJKChunker) — достаточно модификации существующего.
- Обработка тайского, лаосского, кхмерского и других non-space-separated языков.
- Поддержка вертикального письма (tategaki).
- Изменение формата Chunk или Document.

## Критерии приемки

### AC-001 CJK punctuation as sentence delimiters

- Почему это важно: SemanticChunker не видит границ CJK-предложений → весь CJK-текст становится одним чанком.
- **Given** текст на китайском с несколькими предложениями, разделёнными `。`
- **When** `splitSentences(text)` вызывается
- **Then** результат содержит более одного предложения, и каждое предложение заканчивается на `。`
- Evidence: `TestCJK_SplitSentences_Chinese`

### AC-002 CJK exclamation and question marks

- Почему это важно: восклицательные и вопросительные предложения в CJK используют `！` и `？`.
- **Given** текст на японском с предложениями, заканчивающимися на `！` и `？`
- **When** `splitSentences(text)` вызывается
- **Then** результат содержит корректно разделённые предложения с `！` и `？`
- Evidence: `TestCJK_SplitSentences_Japanese`

### AC-003 CJK sentence boundary detection

- Почему это важно: `isSentenceBoundary` использует `unicode.IsUpper`, который для CJK всегда false.
- **Given** CJK-текст с предложением, заканчивающимся на `。` и за которым следует новый символ CJK
- **When** `isSentenceBoundary` вызывается для позиции `。`
- **Then** возвращается true (граница предложения)
- Evidence: `TestCJK_SentenceBoundary`

### AC-004 SemanticChunker produces multiple CJK chunks

- Почему это важно: интеграционный признак, что CJK-поддержка работает end-to-end.
- **Given** Pipeline с SemanticChunker и CJK-документом с 3+ предложениями
- **When** Pipeline.Index вызывается
- **Then** результат содержит более одного чанка
- Evidence: `TestCJK_SemanticChunker_MultipleChunks`

### AC-005 No regression for Latin text

- Почему это важно: изменения не должны ломать существующую функциональность для латиницы.
- **Given** латинский текст с предложениями, разделёнными `.`, `!`, `?`
- **When** `splitSentences` вызывается до и после изменений
- **Then** результат идентичен
- Evidence: существующие тесты `TestSplitSentences*` проходят без изменений

### AC-006 BasicChunker CJK compatibility

- Почему это важно: BasicChunker должен корректно работать с CJK-текстом без изменений.
- **Given** BasicChunker и CJK-документ
- **When** `Chunk` вызывается
- **Then** результат содержит ожидаемое количество чанков, контент не обрезан посередине руны
- Evidence: `TestCJK_BasicChunker_RuneSplit`

## Допущения

- CJK-пунктуация (`。`, `！`, `？`) — единственные CJK-специфичные sentence-ending символы; традиционные варианты (U+FE12, U+FE15, U+FE16, U+FF0E) выходят за рамки MVP.
- `splitSentences` остаётся stateless и не использует внешние словари.
- BasicChunker не требует изменений: `[]rune` уже корректно обрабатывает CJK.

## Критерии успеха

- SC-001: Все новые тесты для CJK проходят за <100ms.
- SC-002: Существующие тесты латинского текста (`TestSplitSentences*`) не требуют изменений.

## Краевые случаи

- Пустой CJK-текст → пустой слайс (как и для латиницы).
- CJK-текст без пунктуации → одно предложение (как и для латиницы).
- Mixed CJK/Latin: `「Hello world。Goodbye。」` → корректный split на `「Hello world。` и `Goodbye。」`.
- CJK-символы, не являющиеся пунктуацией (основной блок CJK Unified Ideographs U+4E00–U+9FFF), не должны вызывать split.

## Открытые вопросы

- Стоит ли добавить поддержку fullwidth Latin punctuation (`．` U+FF0E) как sentence delimiter? — Решение: нет в MVP, т.к. fullwidth Latin используется непоследовательно.

## Self-Check

- [x] Нет `TODO`, `???`, `<placeholder>`, `TKTK`, `[NEEDS CLARIFICATION]`
- [x] Каждый AC-* содержит `Given`, `When`, `Then` с observable proof в Then
- [x] Секции `Out of Scope`, `Допущения`, `Открытые вопросы` существуют
- [x] Нет implementation steps или декомпозиции
- [x] Технологии/версии не зафиксированы
- [x] Spec описывает ровно одну фичу
- [x] Goal и RQ-* ID согласованы с AC-* критериями
- [x] Каждый AC-* ведёт к уникальному observable outcome
