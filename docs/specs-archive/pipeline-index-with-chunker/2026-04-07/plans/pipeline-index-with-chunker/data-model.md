# Pipeline.Index с Chunker для draftRAG — Модель данных

## Scope

- Связанные `AC-*`: `AC-002`, `AC-003`
- Связанные `DEC-*`: `DEC-001`, `DEC-002`
- Persisted data model не меняется: добавляется только optional dependency `Chunker` и вычисляемые embeddings на chunk.Content.

## Сущности

### DM-001 ChunkingPath (вычисляемый путь Index)

- Назначение: выбирать путь индексирования (legacy vs chunker) на основании наличия `Chunker`.
- Источник истины: application слой (`internal/application.Pipeline`).
- Инварианты:
  - если `chunker != nil`, `Index` использует `chunker.Chunk(ctx, doc)` и индексирует каждый returned chunk;
  - если `chunker == nil`, `Index` ведёт себя как раньше: один чанк на документ.
- Связанные `AC-*`: `AC-002`, `AC-003`
- Связанные `DEC-*`: `DEC-001`
- Поля:
  - `chunker` — `domain.Chunker`, optional.
- Жизненный цикл:
  - задаётся при создании pipeline (через соответствующий конструктор).

### DM-002 IndexedChunk (вычисляемый доменный Chunk перед Upsert)

- Назначение: доменный чанк, который будет сохранён в `VectorStore` после вычисления embedding.
- Источник истины: либо chunker output (`Chunk.Content/ID/ParentID/Position`), либо legacy mapping (doc→chunk).
- Инварианты:
  - `Chunk.Validate()` должен проходить перед `Upsert`.
  - `Embedding` вычисляется через `Embedder.Embed(ctx, chunk.Content)` и записывается в `Chunk.Embedding`.
- Связанные `AC-*`: `AC-002`
- Связанные `DEC-*`: `DEC-002`
- Поля:
  - `ID` — `string`, required.
  - `ParentID` — `string`, required.
  - `Position` — `int`, required.
  - `Content` — `string`, required.
  - `Embedding` — `[]float64`, derived.

## Связи

- `DM-001 -> DM-002`: выбранный путь определяет источник `Chunk` данных.

## Производные правила

- На chunker пути: для каждого returned chunk вычисляется embedding по `chunk.Content`, затем выполняется `Upsert` этого же чанка с заполненным `Embedding`.
- На legacy пути: создаётся `Chunk{ID: fmt.Sprintf(\"%s#%d\", doc.ID, 0), Content: doc.Content, ParentID: doc.ID, Position:0}` и индексируется как раньше.

## Переходы состояний

- Не применимо: pipeline stateless.

## Вне scope

- Миграция ранее сохранённых чанков и переиндексация.
- Параллельное индексирование и batch processing.

