# PipelineOptions / NewPipelineWithOptions для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: `AC-002`, `AC-003`, `AC-004`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`, `DEC-003`
- Persisted data model не меняется: добавляется только конфигурация pipeline при создании.

## Сущности

### DM-001 PipelineOptions (публичная конфигурация)

- Назначение: управлять дефолтами и опциональными зависимостями pipeline через один конструктор.
- Источник истины: `pkg/draftrag` (`PipelineOptions`).
- Инварианты:
  - `DefaultTopK > 0` (если задан; в v1 считаем `<=0` ошибкой конфигурации).
  - `SystemPrompt` может быть пустым (тогда используется дефолт).
  - `Chunker` может быть nil (тогда используется legacy индексирование).
- Связанные `AC-*`: `AC-002`, `AC-003`, `AC-004`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`, `DEC-003`
- Поля:
  - `DefaultTopK` — `int`, optional, default `5`.
  - `SystemPrompt` — `string`, optional, default пусто (использовать дефолт v1).
  - `Chunker` — `draftrag.Chunker`, optional, default nil.
- Жизненный цикл:
  - создаётся пользователем при вызове `NewPipelineWithOptions`.
  - не изменяется после создания pipeline (immutable-by-convention).

### DM-002 PipelineInternalConfig (internal конфигурация application use-case)

- Назначение: минимальные значения, которые application слой должен знать для выполнения use-case (например, system prompt).
- Источник истины: `internal/application` (поля в struct Pipeline или отдельный internal config struct).
- Инварианты:
  - `systemPrompt` всегда непустой на момент вызова `LLM.Generate` (либо дефолт, либо override).
- Связанные `AC-*`: `AC-003`
- Связанные `DEC-*`: `DEC-001`
- Поля:
  - `systemPrompt` — `string`, required.
  - `chunker` — `domain.Chunker`, optional (используется в Index).

## Связи

- `DM-001 -> DM-002`: `NewPipelineWithOptions` маппит публичные options в internal config.

## Производные правила

- Если `PipelineOptions.SystemPrompt != ""`, он переопределяет дефолтный system prompt.
- Если `PipelineOptions.Chunker != nil`, Index идёт по chunker пути.
- `DefaultTopK` применяется только в convenience методах `Query`/`Answer`.

## Переходы состояний

- Не применимо: pipeline stateless.

## Вне scope

- Глобальная конфигурация через env vars/файлы.
- Горячая смена конфигурации на лету.

