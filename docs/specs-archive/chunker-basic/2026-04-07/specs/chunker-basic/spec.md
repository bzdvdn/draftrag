# BasicChunker для draftRAG

## Scope Snapshot

- In scope: базовая реализация `Chunker`, которая детерминированно разбивает `Document.Content` на последовательность `Chunk` фиксированного размера (по рунам), с опциональным overlap и лимитом `MaxChunks`, с поддержкой `context.Context`.
- Out of scope: токенизация по BPE/WordPiece, семантический чанкинг, split по предложениям/markdown, language-aware правила, извлечение заголовков/структуры документа, persisted state.

## Цель

Дать пользователю draftRAG готовую реализацию `Chunker`, чтобы он мог получить чанки из `Document` без сторонних библиотек и без собственного кода разбиения, с предсказуемым контрактом (IDs/Position/ParentID), валидацией опций и уважением отмены контекста.

## Основной сценарий

1. Разработчик создаёт `ch := draftrag.NewBasicChunker(opts)`.
2. Вызывает `chunks, err := ch.Chunk(ctx, doc)`.
3. Получает `[]Chunk`, где каждый `Chunk`:
   - имеет `ParentID == doc.ID`,
   - имеет монотонный `Position` (0..n-1),
   - имеет детерминированный `ID` на основе `doc.ID` и `Position`,
   - содержит непустой `Content` (после `TrimSpace`).

## Scope

- Public API в `pkg/draftrag`:
  - options struct (chunk size / overlap / max chunks (опционально))
  - фабрика `NewBasicChunker(opts) Chunker`
  - sentinel-ошибка конфигурации `ErrInvalidChunkerConfig` для проверок через `errors.Is`
- Infrastructure реализация в `internal/infrastructure/chunker` (только stdlib).
- Unit-тесты без внешней сети (обычный `testing`).

## Контекст

- В домене уже существует интерфейс `Chunker` и модели `Document`/`Chunk`.
- Конституция требует: `context.Context` первым параметром, `nil` context — panic, минимальные зависимости, русские godoc для публичного API.

## Требования

- RQ-001 ДОЛЖНА существовать публичная фабрика `NewBasicChunker(opts)` в `pkg/draftrag`, возвращающая `draftrag.Chunker` без импорта `internal/...`.
- RQ-002 Реализация ДОЛЖНА детерминированно разбивать `Document.Content` по рунам на чанки длиной `ChunkSize` (последний чанк может быть короче).
- RQ-003 Реализация ДОЛЖНА поддерживать overlap: следующий чанк начинается не позже чем на `Overlap` рун раньше конца предыдущего (при `Overlap > 0`).
- RQ-004 Все операции ДОЛЖНЫ уважать `ctx.Done()`; при отмене возвращать `context.Canceled`/`context.DeadlineExceeded` (без оборачивания в несопоставимые ошибки).
- RQ-005 Валидация конфигурации ДОЛЖНА быть детерминированной и сопоставимой через `errors.Is(err, draftrag.ErrInvalidChunkerConfig)`.
- RQ-006 Публичные типы/функции chunker ДОЛЖНЫ иметь godoc на русском языке.
- RQ-007 Тесты ДОЛЖНЫ проходить без внешней сети и без дополнительных сервисов.
- RQ-008 ДОЛЖЕН поддерживаться лимит `MaxChunks`: при `MaxChunks > 0` метод `Chunk` ДОЛЖЕН вернуть не более `MaxChunks` чанков (best-effort: обрезка без ошибки).

## Алгоритм (v1, минимальный контракт)

- `ChunkSize` задаёт целевую длину чанка в рунах (обязательно `> 0`).
- `Overlap` задаёт количество рун перекрытия между чанками (обязательно `>= 0` и `< ChunkSize`).
- `MaxChunks` ограничивает количество возвращаемых чанков (обязательно `>= 0`; `0` означает “без лимита”).
- Разбиение идёт слева направо:
  - берём диапазон `[start, start+ChunkSize)` по рунам (с клиппингом по длине),
  - формируем `Content` как подстроку этого диапазона,
  - применяем `strings.TrimSpace` к `Content`; если пусто — чанк пропускаем,
  - `ParentID = doc.ID`, `Position = i` (монотонно, без “дыр”), `ID = fmt.Sprintf("%s:%d", doc.ID, Position)`,
  - следующий `start = end - Overlap` (или `end`, если `Overlap == 0`), при этом `start` всегда увеличивается и цикл завершается.

Если `MaxChunks > 0`, чанкер прекращает добавлять чанки после достижения лимита и возвращает уже собранный префикс чанков (обрезка без ошибки).

## Критерии приемки

### AC-001 Публичная фабрика Chunker доступна из pkg/draftrag

- **Given** пользователь импортирует только `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он создаёт chunker через `draftrag.NewBasicChunker(opts)`
- **Then** код компилируется, а возвращаемое значение удовлетворяет интерфейсу `draftrag.Chunker`
- Evidence: compile-time assertion тест в `pkg/draftrag`.

### AC-002 Чанкинг детерминирован и формирует корректные поля Chunk

- **Given** документ с фиксированным `ID` и `Content`
- **When** вызывается `Chunk(ctx, doc)` с валидными опциями
- **Then** возвращаются чанки, где `ParentID == doc.ID`, `Position` монотонен, `ID` детерминирован и `Content` непустой
- Evidence: unit-тест в infrastructure.

### AC-003 Overlap работает

- **Given** `ChunkSize` и `Overlap > 0`
- **When** выполняется `Chunk(ctx, doc)`
- **Then** соседние чанки перекрываются на `Overlap` рун (в пределах границ)
- Evidence: unit-тест на известной строке.

### AC-004 Контекстная отмена работает

- **Given** отменённый `ctx` (cancel/deadline)
- **When** вызывается `Chunk(ctx, doc)`
- **Then** метод возвращает `context.Canceled`/`context.DeadlineExceeded` не позднее чем через 100мс (в тестовом сценарии)
- Evidence: unit-тест.

### AC-005 Конфигурация валидируется через errors.Is

- **Given** невалидные options (`ChunkSize <= 0`, `Overlap < 0` или `Overlap >= ChunkSize`)
- **When** вызывается `Chunk(ctx, doc)`
- **Then** возвращается ошибка, совместимая с `errors.Is(err, draftrag.ErrInvalidChunkerConfig)`
- Evidence: unit-тест в `pkg/draftrag` или infrastructure.

### AC-006 MaxChunks ограничивает количество возвращаемых чанков

- **Given** документ с длинным `Content` и `MaxChunks > 0`
- **When** вызывается `Chunk(ctx, doc)`
- **Then** возвращается не более `MaxChunks` чанков
- Evidence: unit-тест в infrastructure.

## Вне scope

- Токен-ориентированный чанкинг.
- Разбиение с учётом структуры (markdown/html/pdf).
- Семантический чанкинг и эвристики по предложениям.
- Параллельная обработка и streaming chunk generation.

## Открытые вопросы

- none
