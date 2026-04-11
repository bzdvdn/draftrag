# Концепции

## Обзор архитектуры

draftRAG состоит из четырёх компонентов-интерфейсов и Pipeline, который их связывает:

```
Document → [Chunker] → Chunk → [Embedder] → vector → [VectorStore]
                                                              ↓
Question → [Embedder] → vector → [VectorStore].Search → Chunks → [LLM] → Answer
```

## Document

Единица индексации. Содержит текст и опциональные метаданные.

```go
type Document struct {
    ID       string            // уникальный идентификатор
    Content  string            // текстовое содержимое (обязательно)
    Metadata map[string]string // произвольные поля для фильтрации
}
```

`ID` используется как `ParentID` для всех чанков, созданных из этого документа. Это позволяет связать чанки с источником и фильтровать поиск по документам.

## Chunk

Фрагмент документа, который реально хранится в VectorStore.

```go
type Chunk struct {
    ID        string            // уникальный ID чанка (обычно "docID:position")
    Content   string            // текст фрагмента
    ParentID  string            // ID родительского документа
    Position  int               // порядковый номер в документе
    Embedding []float64         // вектор (заполняется Pipeline)
    Metadata  map[string]string // метаданные (наследуются от Document)
}
```

## VectorStore

Хранит чанки и выполняет векторный поиск.

```go
type VectorStore interface {
    Upsert(ctx context.Context, chunk Chunk) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, embedding []float64, topK int) (RetrievalResult, error)
}
```

Расширенные возможности реализуются через дополнительные интерфейсы:

| Интерфейс | Что добавляет |
|---|---|
| `VectorStoreWithFilters` | `SearchWithFilter` (по ParentID), `SearchWithMetadataFilter` |
| `HybridSearcher` | `SearchHybrid` (BM25 + semantic) |

Pipeline автоматически определяет поддерживаемые возможности через type assertion.

## Embedder

Преобразует текст в числовой вектор.

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float64, error)
}
```

Один и тот же Embedder используется и для индексации (вопрос → вектор), и для поиска (документ → вектор). **Важно**: использовать одну и ту же модель для обеих операций.

## LLMProvider

Генерирует ответ на основе системного промпта и пользовательского сообщения.

```go
type LLMProvider interface {
    Generate(ctx context.Context, systemPrompt, userMessage string) (string, error)
}
```

Для streaming добавляется:

```go
type StreamingLLMProvider interface {
    LLMProvider
    GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error)
}
```

Pipeline проверяет, реализует ли LLM `StreamingLLMProvider`, и возвращает `ErrStreamingNotSupported` если нет.

## Chunker

Разбивает Document на Chunk'и.

```go
type Chunker interface {
    Chunk(ctx context.Context, doc Document) ([]Chunk, error)
}
```

Если `Chunker` не задан в `PipelineOptions`, каждый документ индексируется как один чанк.

## Pipeline

Оркестрирует все компоненты. Создаётся один раз, используется многократно.

```go
// Минимальная конфигурация
pipeline := draftrag.NewPipeline(store, llm, embedder)

// Полная конфигурация
pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
    DefaultTopK:            5,
    Chunker:                myChunker,
    SystemPrompt:           "Ты — помощник...",
    MaxContextChars:        4000,
    MaxContextChunks:       10,
    DedupSourcesByParentID: true,
    MMREnabled:             true,
    MMRLambda:              0.6,
    Hooks:                  myHooks,
})
```

## RetrievalResult

Возвращается методами `Query*` и передаётся в ответах с цитатами.

```go
type RetrievalResult struct {
    Chunks     []RetrievedChunk
    TotalFound int
}

type RetrievedChunk struct {
    Chunk Chunk
    Score float64  // cosine similarity [0, 1] или RRF score для hybrid
}
```

## Обработка ошибок

Все публичные ошибки — sentinel values, сравниваемые через `errors.Is`:

```go
var (
    ErrEmptyDocument            // пустой документ при индексации
    ErrEmptyQuery               // пустой вопрос
    ErrInvalidTopK              // topK <= 0
    ErrFiltersNotSupported      // store не поддерживает фильтры
    ErrStreamingNotSupported    // LLM не поддерживает streaming
    ErrHybridNotSupported       // store не поддерживает hybrid search
    ErrEmbeddingDimensionMismatch
    ErrInvalidEmbedderConfig
    ErrInvalidLLMConfig
    ErrInvalidChunkerConfig
    ErrInvalidVectorStoreConfig
)
```
