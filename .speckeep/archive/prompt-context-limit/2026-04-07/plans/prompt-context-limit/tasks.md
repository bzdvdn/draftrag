# Ограничение контекста в Prompt для draftRAG — Задачи

## Phase Contract

Inputs: `.draftspec/plans/prompt-context-limit/plan.md`, `.draftspec/plans/prompt-context-limit/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/draftrag.go | T1.1, T2.1 |
| internal/application/pipeline.go | T2.2 |
| pkg/draftrag/prompt_context_limit_test.go | T3.1 |
| internal/application/prompt_context_limit_test.go | T3.2 |
| domain.LLMProvider | T2.2, T3.2 |

## Фаза 1: Options API

Цель: расширить `PipelineOptions` лимитами контекста.

- [x] T1.1 Обновить `pkg/draftrag/draftrag.go` — добавить в `PipelineOptions` поля `MaxContextChars int` и `MaxContextChunks int` с русским godoc; `0` означает “без лимита”, `<0` — panic (ошибка конфигурации). Touches: pkg/draftrag/draftrag.go

## Фаза 2: Wiring и prompt builder

Цель: применить лимиты при построении `userMessage` в Answer*.

- [x] T2.1 Обновить `pkg/draftrag/draftrag.go` — прокинуть лимиты из `PipelineOptions` в internal config при `NewPipelineWithOptions`. Touches: pkg/draftrag/draftrag.go
- [x] T2.2 Обновить `internal/application/pipeline.go` — расширить `PipelineConfig` лимитами контекста и обновить сборку user message: применять `MaxContextChunks` и `MaxContextChars` к секции “Контекст:” (в т.ч. обрезание внутри последнего чанка). Touches: internal/application/pipeline.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC-001..AC-004.

- [x] T3.1 Создать `pkg/draftrag/prompt_context_limit_test.go` — compile-time тест, что поля options доступны (AC-001). Touches: pkg/draftrag/prompt_context_limit_test.go
- [x] T3.2 Создать `internal/application/prompt_context_limit_test.go` — unit-тесты: AC-002 (MaxContextChunks), AC-003 (MaxContextChars), AC-004 (оба лимита). Touches: internal/application/prompt_context_limit_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1
- AC-002 -> T2.2, T3.2
- AC-003 -> T2.2, T3.2
- AC-004 -> T2.2, T3.2

## Заметки

- В v1 лимитируем только секцию “Контекст:”; “Вопрос:” не ограничиваем.
