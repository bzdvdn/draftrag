# Pipeline.Index с Chunker для draftRAG — Задачи

## Phase Contract

Inputs: `.speckeep/plans/pipeline-index-with-chunker/plan.md`, `.speckeep/plans/pipeline-index-with-chunker/data-model.md`
Outputs: упорядоченные исполнимые задачи с покрытием критериев
Stop if: задачи получаются расплывчатыми или coverage по AC не удаётся сопоставить

## Surface Map

| Surface | Tasks |
|---------|-------|
| pkg/draftrag/draftrag.go | T1.1 |
| internal/application/pipeline.go | T2.1 |
| pkg/draftrag/pipeline_chunker_test.go | T3.1 |
| internal/application/pipeline_chunker_test.go | T3.2 |
| domain.Chunker | T2.1, T3.2 |
| domain.VectorStore | T2.1, T3.2 |
| domain.Embedder | T2.1, T3.2 |

## Фаза 1: Публичный конструктор

Цель: добавить entrypoint для pipeline с chunker.

- [x] T1.1 Обновить `pkg/draftrag/draftrag.go` — добавить `NewPipelineWithChunker(store, llm, embedder, chunker) *Pipeline` с русским godoc; сохранять старую фабрику `NewPipeline` без изменений. Touches: pkg/draftrag/draftrag.go

## Фаза 2: Интеграция Chunker в Index (application)

Цель: реализовать chunker путь и сохранить legacy путь.

- [x] T2.1 Обновить `internal/application/pipeline.go` — расширить struct `Pipeline` полем `chunker domain.Chunker` (optional), обновить `NewPipeline` (или добавить новый конструктор в application слое), обновить `Index`: если `chunker != nil` → `Chunk(ctx, doc)` → для каждого чанка `Embed(chunk.Content)` → заполнить `Embedding` → `Chunk.Validate()` → `Upsert`; иначе оставить legacy поведение (1 чанк на документ). Touches: internal/application/pipeline.go

## Фаза 3: Тестирование (без внешней сети)

Цель: подтвердить AC и backward compatibility.

- [x] T3.1 Создать `pkg/draftrag/pipeline_chunker_test.go` — тесты публичного API: AC-001 (compile-time/создание через `NewPipelineWithChunker`). Touches: pkg/draftrag/pipeline_chunker_test.go
- [x] T3.2 Создать `internal/application/pipeline_chunker_test.go` — unit-тесты: AC-002 (chunker возвращает 2 чанка → 2×Embed + 2×Upsert), AC-003 (pipeline без chunker индексирует 1 чанк на документ, сохраняя прежний контракт), AC-004 (ctx cancel ≤ 100мс). Touches: internal/application/pipeline_chunker_test.go

## Покрытие критериев приемки

- AC-001 -> T1.1, T3.1
- AC-002 -> T2.1, T3.2
- AC-003 -> T2.1, T3.2
- AC-004 -> T2.1, T3.2

## Заметки

- Все тесты должны проходить без внешней сети.
- На chunker пути embedding считается по `chunk.Content` (не по `doc.Content`).
