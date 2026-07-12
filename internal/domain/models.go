package domain

import (
	"encoding/json"
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

	// ErrUpdateNotAtomic возвращается, если UpdateDocument завершился частично:
	// delete выполнен успешно, но переиндексация упала. Для транзакционных store
	// rollback восстановил исходные чанки; для best-effort store часть чанков
	// может быть потеряна. Ошибка предназначена для классификации через errors.Is.
	//
	// @sk-task api-consistency-pass#T1.1: введён sentinel для degraded-path UpdateDocument (RQ-005, AC-009)
	ErrUpdateNotAtomic = errors.New("update not atomic; old chunks may be partially deleted")
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

// @sk-task cost-tracking: TokenUsage для cost tracking (AC-001, RQ-001)
// TokenUsage содержит количество токенов, использованных в одном LLM-вызове.
type TokenUsage struct {
	// PromptTokens — количество токенов во входном сообщении (system + user).
	PromptTokens int64
	// CompletionTokens — количество токенов в сгенерированном ответе.
	CompletionTokens int64
	// TotalTokens — общее количество токенов (может не совпадать с суммой,
	// если API возвращает только суммарное значение).
	TotalTokens int64
}

// @sk-task cost-tracking: ModelPricing для расчёта стоимости (AC-002, RQ-002)
// ModelPricing задаёт цены за 1K токенов для модели.
type ModelPricing struct {
	// InputCostPer1K — стоимость за 1K input (prompt) токенов в USD.
	InputCostPer1K float64
	// OutputCostPer1K — стоимость за 1K output (completion) токенов в USD.
	OutputCostPer1K float64
}

// @sk-task hierarchical-indices#T1.2: ParentContent field on RetrievedChunk (AC-001, DM-001)
//
// RetrievedChunk представляет чанк с оценкой релевантности в результате поиска.
// ParentContent содержит полный текст родительского документа (пустая строка,
// если parent недоступен: store не поддерживает или ParentContextEnabled=false).
type RetrievedChunk struct {
	Chunk         Chunk
	Score         float64
	ParentContent string
}

// @sk-task cost-tracking: CostSnapshot для снапшота статистики (AC-003, RQ-003, RQ-007)
// CostSnapshot — атомарный срез накопленной статистики cost tracker'а.
type CostSnapshot struct {
	// PromptTokens — общее количество prompt токенов.
	PromptTokens int64
	// CompletionTokens — общее количество completion токенов.
	CompletionTokens int64
	// TotalTokens — общее количество токенов (prompt + completion).
	TotalTokens int64
	// TotalCost — общая стоимость всех вызовов в USD.
	TotalCost float64
	// CallsCount — количество успешных LLM-вызовов.
	CallsCount int64
}

// @sk-task cost-tracking: Diff — дельта между двумя CostSnapshot (AC-007, RQ-007)
// Diff возвращает разницу между двумя снапшотами (curr - prev).
// Если curr.TotalTokens < prev.TotalTokens, результат обнуляется
// (что может произойти при Reset между checkpoint'ами).
func Diff(prev, curr CostSnapshot) CostSnapshot {
	diff := CostSnapshot{
		PromptTokens:     curr.PromptTokens - prev.PromptTokens,
		CompletionTokens: curr.CompletionTokens - prev.CompletionTokens,
		TotalTokens:      curr.TotalTokens - prev.TotalTokens,
		TotalCost:        curr.TotalCost - prev.TotalCost,
		CallsCount:       curr.CallsCount - prev.CallsCount,
	}
	if diff.PromptTokens < 0 {
		diff.PromptTokens = 0
	}
	if diff.CompletionTokens < 0 {
		diff.CompletionTokens = 0
	}
	if diff.TotalTokens < 0 {
		diff.TotalTokens = 0
	}
	if diff.TotalCost < 0 {
		diff.TotalCost = 0
	}
	if diff.CallsCount < 0 {
		diff.CallsCount = 0
	}
	return diff
}

// @sk-task query-rewriting#T1.2: RewrittenQuery и QueryHistory (AC-001)

// RewrittenQuery представляет результат переформулировки запроса.
type RewrittenQuery struct {
	// Query — переписанный текст запроса.
	Query string

	// Weight — вес при fusion (0 — эквивалентно 1.0).
	// Зарезервировано для weighted fusion в будущем.
	Weight float64
}

// Message представляет одно сообщение в истории диалога.
type Message struct {
	// Role — отправитель: "user" или "assistant".
	Role string

	// Content — текст сообщения.
	Content string
}

// QueryHistory содержит историю предыдущих сообщений диалога для multi-turn контекста.
//
// Caller управляет жизненным циклом и размером истории. Pipeline не хранит,
// не обрезает и не персистирует QueryHistory.
type QueryHistory struct {
	Entries []Message
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

// @sk-task arch-issues#T1.1: ToolDefinition для tool calling (AC-003, AC-004)
// ToolDefinition описывает инструмент для LLM tool calling.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// @sk-task arch-issues#T1.1: ToolCall для tool calling (AC-003, AC-004)
// ToolCall представляет вызов инструмента от LLM.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// @sk-task arch-issues#T1.1: ToolResult для tool calling (AC-003, AC-004)
// ToolResult представляет результат выполнения инструмента.
type ToolResult struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Result string `json:"result"`
}
