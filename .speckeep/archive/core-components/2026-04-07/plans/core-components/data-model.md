# Core компоненты пакета draftRAG — Модель данных

## Scope

- Связанные `AC-*`: AC-002, AC-004, AC-005
- Связанные `DEC-*`: DEC-001, DEC-002
- Явно укажите, если для этой фичи значимого изменения data model не требуется: создаются 5 новых domain-моделей без существующих сущностей

## Сущности

### DM-001 Document

- Назначение: представляет документ для индексации в RAG-системе
- Источник истины: создаётся клиентом пакета, передаётся в Pipeline.Index()
- Инварианты: ID уникален в рамках одной индексации; Content не может быть пустым (валидация)
- Связанные `AC-*`: AC-002, AC-005
- Связанные `DEC-*`: DEC-001
- Поля:
  - `ID` — string, required, уникальный идентификатор документа
  - `Content` — string, required, текст документа (не пустой)
  - `Metadata` — map[string]string, optional, дополнительные атрибуты (автор, источник, дата и т.д.)
  - `CreatedAt` — time.Time, required, устанавливается при создании
  - `UpdatedAt` — time.Time, required, обновляется при модификации
- Жизненный цикл:
  - создаётся клиентом или загружается из внешнего источника
  - передаётся в Pipeline.Index(ctx, []Document) для индексации
  - может быть обновлён через повторный Upsert с тем же ID
  - удаляется через VectorStore.Delete(ctx, id)
- Замечания по консистентности:
  - Metadata не должен содержать nil values
  - Content должен быть валидным UTF-8

### DM-002 Chunk

- Назначение: фрагмент документа, полученный в результате чанкинга для эмбеддинга
- Источник истины: создаётся Chunker из Document
- Инварианты: ParentID ссылается на существующий Document; Embedding nil до вычисления
- Связанные `AC-*`: AC-002, AC-004
- Связанные `DEC-*`: DEC-001, DEC-002
- Поля:
  - `ID` — string, required, уникальный идентификатор чанка
  - `Content` — string, required, текст фрагмента
  - `ParentID` — string, required, ID родительского Document
  - `Embedding` — []float64, optional, векторное представление (nil до вычисления Embedder)
  - `Position` — int, required, порядковый номер чанка в документе (начиная с 0)
- Жизненный цикл:
  - создаётся Chunker.Chunk(document) из Document
  - передаётся в Embedder.Embed(ctx, content) для вычисления Embedding
  - сохраняется в VectorStore.Upsert(ctx, chunk)
  - удаляется каскадно при удалении родительского Document
- Замечания по консистентности:
  - ParentID должен соответствовать существующему Document
  - Embedding размерность должна совпадать с ожидаемой моделью эмбеддера

### DM-003 Query

- Назначение: пользовательский запрос для поиска релевантных чанков
- Источник истины: создаётся клиентом, передаётся в Pipeline.Query()
- Инварианты: Text не может быть пустым (валидация)
- Связанные `AC-*`: AC-002, AC-005
- Связанные `DEC-*`: DEC-001
- Поля:
  - `Text` — string, required, текст запроса (не пустой)
  - `TopK` — int, optional, количество возвращаемых результатов (по умолчанию 5)
  - `Filter` — map[string]string, optional, фильтрация по metadata
- Жизненный цикл:
  - создаётся клиентом
  - передаётся в Pipeline.Query(ctx, query)
  - используется для поиска через VectorStore.Search(ctx, embedding, topK)
  - не сохраняется персистентно
- Замечания по консистентности:
  - TopK должен быть > 0
  - Filter ключи должны соответствовать ключам в Document.Metadata

### DM-004 RetrievalResult

- Назначение: результат поиска релевантных чанков для запроса
- Источник истины: возвращается VectorStore.Search()
- Инварианты: Score в диапазоне [-1, 1] для cosine similarity; Results отсортированы по убыванию Score
- Связанные `AC-*`: AC-002, AC-004, AC-005
- Связанные `DEC-*`: DEC-002, DEC-003
- Поля:
  - `Chunks` — []RetrievedChunk, required, список найденных чанков с score
  - `QueryText` — string, required, исходный текст запроса (для tracing)
  - `TotalFound` — int, required, общее количество найденных чанков до применения TopK
- Жизненный цикл:
  - создаётся VectorStore.Search(ctx, embedding, query)
  - возвращается клиенту через Pipeline.Query(ctx, query)
  - не сохраняется персистентно
- Замечания по консистентности:
  - Chunks должны быть отсортированы по Score descending
  - Score должен быть валидным float64 (не NaN, не Inf)

### DM-005 RetrievedChunk

- Назначение: чанк с附加ённым score в результате поиска
- Источник истины: создаётся VectorStore.Search() внутри RetrievalResult
- Инварианты: Score в диапазоне [-1, 1]; Chunk не nil
- Связанные `AC-*`: AC-004
- Связанные `DEC-*`: DEC-003
- Поля:
  - `Chunk` — Chunk, required, найденный чанк
  - `Score` — float64, required, оценка релевантности (cosine similarity)
- Жизненный цикл:
  - создаётся внутри VectorStore.Search()
  - возвращается в составе RetrievalResult
  - не сохраняется персистентно
- Замечания по консистентности:
  - Score должен соответствовать метрике similarity, используемой VectorStore

### DM-006 Embedding

- Назначение: векторное представление текста
- Источник истины: вычисляется Embedder.Embed()
- Инварианты: размерность фиксирована для одной модели; все значения конечны (не NaN, не Inf)
- Связанные `AC-*`: AC-002
- Связанные `DEC-*`: DEC-002
- Поля:
  - `Vector` — []float64, required, вектор чисел
  - `Dimension` — int, required, размерность вектора
  - `Model` — string, optional, название модели эмбеддера
- Жизненный цикл:
  - создаётся Embedder.Embed(ctx, text)
  - присваивается Chunk.Embedding
  - используется VectorStore.Search(ctx, embedding, topK)
  - не сохраняется персистентно отдельно от Chunk
- Замечания по консистентности:
  - Dimension должен совпадать с ожидаемой размерностью модели
  - Vector длина должна равняться Dimension

## Связи

- `DM-001 (Document) -> DM-002 (Chunk)`: один Document порождает много Chunk через Chunker; ownership: Document владеет Chunk (каскадное удаление)
- `DM-002 (Chunk) -> DM-006 (Embedding)`: Chunk имеет один Embedding (опционально); Embedding не существует без Chunk
- `DM-003 (Query) -> DM-004 (RetrievalResult)`: один Query порождает один RetrievalResult
- `DM-004 (RetrievalResult) -> DM-005 (RetrievedChunk)`: RetrievalResult содержит много RetrievedChunk; lifetime绑定 к RetrievalResult

## Производные правила

- Chunk генерируется из Document через Chunker.Chunk() с параметрами (maxChunkSize, overlap)
- Embedding вычисляется из Chunk.Content через Embedder.Embed()
- RetrievedChunk.Score вычисляется через cosine_similarity(query_embedding, chunk.embedding)
- RetrievalResult.Chunks сортируется по Score descending

## Переходы состояний

- Chunk: Created (Embedding == nil) -> Embedded (Embedding != nil) через Embedder.Embed()
- Document: Transient -> Indexed после успешного Pipeline.Index()

## Вне scope

- Персистентное хранение документов и чанков (определяется конкретными реализациями VectorStore)
- Версионирование документов
- Полнотекстовый индекс (только vector search)
- Soft delete и архивация
