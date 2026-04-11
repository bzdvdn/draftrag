# PipelineOptions / NewPipelineWithOptions для draftRAG

## Scope Snapshot

- In scope: унификация конфигурации `Pipeline` через options struct и один публичный конструктор, чтобы управлять default topK, system prompt и (опционально) chunker’ом без разрастания числа фабрик.
- Out of scope: глобальные конфиги через env vars, конфиг-файлы, динамическая смена конфигурации во время работы, расширенные prompt templates, лимитирование контекста по токенам.

## Цель

Сделать публичный API `Pipeline` более стабильным и расширяемым: вместо множества конструкторов и “зашитых” дефолтов дать пользователю один entrypoint `NewPipelineWithOptions(...)` и `PipelineOptions`, сохраняя обратную совместимость.

## Основной сценарий

1. Пользователь создаёт зависимости `store`, `llm`, `embedder` (и опционально `chunker`).
2. Создаёт pipeline:
   - `p := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{...})`
3. Вызывает `p.Index`, `p.Query`, `p.Answer` как раньше, но:
   - `Answer`/`Query` используют `DefaultTopK` из options,
   - `Answer` использует `SystemPrompt` из options (если задан),
   - `Index` использует `Chunker` из options (если задан).

## Scope

- Public API (`pkg/draftrag`):
  - новый type `PipelineOptions`
  - новый конструктор `NewPipelineWithOptions(store, llm, embedder, opts) *Pipeline`
  - сохранение существующих фабрик `NewPipeline` и `NewPipelineWithChunker` (backward compatibility).
- Application (`internal/application`):
  - минимальная реорганизация для передачи конфигурации (например, system prompt) в use-case слой, если это требуется контрактом.
- Testing:
  - unit-тесты на корректное применение defaultTopK и system prompt.

## Контекст

- Сейчас дефолт `defaultTop=5` хранится в `pkg/draftrag.Pipeline`.
- System prompt v1 зашит в `internal/application` (`defaultSystemPromptV1`).
- Chunker интеграция реализована отдельным конструктором `NewPipelineWithChunker`.
- Конституция: `context.Context` первым параметром, `nil ctx` — panic, русские godoc для публичных символов, минимальные зависимости.

## Требования

- RQ-001 ДОЛЖЕН быть добавлен публичный тип `PipelineOptions` в `pkg/draftrag`.
- RQ-002 ДОЛЖЕН быть добавлен публичный конструктор `NewPipelineWithOptions(store, llm, embedder, opts) *Pipeline` в `pkg/draftrag`.
- RQ-003 Опция `DefaultTopK` ДОЛЖНА задавать значение topK по умолчанию для `Pipeline.Query` и `Pipeline.Answer`.
- RQ-004 Опция `SystemPrompt` ДОЛЖНА переопределять system prompt, используемый `Pipeline.Answer*` (если непустая строка).
- RQ-005 Опция `Chunker` ДОЛЖНА включать чанкинг в `Index` (как в `pipeline-index-with-chunker`) при `Chunker != nil`.
- RQ-006 Backward compatibility: существующие `NewPipeline` и `NewPipelineWithChunker` ДОЛЖНЫ оставаться рабочими и сохранять прежние дефолты (defaultTopK=5, system prompt v1, chunker только в соответствующей фабрике).
- RQ-007 Валидация options:
  - `DefaultTopK <= 0` -> panic (как программистская ошибка конфигурации), либо deterministic error на первом вызове — решение фиксируем в plan; в v1 предпочтительно panic при создании pipeline.
- RQ-008 Публичные типы/функции (`PipelineOptions`, `NewPipelineWithOptions`) ДОЛЖНЫ иметь godoc на русском.
- RQ-009 Тесты ДОЛЖНЫ проходить без внешней сети.

## Критерии приемки

### AC-001 Публичный конструктор и options доступны из pkg/draftrag

- **Given** пользователь импортирует `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он создаёт pipeline через `NewPipelineWithOptions(...)` и передаёт `PipelineOptions`
- **Then** код компилируется
- Evidence: compile-time тест в `pkg/draftrag`.

### AC-002 DefaultTopK применяется в Query/Answer

- **Given** pipeline создан с `PipelineOptions{DefaultTopK: 3}`
- **When** вызывается `p.Query(ctx, q)` и `p.Answer(ctx, q)`
- **Then** внутренняя логика использует topK=3 (то есть делегирует на `QueryTopK`/`AnswerTopK` с 3)
- Evidence: unit-тест с заглушками зависимостей/счётчиками.

### AC-003 SystemPrompt переопределяется

- **Given** pipeline создан с `PipelineOptions{SystemPrompt: "X"}`
- **When** вызывается `p.AnswerTopK(ctx, q, k)`
- **Then** `LLMProvider.Generate` получает `systemPrompt == "X"`
- Evidence: unit-тест, проверяющий аргументы Generate.

### AC-004 Chunker включается через options

- **Given** pipeline создан с `PipelineOptions{Chunker: someChunker}`
- **When** вызывается `p.Index(ctx, docs)`
- **Then** используется chunker путь (Chunk→Embed per chunk→Upsert per chunk)
- Evidence: unit-тест, аналогичный `pipeline-index-with-chunker`.

## Вне scope

- Сложные шаблоны промптов и параметризация под разные LLM.
- Ограничение размера контекста (MaxContextChars/MaxChunksInPrompt) — отдельная фича.

## Открытые вопросы

- Считаем ли `DefaultTopK <= 0` программной ошибкой (panic на конструкторе), или возвращаем публичную sentinel-ошибку и делаем конструктор `(...)(*Pipeline, error)`? (предпочтение: panic, чтобы не усложнять API).

