# Contextual Chunking — Задачи

## Phase Contract

Inputs: plan, spec, data-model.md (no-change).
Outputs: упорядоченные исполнимые задачи с покрытием критериев.
Stop if: нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/infrastructure/chunker/contextual.go` | T1.1 |
| `internal/infrastructure/chunker/contextual_test.go` | T2.1, T4.1 |
| `pkg/draftrag/contextual_chunker.go` | T1.2 |
| `pkg/draftrag/contextual_chunker_test.go` | T2.2, T4.1 |
| `pkg/draftrag/errors.go` | T1.2 |

## Implementation Context

- Цель MVP: `ContextualChunker` — декоратор, обогащающий чанки контекстом из `Document.Metadata` через шаблон.
- Инварианты/семантика:
  - `ContextualChunker` реализует `domain.Chunker` через композицию с другим `Chunker`
  - Контекст — строка из `Document.Metadata[ContextKey]`; при пустом/отсутствующем — чанки без изменений
  - Шаблон: `{context}` и `{content}` — обязательные плейсхолдеры; пустой шаблон = ошибка
  - Уважает `ctx.Err()` — проверка перед вызовом базового чанкера
- Ошибки/коды: `ErrInvalidChunkerConfig` (существует в `pkg/draftrag/errors.go`)
- Контракты/протокол: `ContextualChunkerOptions{Base Chunker, ContextKey string, Template string}`
- Границы scope: не меняем `Document`, `Chunk`, `domain.Chunker`;
  не добавляем multi-key источники;
  не добавляем поле контекста в модель;
  не расширяем PipelineOptions.
- Proof signals: `go test ./internal/infrastructure/chunker/ -run TestContextual -v && go test ./pkg/draftrag/ -run TestContextual -v`
- References: DEC-001 (декоратор), DEC-002 (контекст в Content), DEC-003 (одно поле).

## Фаза 1: Основа

Цель: реализовать внутренний декоратор и публичный wrapper.

- [x] T1.1 Реализовать `ContextualChunker` в `internal/infrastructure/chunker/contextual.go`
  - Поля: `base domain.Chunker`, `contextKey string`, `template string`
  - `Chunk(ctx, doc)`: проверить ctx.Err, вызвать `base.Chunk`, извлечь контекст из `doc.Metadata[contextKey]`, при пустом — вернуть как есть; иначе для каждого чанка заменить `Content` на `strings.ReplaceAll(template, "{context}", ctx) + ReplaceAll("{content}", chunk.Content)`.
  - Touches: `internal/infrastructure/chunker/contextual.go`, `internal/domain/interfaces.go` (Chunker interface — read-only reference)
- [x] T1.2 Реализовать публичный `NewContextualChunker` в `pkg/draftrag/contextual_chunker.go`
  - `ContextualChunkerOptions{Base Chunker, ContextKey string, Template string}`
  - Валидация: Base не nil, Template не пуст, Template содержит `{content}`, ContextKey не пуст.
  - Возвращает `(Chunker, error)` с `ErrInvalidChunkerConfig` при ошибке.
  - Touches: `pkg/draftrag/contextual_chunker.go`, `pkg/draftrag/errors.go` (read-only reference)

## Фаза 2: MVP Slice

Цель: unit-тесты для базового поведения.

- [x] T2.1 Добавить unit-тесты внутреннего `ContextualChunker` в `internal/infrastructure/chunker/contextual_test.go`
  - AC-001: дефолтный шаблон + контекст — `HasPrefix(chunk.Content, "CONTEXT тест\\n")`
  - AC-002: кастомный шаблон — `HasPrefix(chunk.Content, "Doc: ")` + контент после `---`
  - AC-003: пустой/отсутствующий Metadata — чанки идентичны базовому чанкеру
  - AC-004: отменённый контекст — возвращает `ctx.Err()`
  - AC-006: `ContextKey="description"` — чанки с "Annual Report 2025"
  - Touches: `internal/infrastructure/chunker/contextual_test.go`, `internal/infrastructure/chunker/contextual.go`
- [x] T2.2 Добавить unit-тесты публичного wrapper в `pkg/draftrag/contextual_chunker_test.go`
  - Валидация: nil Base → ошибка, пустой Template → ошибка, Template без `{content}` → ошибка, пустой ContextKey → ошибка
  - Успешный конструктор с валидными опциями
  - Touches: `pkg/draftrag/contextual_chunker_test.go`, `pkg/draftrag/contextual_chunker.go`

## Фаза 3: Основная реализация

Цель: интеграционный тест через Pipeline (AC-005).

- [x] T3.1 Добавить интеграционный тест AC-005 в `pkg/draftrag/contextual_chunker_test.go`
  - Создать Pipeline с ContextualChunker (оборачивает BasicChunker), проиндексировать документ с контекстом, выполнить поиск по слову из контекста (отсутствующему в исходном тексте), проверить что чанк найден.
  - Touches: `pkg/draftrag/contextual_chunker_test.go`, `pkg/draftrag/contextual_chunker.go`

## Фаза 4: Проверка

Цель: доказать работоспособность и чистоту кода.

- [x] T4.1 Запустить `go vet ./internal/infrastructure/chunker/ ./pkg/draftrag/`, `golangci-lint run ./internal/infrastructure/chunker/ ./pkg/draftrag/` и все тесты (`go test ./internal/infrastructure/chunker/ ./pkg/draftrag/ -v`) — без ошибок.
  - Touches: нет изменений кода

## Покрытие критериев приемки

- AC-001 -> T1.1, T2.1
- AC-002 -> T1.1, T2.1
- AC-003 -> T1.1, T2.1
- AC-004 -> T1.1, T2.1
- AC-005 -> T1.1, T1.2, T3.1
- AC-006 -> T1.1, T1.2, T2.1
- RQ-005 (валидация) -> T1.2, T2.2
