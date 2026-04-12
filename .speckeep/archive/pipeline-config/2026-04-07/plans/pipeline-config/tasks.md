# PipelineOptions / NewPipelineWithOptions для draftRAG — Задачи

## Phase Contract

Inputs: `.speckeep/plans/pipeline-config/plan.md`, `.speckeep/plans/pipeline-config/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/draftrag.go | T1.1, T1.2 |
| internal/application/pipeline.go | T2.1 |
| pkg/draftrag/pipeline_options_test.go | T3.1 |
| internal/application/pipeline_options_test.go | T3.2 |
| domain.Chunker | T2.1, T3.2 |
| domain.LLMProvider | T2.1, T3.2 |

## Фаза 1: Публичные options и конструктор

Цель: добавить `PipelineOptions` и единый entrypoint.

- [x] T1.1 Обновить `pkg/draftrag/draftrag.go` — добавить `type PipelineOptions struct { DefaultTopK int; SystemPrompt string; Chunker Chunker }` с русским godoc и дефолтами (DefaultTopK=5, пустой SystemPrompt означает дефолт v1). Touches: pkg/draftrag/draftrag.go
- [x] T1.2 Обновить `pkg/draftrag/draftrag.go` — добавить `NewPipelineWithOptions(store, llm, embedder, opts) *Pipeline` с русским godoc; `DefaultTopK <= 0` -> panic; сохранить backward compatibility (`NewPipeline`, `NewPipelineWithChunker`). Touches: pkg/draftrag/draftrag.go

## Фаза 2: Прокидывание конфигурации в application

Цель: применить `SystemPrompt` и `Chunker` через internal config.

- [x] T2.1 Обновить `internal/application/pipeline.go` — поддержать переопределение system prompt (если opts.SystemPrompt != ""), и wiring chunker через options (Chunker != nil включает chunker path индексации). Touches: internal/application/pipeline.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC-001..AC-004 через unit-тесты.

- [x] T3.1 Создать `pkg/draftrag/pipeline_options_test.go` — тесты публичного API: AC-001 (compile-time), AC-002 (DefaultTopK влияет на Query/Answer делегирование), плюс sanity на panic при `DefaultTopK <= 0` (опционально). Touches: pkg/draftrag/pipeline_options_test.go
- [x] T3.2 Создать `internal/application/pipeline_options_test.go` — тесты use-case: AC-003 (SystemPrompt override попадает в Generate), AC-004 (Chunker через options включает chunker path). Touches: internal/application/pipeline_options_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T1.2, T3.1
- AC-002 -> T1.1, T1.2, T3.1
- AC-003 -> T2.1, T3.2
- AC-004 -> T2.1, T3.2

## Заметки

- Все тесты должны проходить без внешней сети.
- В v1 не добавляем лимит контекста; это отдельная фича.
