# Pipeline.Index с Chunker для draftRAG — План

## Phase Contract

Inputs: `.speckeep/specs/pipeline-index-with-chunker/spec.md`, `.speckeep/specs/pipeline-index-with-chunker/inspect.md`, конституция проекта.
Outputs: `plan.md`, `data-model.md` (contracts/research не требуются).
Stop if: невозможно сохранить backward compatibility и при этом интегрировать chunker без расплывчатости контракта.

## Цель

Интегрировать `Chunker` в индексирование документов: при наличии chunker `Index` должен индексировать чанки (Embed+Upsert на каждый чанк), а при отсутствии chunker — сохранить прежнее поведение “1 документ = 1 чанк”.

## Scope

- Public API: новый конструктор `NewPipelineWithChunker` в `pkg/draftrag`.
- Application: pipeline хранит опциональный `Chunker` и использует его в `Index`.
- Backward compatibility: `NewPipeline` без chunker остаётся прежним.
- Testing: unit-тесты с заглушками и in-memory store.

## Implementation Surfaces

- `pkg/draftrag/draftrag.go` — добавить `NewPipelineWithChunker(store, llm, embedder, chunker) *Pipeline` + godoc (T1.1).
- `internal/application/pipeline.go` — расширить struct `Pipeline` полем `chunker domain.Chunker` (optional) и обновить `Index` (T2.1).
- `pkg/draftrag/pipeline_chunker_test.go` — тесты публичного API: compile-time/конструктор, базовый сценарий индексации чанков через chunker (T3.1).
- `internal/application/pipeline_chunker_test.go` — тесты use-case: chunker вызывается, upsert’ится несколько чанков; проверка backward compatibility (T3.2).
- `domain.Chunker`, `domain.Embedder`, `domain.VectorStore` — зависимости, используются заглушки/реализации memory store в тестах.

## Влияние на архитектуру

- Application слой расширяется новой зависимостью `Chunker`, но остаётся чистым (domain интерфейсы).
- Публичный API расширяется аддитивно новым конструктором.
- Никаких миграций: меняется только поведение индексирования для pipeline, созданных с chunker.

## Acceptance Approach

- AC-001 -> compile-time/создание pipeline через `NewPipelineWithChunker` в `pkg/draftrag/pipeline_chunker_test.go`.
- AC-002 -> unit-тест: chunker возвращает 2 чанка, проверяем 2×Embed и 2×Upsert. Surface: `internal/application/pipeline_chunker_test.go`.
- AC-003 -> существующие тесты `Index` без chunker не ломаются; добавляем явный unit-тест, что индексируется 1 чанк на документ при `NewPipeline(...)`. Surface: `internal/application/pipeline_chunker_test.go`.
- AC-004 -> unit-тест отмены ctx (cancel) возвращает `context.Canceled` ≤ 100мс. Surface: `internal/application/pipeline_chunker_test.go`.

## Данные и контракты

- Persisted data model не меняется.
- Контракт `Chunk`:
  - если используется chunker: чанк приходит с `ParentID/Position/ID/Content`, embedder заполняет `Embedding` перед `Upsert`.
  - если chunker отсутствует: сохраняем legacy ID `fmt.Sprintf(\"%s#%d\", doc.ID, 0)` и content = `doc.Content`.

## Стратегия реализации

- DEC-001 Chunker как optional dependency pipeline
  Why: сохраняем минимальную конфигурацию и backward compatibility.
  Tradeoff: два пути исполнения `Index`.
  Affects: `internal/application/pipeline.go`, `pkg/draftrag/draftrag.go`
  Validation: unit-тесты AC-002/AC-003.

- DEC-002 Embed применяется к `chunk.Content` на каждый чанк
  Why: корректный embeddings per-chunk для retrieval.
  Tradeoff: больше вызовов embedder.
  Affects: `internal/application/pipeline.go`
  Validation: unit-тест AC-002.

## Incremental Delivery

### MVP (Первая ценность)

- `NewPipelineWithChunker` + интеграция chunker в `Index` + unit-тесты AC-001..AC-004.

### Итеративное расширение

- (Out of scope) batch embeddings/Upsert.
- (Out of scope) дедупликация чанков.

## Порядок реализации

1. Application: расширить `internal/application.Pipeline` (поле chunker) и обновить `Index`.
2. Public: добавить `NewPipelineWithChunker` в `pkg/draftrag`.
3. Добавить unit-тесты на chunker path и backward compatibility.

## Риски

- Риск 1: изменения в `Index` могут сломать существующие тесты/контракты.
  Mitigation: явный путь backward compatibility + отдельный тест AC-003.
- Риск 2: chunker может возвращать чанки с невалидными полями.
  Mitigation: `Index` валидирует `Chunk.Validate()` перед `Upsert` (как и ранее).

## Rollout и compatibility

- Rollout не требуется.
- Compatibility сохраняется за счёт `NewPipeline` и поведения `Index` без chunker.

## Проверка

- `go test ./...`
- Unit-тесты по AC-001..AC-004.

## Соответствие конституции

- нет конфликтов: `context.Context` первым параметром, зависимости — интерфейсы domain, тестируемость через заглушки/мemory store.

