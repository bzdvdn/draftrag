# Pipeline API

## Конструкторы

### NewPipeline

```go
func NewPipeline(store VectorStore, llm LLMProvider, embedder Embedder) *Pipeline
```

Минимальная конфигурация: `DefaultTopK = 5`, без чанкинга (1 документ = 1 чанк).

### NewPipelineWithChunker

```go
func NewPipelineWithChunker(store VectorStore, llm LLMProvider, embedder Embedder, chunker Chunker) *Pipeline
```

### NewPipelineWithOptions

```go
func NewPipelineWithOptions(store VectorStore, llm LLMProvider, embedder Embedder, opts PipelineOptions) *Pipeline
```

## PipelineOptions

```go
type PipelineOptions struct {
    // DefaultTopK — количество чанков для поиска по умолчанию. 0 → 5. <0 → panic.
    DefaultTopK int

    // SystemPrompt — переопределение системного промпта для Answer*.
    // Пустая строка — встроенный промпт v1.
    SystemPrompt string

    // Chunker — если задан, Index разбивает документы на чанки перед индексацией.
    Chunker Chunker

    // Hooks — хуки наблюдаемости. nil → no-op.
    Hooks Hooks

    // MaxContextChars — лимит секции "Контекст" в prompt (символов). 0 → без лимита.
    MaxContextChars int
    // MaxContextChunks — лимит количества чанков в контексте. 0 → без лимита.
    MaxContextChunks int

    // DedupSourcesByParentID — дедупликация чанков по ParentID в RetrievalResult.
    DedupSourcesByParentID bool

    // MMREnabled — включить MMR reranking (диверсификация контекста).
    MMREnabled bool
    // MMRLambda — баланс релевантность/разнообразие [0..1]. 0 → 0.5.
    MMRLambda float64
    // MMRCandidatePool — сколько кандидатов запросить до MMR-отбора. 0 → topK.
    MMRCandidatePool int

    // IndexConcurrency — workers для IndexBatch. 0 → 4.
    IndexConcurrency int
    // IndexBatchRateLimit — макс. вызовов Embed/сек в IndexBatch. 0 → без лимита.
    IndexBatchRateLimit int
}
```

## Методы индексации

### Index

```go
func (p *Pipeline) Index(ctx context.Context, docs []Document) error
```

Последовательная индексация. Если задан `Chunker` — каждый документ разбивается на чанки. Ошибка одного документа прерывает всю операцию.

### IndexBatch

```go
func (p *Pipeline) IndexBatch(ctx context.Context, docs []Document, batchSize int) (*IndexBatchResult, error)
```

Параллельная индексация с `batchSize` workers. Ошибки отдельных документов не прерывают обработку остальных.

```go
type IndexBatchResult struct {
    Successful     []Document       // успешно проиндексированные
    Errors         []IndexBatchError // ошибки по документам
    ProcessedCount int               // всего обработано
}

type IndexBatchError struct {
    DocumentID string
    Error      error
}
```

```go
result, err := pipeline.IndexBatch(ctx, docs, 8)
if err != nil {
    return err // системная ошибка (контекст отменён и т.д.)
}
if len(result.Errors) > 0 {
    for _, e := range result.Errors {
        log.Printf("doc %s failed: %v", e.DocumentID, e.Error)
    }
}
fmt.Printf("indexed %d/%d docs\n", len(result.Successful), result.ProcessedCount)
```

## Search Builder

Основной API для поиска и генерации ответов — fluent builder. Создаётся через `pipeline.Search(question)`, параметры задаются цепочкой вызовов, запрос выполняется терминальным методом.

```go
// Создать builder
b := pipeline.Search("вопрос")

// Параметры (все опциональны)
b.TopK(5)                                        // количество чанков (default: PipelineOptions.DefaultTopK)
b.ParentIDs("doc-1", "doc-2")                    // поиск только в этих документах
b.Filter(draftrag.MetadataFilter{                // фильтр по метаданным (AND)
    Fields: map[string]string{"lang": "ru"},
})
b.Hybrid(draftrag.DefaultHybridConfig())         // hybrid BM25 + semantic
```

### Терминальные методы

| Метод | Возвращает | Описание |
|---|---|---|
| `Retrieve(ctx)` | `(RetrievalResult, error)` | Только поиск, без генерации |
| `Answer(ctx)` | `(string, error)` | RAG-ответ |
| `Cite(ctx)` | `(string, RetrievalResult, error)` | Ответ + источники со score |
| `InlineCite(ctx)` | `(string, RetrievalResult, []InlineCitation, error)` | Ответ с `[n]` цитатами в тексте |
| `Stream(ctx)` | `(<-chan string, error)` | Streaming-ответ токен за токеном |
| `StreamCite(ctx)` | `(<-chan string, RetrievalResult, []InlineCitation, error)` | Streaming с inline-цитатами |

### Примеры

```go
// Простой поиск
result, err := pipeline.Search("вопрос").TopK(5).Retrieve(ctx)

// RAG-ответ с hybrid search
answer, err := pipeline.Search("вопрос").TopK(5).Hybrid(draftrag.DefaultHybridConfig()).Answer(ctx)

// Ответ с цитатами, ограниченный одним документом
answer, sources, citations, err := pipeline.Search("вопрос").
    TopK(5).
    ParentIDs("doc-1").
    InlineCite(ctx)

// Streaming для HTTP SSE
tokens, err := pipeline.Search("вопрос").TopK(5).Stream(ctx)

// Streaming с цитатами (sources/citations готовы сразу)
tokens, sources, citations, err := pipeline.Search("вопрос").TopK(5).StreamCite(ctx)
```

## Простые методы (без параметров)

Для базового использования с `DefaultTopK`:

```go
func (p *Pipeline) Query(ctx context.Context, question string) (RetrievalResult, error)
func (p *Pipeline) Answer(ctx context.Context, question string) (string, error)
func (p *Pipeline) Retrieve(ctx context.Context, question string, topK int) (RetrievalResult, error)
```

`Retrieve` реализует `eval.RetrievalRunner` и используется в eval harness напрямую.

## DeleteDocument

Удаляет документ и все его чанки из хранилища по ParentID.

```go
err := pipeline.DeleteDocument(ctx, "doc-id")
if errors.Is(err, draftrag.ErrDeleteNotSupported) {
    // хранилище не поддерживает удаление по ParentID
}
```

Поддерживается всеми хранилищами: **InMemoryStore**, **pgvector**, **Qdrant**, **ChromaDB**.

Каждый использует нативный batch-delete по полю `parent_id`:
- InMemoryStore — итерация по карте
- pgvector — `DELETE FROM ... WHERE parent_id = $1`
- Qdrant — filter API (`must: key=parent_id`)
- ChromaDB — where-фильтр (`{"parent_id": "..."}`)

`ErrDeleteNotSupported` возвращается только для кастомных реализаций `VectorStore`, не реализующих `DocumentStore`.
