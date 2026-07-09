# Ограничение контекста в Prompt (MaxContextChars/MaxContextChunks) для draftRAG

## Scope Snapshot

- In scope: ограничение размера секции “Контекст:” в prompt для `Pipeline.Answer*` через options (например, `MaxContextChars` и/или `MaxContextChunks`), чтобы не формировать слишком большие сообщения для LLM.
- Out of scope: лимитирование по токенам (BPE), авто-компрессия/суммаризация контекста, reranking, динамические prompt templates, streaming.

## Цель

Сделать `Pipeline.Answer*` безопаснее и предсказуемее: даже если retrieval вернул много чанков или очень длинные чанки, сформированный prompt не должен бесконтрольно расти.

## Основной сценарий

1. Пользователь создаёт pipeline с options:
   - `PipelineOptions{MaxContextChars: 2000}` (и/или `MaxContextChunks: 5`)
2. Вызывает `p.AnswerTopK(ctx, question, topK=20)`.
3. Retrieval может вернуть 20 чанков, но в prompt попадёт только часть:
   - не более `MaxContextChunks` чанков,
   - и/или не более `MaxContextChars` символов секции контекста.

## Scope

- Public API (`pkg/draftrag`):
  - добавить поля в `PipelineOptions`: `MaxContextChars int` и `MaxContextChunks int` (оба optional; `0` означает “без лимита”).
- Application (`internal/application`):
  - обновить построение user message (Prompt Contract v1), чтобы применять лимиты к секции “Контекст:”.
- Testing:
  - unit-тесты на детерминированное обрезание контекста по чанкам и по символам.

## Контекст

- Сейчас `Answer*` формирует `userMessage` через `buildUserMessageV1`, добавляя все чанки из retrieval результата.
- Уже есть `PipelineOptions` и `NewPipelineWithOptions`, поэтому лимит логично задавать через options.
- Конституция: минимальные зависимости, русские godoc, тестируемость без внешней сети.

## Требования

- RQ-001 ДОЛЖНЫ быть добавлены опции `MaxContextChars` и `MaxContextChunks` в `PipelineOptions` (в `pkg/draftrag`).
- RQ-002 Значение `0` ДОЛЖНО означать “лимит выключен”.
- RQ-003 При `MaxContextChunks > 0` в секцию “Контекст:” ДОЛЖНЫ попасть только первые `MaxContextChunks` чанков (в порядке retrieval результата).
- RQ-004 При `MaxContextChars > 0` суммарная длина секции “Контекст:” (только контекст, без “Вопрос:”) ДОЛЖНА быть не больше `MaxContextChars` символов.
- RQ-005 Если включены оба лимита — ДОЛЖНЫ применяться оба (результат удовлетворяет обоим ограничениям).
- RQ-006 Лимитирование не ДОЛЖНО менять формат Prompt Contract v1 (заголовки “Контекст:” и “Вопрос:” остаются).
- RQ-007 Валидация options:
  - `MaxContextChars < 0` или `MaxContextChunks < 0` -> panic (ошибка конфигурации).
- RQ-008 Тесты ДОЛЖНЫ проходить без внешней сети.
- RQ-009 Публичные поля options ДОЛЖНЫ иметь godoc на русском.

## Критерии приемки

### AC-001 Options расширены и доступны из pkg/draftrag

- **Given** пользователь импортирует `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он использует `PipelineOptions{MaxContextChars: ..., MaxContextChunks: ...}`
- **Then** код компилируется
- Evidence: compile-time тест в `pkg/draftrag`.

### AC-002 MaxContextChunks ограничивает количество чанков в prompt

- **Given** retrieval вернул 3 чанка и `MaxContextChunks=1`
- **When** вызывается `AnswerTopK`
- **Then** в user message присутствует только 1 чанк из контекста
- Evidence: unit-тест на аргументы `LLM.Generate`.

### AC-003 MaxContextChars ограничивает длину контекста

- **Given** retrieval вернул длинные чанки и `MaxContextChars` малый
- **When** вызывается `AnswerTopK`
- **Then** длина секции контекста не превышает лимит
- Evidence: unit-тест.

### AC-004 Совместное применение лимитов

- **Given** включены оба лимита
- **When** вызывается `AnswerTopK`
- **Then** результат одновременно удовлетворяет `MaxContextChunks` и `MaxContextChars`
- Evidence: unit-тест.

## Вне scope

- Токен-лимиты и адаптация под конкретные модели.
- Суммаризация/компрессия контекста.

## Открытые вопросы

- Обрезаем ли контекст “по границе чанка” только (проще и детерминированно), или допускаем обрезание внутри чанка для соблюдения `MaxContextChars`? (предпочтение: допускаем обрезание внутри последнего чанка, чтобы строго соблюдать лимит по символам).

