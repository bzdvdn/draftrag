package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrEmptyDocumentID возвращается при попытке валидировать документ без ID.
	ErrEmptyDocumentID = errors.New("document id is empty")
	// ErrEmptyDocumentContent возвращается при попытке валидировать документ без содержимого.
	ErrEmptyDocumentContent = errors.New("document content is empty")
	// ErrEmptyChunkID возвращается при попытке валидировать чанк без ID.
	ErrEmptyChunkID = errors.New("chunk id is empty")
	// ErrEmptyChunkContent возвращается при попытке валидировать чанк без содержимого.
	ErrEmptyChunkContent = errors.New("chunk content is empty")
	// ErrEmptyChunkParentID возвращается при попытке валидировать чанк без ParentID.
	ErrEmptyChunkParentID = errors.New("chunk parent id is empty")
	// ErrEmptyQueryText возвращается при попытке валидировать запрос без текста.
	ErrEmptyQueryText = errors.New("query text is empty")
	// ErrInvalidQueryTopK возвращается при попытке валидировать TopK <= 0.
	ErrInvalidQueryTopK = errors.New("query topK must be > 0")

	// ErrFilterNotSupported возвращается, если pipeline-метод с MetadataFilter вызван,
	// а underlying VectorStore не реализует VectorStoreWithFilters.
	ErrFilterNotSupported = errors.New("vector store does not support metadata filter")

	// ErrEmbeddingDimensionMismatch возвращается, если размерность embedding-вектора не соответствует ожидаемой.
	//
	// Ошибка предназначена для классификации через errors.Is.
	ErrEmbeddingDimensionMismatch = errors.New("embedding dimension mismatch")

	// ErrInvalidHybridConfig возвращается при невалидной конфигурации гибридного поиска.
	ErrInvalidHybridConfig = errors.New("invalid hybrid config")
)

// Document представляет документ для индексации в RAG-системе.
type Document struct {
	ID        string
	Content   string
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate проверяет инварианты Document.
func (d Document) Validate() error {
	if d.ID == "" {
		return ErrEmptyDocumentID
	}
	if d.Content == "" {
		return ErrEmptyDocumentContent
	}
	return nil
}

// Chunk представляет фрагмент документа, полученный в результате чанкинга.
//
// @ds-task T1.2: Добавить поле Metadata в Chunk (DEC-005)
type Chunk struct {
	ID        string
	Content   string
	ParentID  string
	Embedding []float64
	Position  int
	// Metadata хранит произвольные метаданные чанка, унаследованные от родительского документа.
	// nil означает отсутствие метаданных и не влияет на результат Validate.
	Metadata map[string]string
}

// Validate проверяет инварианты Chunk.
func (c Chunk) Validate() error {
	if c.ID == "" {
		return ErrEmptyChunkID
	}
	if c.Content == "" {
		return ErrEmptyChunkContent
	}
	if c.ParentID == "" {
		return ErrEmptyChunkParentID
	}
	return nil
}

// MetadataFilter задаёт условие точного совпадения по полям метаданных документа при поиске.
// Пустой Fields (nil или len==0) означает «без фильтра» — поведение идентично поиску без фильтра.
// Все условия применяются как AND: все пары ключ-значение из Fields должны совпасть.
//
// @ds-task T1.1: Добавить тип MetadataFilter в domain (RQ-001, DEC-001)
type MetadataFilter struct {
	// Fields — карта имён полей метаданных и их ожидаемых строковых значений.
	Fields map[string]string
}

// Query представляет пользовательский запрос для поиска.
type Query struct {
	Text   string
	TopK   int
	Filter map[string]string
}

// Validate проверяет инварианты Query.
func (q Query) Validate() error {
	if q.Text == "" {
		return ErrEmptyQueryText
	}
	if q.TopK <= 0 {
		return ErrInvalidQueryTopK
	}
	return nil
}

// RetrievalResult содержит результаты поиска по запросу.
type RetrievalResult struct {
	Chunks     []RetrievedChunk
	QueryText  string
	TotalFound int
}

// RetrievedChunk представляет чанк с оценкой релевантности в результате поиска.
type RetrievedChunk struct {
	Chunk Chunk
	Score float64
}

// InlineCitation задаёт детерминированный маппинг номера цитаты (используется как `[n]`)
// на конкретный retrieval-источник (чанк + score).
//
// Нумерация начинается с 1 и соответствует порядку источников в prompt.
type InlineCitation struct {
	Number int
	Chunk  RetrievedChunk
}

// Embedding представляет векторное представление текста.
type Embedding struct {
	Vector    []float64
	Dimension int
	Model     string
}

// HybridConfig задаёт параметры гибридного поиска (BM25 + semantic).
type HybridConfig struct {
	// SemanticWeight вес семантического скора (0.0 - 1.0).
	// BM25Weight вычисляется как 1.0 - SemanticWeight.
	// При значении 0.0 используется только BM25, при 1.0 — только семантический.
	// Default: 0.7
	SemanticWeight float64

	// UseRRF если true, используется Reciprocal Rank Fusion вместо weighted score.
	// При UseRRF=true поле SemanticWeight игнорируется.
	// Default: true
	UseRRF bool

	// RRFK константа для RRF-формулы: score = 1/(k + rank).
	// Default: 60
	RRFK int

	// BMFinalK количество результатов, возвращаемых после fusion.
	// Должно быть <= topK.
	// Default: равно topK (0 означает "использовать topK")
	BMFinalK int
}

// Validate проверяет инварианты HybridConfig.
func (c HybridConfig) Validate() error {
	if c.SemanticWeight < 0 || c.SemanticWeight > 1 {
		return fmt.Errorf("%w: SemanticWeight must be in [0,1], got %f", ErrInvalidHybridConfig, c.SemanticWeight)
	}
	if c.RRFK < 1 {
		return fmt.Errorf("%w: RRFK must be > 0, got %d", ErrInvalidHybridConfig, c.RRFK)
	}
	if c.BMFinalK < 0 {
		return fmt.Errorf("%w: BMFinalK must be >= 0, got %d", ErrInvalidHybridConfig, c.BMFinalK)
	}
	return nil
}

// DefaultHybridConfig возвращает конфигурацию гибридного поиска по умолчанию.
func DefaultHybridConfig() HybridConfig {
	return HybridConfig{
		SemanticWeight: 0.7,
		UseRRF:         true,
		RRFK:           60,
		BMFinalK:       0, // 0 означает "использовать topK"
	}
}

// IndexBatchResult содержит результат batch-индексации документов.
//
// @ds-task T1.1: Добавить тип IndexBatchResult для возврата результатов batch-индексации (AC-003, AC-004)
type IndexBatchResult struct {
	// Successful — документы, успешно проиндексированные (все чанки сохранены).
	Successful []Document
	// Errors — ошибки по документам (partial failure).
	Errors []IndexBatchError
	// ProcessedCount — общее количество обработанных документов (успешных + с ошибками).
	ProcessedCount int
}

// IndexBatchError представляет ошибку индексации конкретного документа.
//
// @ds-task T1.1: Добавить тип IndexBatchError для идентификации failed документов (AC-003)
type IndexBatchError struct {
	// DocumentID — идентификатор документа, который не удалось проиндексировать.
	DocumentID string
	// Error — оригинальная ошибка (embed, chunking или upsert).
	Error error
}
