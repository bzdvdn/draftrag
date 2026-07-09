# Pipeline.Index с Chunker для draftRAG

## Scope Snapshot

- In scope: расширение pipeline так, чтобы индексирование документов использовало `Chunker` (разбиение `Document` на `[]Chunk`) перед вычислением embeddings и upsert в `VectorStore`.
- Out of scope: параллельное индексирование, дедупликация/хеширование чанков, re-chunking существующих данных, backfill/migrations, batch-API для embedder/vectorstore, ретраи/backoff, логирование/метрики.

## Цель

Сделать индексирование в draftRAG “реальным”: вместо модели “1 документ = 1 чанк” использовать `Chunker`, чтобы:
- улучшить retrieval качество,
- контролировать размер контекста для LLM (через `ChunkSize/Overlap/MaxChunks`),
- обеспечить совместимость с уже добавленным `BasicChunker`.

## Основной сценарий

1. Пользователь создаёт chunker: `ch := draftrag.NewBasicChunker(opts)`.
2. Создаёт pipeline с chunker: `p := draftrag.NewPipelineWithChunker(store, llm, embedder, ch)`.
3. Вызывает `p.Index(ctx, docs)`.
4. Для каждого документа pipeline:
   - вызывает `ch.Chunk(ctx, doc)` и получает чанки,
   - для каждого чанка вызывает `embedder.Embed(ctx, chunk.Content)`,
   - вызывает `store.Upsert(ctx, chunk)` с заполненным embedding.

## Scope

- Public API (`pkg/draftrag`):
  - новый конструктор `NewPipelineWithChunker(store, llm, embedder, chunker) *Pipeline`
  - поведение `(*Pipeline).Index` меняется: если chunker задан, индексируем чанки; если нет — сохраняем старое поведение (1 чанк на документ) для backward compatibility.
- Application (`internal/application`):
  - pipeline хранит `Chunker` как опциональную зависимость.
  - `Index` использует chunker при наличии.
- Testing:
  - unit-тесты без внешней сети с заглушками, проверяющие что чанкер вызывается, и upsert’ится несколько чанков.

## Контекст

- Сейчас `internal/application.Pipeline.Index` работает как “v1: один чанк на документ” и генерирует ID `fmt.Sprintf("%s#%d", doc.ID, 0)`.
- У нас уже есть `Chunker` и его реализация `BasicChunker`, но она не интегрирована в `Index`.
- Конституция: `context.Context` первым параметром, `nil` ctx — panic, минимальные зависимости, русские godoc для публичного API.

## Требования

- RQ-001 ДОЛЖЕН быть добавлен публичный конструктор `NewPipelineWithChunker(store, llm, embedder, chunker)` в `pkg/draftrag`.
- RQ-002 При использовании pipeline, созданного через `NewPipelineWithChunker`, метод `Index` ДОЛЖЕН вызывать `chunker.Chunk(ctx, doc)` для каждого документа.
- RQ-003 Для каждого возвращённого чанка `Index` ДОЛЖЕН вычислить embedding через `embedder.Embed(ctx, chunk.Content)` и upsert’ить чанк в `VectorStore` с заполненным `Chunk.Embedding`.
- RQ-004 ДОЛЖНА сохраняться backward compatibility: существующий `NewPipeline(store, llm, embedder)` и `Index` продолжают работать без chunker (поведение 1 документ = 1 чанк остаётся).
- RQ-005 `Index` ДОЛЖЕН уважать отмену/таймауты контекста и возвращать `context.Canceled`/`context.DeadlineExceeded` без оборачивания в несопоставимые ошибки.
- RQ-006 Тесты ДОЛЖНЫ проходить без внешней сети.
- RQ-007 Публичный конструктор `NewPipelineWithChunker` ДОЛЖЕН иметь godoc на русском языке.

## Критерии приемки

### AC-001 Публичный конструктор NewPipelineWithChunker доступен

- **Given** пользователь импортирует `github.com/bzdvdn/draftrag/pkg/draftrag`
- **When** он создаёт pipeline через `NewPipelineWithChunker(...)`
- **Then** код компилируется
- Evidence: compile-time тест в `pkg/draftrag`.

### AC-002 Index использует Chunker и индексирует несколько чанков

- **Given** chunker, который возвращает два чанка на документ
- **When** вызывается `p.Index(ctx, []Document{...})`
- **Then** вызываются `Chunk(ctx, doc)`, затем `Embed` для каждого чанка и `Upsert` для каждого чанка
- Evidence: unit-тест use-case или публичного слоя.

### AC-003 Backward compatibility сохранена

- **Given** pipeline, созданный через `NewPipeline(store, llm, embedder)` без chunker
- **When** вызывается `Index(ctx, docs)`
- **Then** индексируется один чанк на документ (как раньше) и тесты/публичный контракт не ломаются
- Evidence: unit-тест или сохранение существующих тестов без изменений.

### AC-004 Контекстная отмена работает

- **Given** отменённый `ctx`
- **When** вызывается `Index(ctx, docs)`
- **Then** метод возвращает `context.Canceled` не позднее чем через 100мс (в тестовом сценарии)
- Evidence: unit-тест.

## Вне scope

- Миграции существующих данных в VectorStore.
- Параллельное индексирование.
- Стриминг/батчинг embeddings.

## Открытые вопросы

- Нужна ли в v1 возможность “автоchunker по умолчанию”, или chunker должен быть только явной зависимостью через `NewPipelineWithChunker`?

