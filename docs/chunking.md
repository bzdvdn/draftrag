# Чанкинг

Чанкинг — разбиение документа на фрагменты перед индексацией. Без чанкера каждый `Document` индексируется как один чанк.

## BasicChunker

Детерминированный чанкер по рунам с поддержкой overlap и ограничения MaxChunks.

```go
chunker := draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
    ChunkSize: 500,  // рун (не байт)
    Overlap:   60,   // перекрытие между чанками
    MaxChunks: 0,    // 0 = без ограничения
})

pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    Chunker: chunker,
})
```

### BasicChunkerOptions

| Поле | Описание |
|---|---|
| `ChunkSize` | **Обязательно > 0.** Целевой размер чанка в рунах |
| `Overlap` | Перекрытие в рунах. `>= 0` и `< ChunkSize` |
| `MaxChunks` | Максимум чанков. `0` → без лимита |

### Как работает overlap

```
Документ: [AAAAAA BBBBBB CCCCCC]
ChunkSize=6, Overlap=2:

Чанк 0: [AAAAAA]
Чанк 1:   [AABB BB]   (начинается с last 2 рун предыдущего)
Чанк 2:       [BBCCCC]
```

Overlap помогает сохранить контекст на границах чанков — предложение, разбитое между двумя чанками, попадёт в каждый из них частично.

### Рекомендуемые параметры

| Сценарий | ChunkSize | Overlap |
|---|---|---|
| Технические тексты | 400–600 | 50–80 |
| Длинные статьи | 800–1200 | 100–150 |
| FAQ, короткие ответы | 200–300 | 30–50 |
| Код | 300–500 | 0–30 |

### MaxChunks

Ограничивает количество чанков из одного документа. Полезно для очень длинных документов, чтобы не создавать избыточный индекс:

```go
chunker := draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
    ChunkSize: 500,
    Overlap:   60,
    MaxChunks: 20,  // не более 20 чанков на документ
})
```

### Ошибки конфигурации

Ошибки возвращаются из `Chunk`, сопоставимы через `errors.Is(err, draftrag.ErrInvalidChunkerConfig)`:
- `ChunkSize <= 0`
- `Overlap < 0`
- `Overlap >= ChunkSize`
- `MaxChunks < 0`

## Кастомный Chunker

```go
type MyChunker struct{}

func (c *MyChunker) Chunk(ctx context.Context, doc draftrag.Document) ([]draftrag.Chunk, error) {
    // разбить doc.Content на фрагменты
    var chunks []draftrag.Chunk
    for i, part := range splitByParagraph(doc.Content) {
        chunks = append(chunks, draftrag.Chunk{
            ID:       fmt.Sprintf("%s:%d", doc.ID, i),
            Content:  part,
            ParentID: doc.ID,
            Position: i,
            Metadata: doc.Metadata,
        })
    }
    return chunks, nil
}
```

## Как Pipeline использует Chunker

При вызове `Index(ctx, docs)`:

1. Если `Chunker == nil`: каждый `Document` → один `Chunk` (Content = Document.Content)
2. Если `Chunker` задан: `Chunker.Chunk(ctx, doc)` → `[]Chunk`, затем каждый чанк эмбеддируется и сохраняется

`IndexBatch` использует тот же Chunker, но параллельно.
