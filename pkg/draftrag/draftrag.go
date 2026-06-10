package draftrag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bzdvdn/draftrag/internal/application"
	"github.com/bzdvdn/draftrag/internal/domain"
)

// VectorStore определяет интерфейс для работы с векторным хранилищем.
type VectorStore = domain.VectorStore

// LLMProvider определяет интерфейс для генерации текста через LLM.
type LLMProvider = domain.LLMProvider

// Embedder определяет интерфейс для преобразования текста в векторное представление.
type Embedder = domain.Embedder

// Chunker определяет интерфейс для разбиения документа на чанки.
type Chunker = domain.Chunker

// Hooks — опциональные хуки наблюдаемости для стадий pipeline.
type Hooks = domain.Hooks

// Logger — опциональный структурированный логгер для инфраструктурных событий (кэш, retry).
// nil означает no-op.
type Logger = domain.Logger

// LogLevel — уровень логирования.
type LogLevel = domain.LogLevel

// Уровни логирования.
const (
	LogLevelDebug = domain.LogLevelDebug
	LogLevelInfo  = domain.LogLevelInfo
	LogLevelWarn  = domain.LogLevelWarn
	LogLevelError = domain.LogLevelError
)

// LogField — структурированное поле лог-события.
type LogField = domain.LogField

// ParentIDFilter задаёт фильтрацию retrieval по ParentID.
type ParentIDFilter = domain.ParentIDFilter

// MetadataFilter задаёт условие точного совпадения по полям метаданных документа при поиске.
// Пустой Fields (nil или len==0) означает «без фильтра».
// Все условия применяются как AND: все пары ключ-значение из Fields должны совпасть.
//
// @ds-task T3.2: Переэкспортировать MetadataFilter из domain в публичный API (RQ-005, RQ-006, AC-003)
type MetadataFilter = domain.MetadataFilter

// VectorStoreWithFilters — опциональная capability интерфейса VectorStore, поддерживающая фильтры.
type VectorStoreWithFilters = domain.VectorStoreWithFilters

// HybridSearcher — опциональная capability интерфейса VectorStore, поддерживающая гибридный поиск (BM25 + semantic).
type HybridSearcher = domain.HybridSearcher

// HybridConfig задаёт параметры гибридного поиска (BM25 + semantic).
type HybridConfig = domain.HybridConfig

// DefaultHybridConfig возвращает конфигурацию гибридного поиска по умолчанию.
var DefaultHybridConfig = domain.DefaultHybridConfig

// Document представляет документ для индексации в RAG-системе.
type Document = domain.Document

// Chunk представляет фрагмент документа.
type Chunk = domain.Chunk

// RetrievalResult содержит результаты поиска.
type RetrievalResult = domain.RetrievalResult

// InlineCitation задаёт соответствие номера цитаты `[n]` и retrieval-источника (чанка).
type InlineCitation = domain.InlineCitation

// IndexBatchResult содержит результат batch-индексации документов.
type IndexBatchResult = domain.IndexBatchResult

// IndexBatchError представляет ошибку индексации конкретного документа при batch-индексации.
type IndexBatchError = domain.IndexBatchError

// StreamingLLMProvider — опциональная capability интерфейса LLMProvider, поддерживающая streaming.
type StreamingLLMProvider = domain.StreamingLLMProvider

// Reranker — опциональный интерфейс для переранжирования результатов retrieval.
type Reranker = domain.Reranker

// PipelineConfig — удалён. Используйте PipelineOptions.
//
// @sk-task arch-quality-pass#T1.1: re-export alias PipelineConfig → PipelineOptions (AC-004)
// @sk-task arch-quality-pass#T3.2: удалён (AC-004)

// DocumentStore — опциональная capability VectorStore для удаления по ParentID.
type DocumentStore = domain.DocumentStore

// TransactionalTx — транзакция в транзакционном vector store.
//
// @sk-task api-consistency-pass#T1.1: re-export для публичного API (RQ-005, AC-008)
type TransactionalTx = domain.TransactionalTx

// TransactionalDocumentStore — опциональная capability VectorStore, поддерживающая
// транзакционные операции для атомарного UpdateDocument.
//
// @sk-task api-consistency-pass#T1.1: re-export для публичного API (RQ-005, AC-008)
type TransactionalDocumentStore = domain.TransactionalDocumentStore

// Pipeline — публичный API для композиции core-компонентов draftRAG.
// Валидация входных данных выполняется здесь (см. errors.go).
type Pipeline struct {
	core       *application.Pipeline
	defaultTop int
}

// PipelineOptions задаёт конфигурацию Pipeline.
type PipelineOptions struct {
	// DefaultTopK — значение topK по умолчанию для Query/Answer.
	// Если 0, используется значение по умолчанию (5).
	// Если < 0, это считается ошибкой конфигурации (panic).
	DefaultTopK int
	// SystemPrompt — переопределение system prompt для Answer*. Пустая строка означает дефолт v1.
	SystemPrompt string
	// Chunker — опциональный чанкер; если не nil, Index индексирует чанки (Chunk→Embed→Upsert).
	Chunker Chunker
	// Hooks — опциональные хуки наблюдаемости для стадий pipeline (chunking/embed/search/generate).
	// nil означает no-op.
	Hooks Hooks

	// MaxContextChars — лимит размера секции “Контекст:” в prompt для Answer* (в символах).
	// 0 означает “без лимита”.
	MaxContextChars int
	// MaxContextChunks — лимит количества чанков, попадающих в секцию “Контекст:” в prompt для Answer*.
	// 0 означает “без лимита”.
	MaxContextChunks int

	// DedupByParentID включает дедупликацию retrieval sources по ParentID.
	// По умолчанию выключено (backward compatibility).
	DedupByParentID bool

	// MMREnabled включает MMR rerank/selection для retrieval sources (диверсификация контекста).
	// По умолчанию выключено (backward compatibility).
	MMREnabled bool
	// MMRLambda задаёт баланс релевантность/разнообразие в диапазоне [0..1].
	// Если 0 и MMR включён — используется значение по умолчанию (0.5).
	MMRLambda float64
	// MMRCandidatePool задаёт сколько кандидатов запросить у VectorStore до отбора.
	// Если 0 — используется topK запроса.
	MMRCandidatePool int

	// IndexConcurrency задаёт количество workers для параллельной индексации в IndexBatch.
	// Если 0 — используется значение по умолчанию (4).
	IndexConcurrency int
	// IndexBatchRateLimit задаёт максимальное количество вызовов Embed в секунду в IndexBatch.
	// Если 0 — без ограничений.
	IndexBatchRateLimit int
	// IndexBatchRateLimitPerWorker включает per-worker rate-limiter: каждый worker
	// индексирует документы с частотой IndexBatchRateLimit per second (а не весь
	// пул суммарно). Используется, когда у каждого worker'а свой внешний квотный
	// лимит (например, несколько подов с независимыми rate-limits в API-ключе).
	//
	// По умолчанию false (backward-compat): один общий ticker на пул.
	// При IndexBatchRateLimit=10 и IndexConcurrency=4:
	//   - false: 10 embed/sec суммарно на пул.
	//   - true:  10 embed/sec на каждого worker'а, т.е. 40 embed/sec суммарно.
	IndexBatchRateLimitPerWorker bool

	// StreamBufferSize задаёт ёмкость буфера канала streaming-вывода в Answer*.
	// 0 — unbuffered (backward-compat, OQ-2: токен передаётся синхронно).
	// N > 0 — producer (LLM-стрим) может обгонять consumer на N токенов
	// без блокировки; при заполнении буфера producer блокируется на send.
	// Используется для bounded backpressure между LLM и consuming кодом.
	StreamBufferSize int

	// Reranker — опциональный reranker, применяется после retrieval.
	// nil означает "без reranking".
	Reranker Reranker
}

// NewPipeline создаёт pipeline из зависимостей: VectorStore, LLMProvider и Embedder.
//
// @sk-task arch-quality-pass#T2.2: error return вместо panic (AC-002)
func NewPipeline(store VectorStore, llm LLMProvider, embedder Embedder) (*Pipeline, error) {
	return NewPipelineWithOptions(store, llm, embedder, PipelineOptions{})
}

// NewPipelineWithChunker создаёт pipeline из зависимостей: VectorStore, LLMProvider, Embedder и Chunker.
//
// При наличии Chunker метод Index будет индексировать чанки (Chunker.Chunk → Embed → Upsert).
//
// @sk-task arch-quality-pass#T2.2: error return вместо panic (AC-002)
func NewPipelineWithChunker(store VectorStore, llm LLMProvider, embedder Embedder, chunker Chunker) (*Pipeline, error) {
	core, err := application.NewPipelineWithChunker(store, llm, embedder, chunker)
	if err != nil {
		return nil, err
	}
	return &Pipeline{
		core:       core,
		defaultTop: 5,
	}, nil
}

// NewPipelineWithOptions создаёт pipeline из зависимостей: VectorStore, LLMProvider и Embedder,
// применяя конфигурацию из PipelineOptions.
//
// @sk-task arch-quality-pass#T2.2: error return вместо panic (AC-002)
func NewPipelineWithOptions(store VectorStore, llm LLMProvider, embedder Embedder, opts PipelineOptions) (*Pipeline, error) {
	defaultTop := 5
	if opts.DefaultTopK < 0 {
		return nil, fmt.Errorf("DefaultTopK must be >= 0, got %d", opts.DefaultTopK)
	}
	if opts.DefaultTopK > 0 {
		defaultTop = opts.DefaultTopK
	}
	if opts.MaxContextChars < 0 {
		return nil, fmt.Errorf("MaxContextChars must be >= 0, got %d", opts.MaxContextChars)
	}
	if opts.MaxContextChunks < 0 {
		return nil, fmt.Errorf("MaxContextChunks must be >= 0, got %d", opts.MaxContextChunks)
	}
	if opts.MMRCandidatePool < 0 {
		return nil, fmt.Errorf("MMRCandidatePool must be >= 0, got %d", opts.MMRCandidatePool)
	}
	if opts.MMRLambda < 0 || opts.MMRLambda > 1 {
		return nil, fmt.Errorf("MMRLambda must be in [0..1], got %f", opts.MMRLambda)
	}

	core, err := application.NewPipelineWithConfig(store, llm, embedder, application.PipelineOptions{
		SystemPrompt:        opts.SystemPrompt,
			Chunker:             opts.Chunker,
			Hooks:               opts.Hooks,
			MaxContextChars:     opts.MaxContextChars,
			MaxContextChunks:    opts.MaxContextChunks,
			DedupByParentID:     opts.DedupByParentID,
			MMREnabled:          opts.MMREnabled,
			MMRLambda:           opts.MMRLambda,
			MMRCandidatePool:    opts.MMRCandidatePool,
			IndexConcurrency:    opts.IndexConcurrency,
			IndexBatchRateLimit: opts.IndexBatchRateLimit,
			IndexBatchRateLimitPerWorker: opts.IndexBatchRateLimitPerWorker,
			StreamBufferSize:    opts.StreamBufferSize,
			Reranker:            opts.Reranker,
		})
	if err != nil {
		return nil, err
	}
	return &Pipeline{
		core:       core,
		defaultTop: defaultTop,
	}, nil
}

// Index индексирует документы.
func (p *Pipeline) Index(ctx context.Context, docs []Document) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, doc := range docs {
		if strings.TrimSpace(doc.Content) == "" {
			return ErrEmptyDocument
		}
	}

	err := p.core.Index(ctx, docs)
	return mapAppError(err)
}

// Query выполняет поиск с topK по умолчанию (PipelineOptions.DefaultTopK или 5).
// Для расширенных параметров используйте Search builder.
func (p *Pipeline) Query(ctx context.Context, question string) (RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return RetrievalResult{}, err
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return RetrievalResult{}, ErrEmptyQuery
	}
	result, err := p.core.Query(ctx, question, p.defaultTop)
	return result, mapAppError(err)
}

// Answer генерирует ответ с topK по умолчанию (PipelineOptions.DefaultTopK или 5).
// Для расширенных параметров используйте Search builder.
func (p *Pipeline) Answer(ctx context.Context, question string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	question = strings.TrimSpace(question)
	if question == "" {
		return "", ErrEmptyQuery
	}
	return p.core.Answer(ctx, question, p.defaultTop)
}

// UpdateDocument удаляет все чанки документа и переиндексирует его.
// Атомарности нет: при ошибке переиндексации старые чанки уже удалены.
// Требует DocumentStore capability.
func (p *Pipeline) UpdateDocument(ctx context.Context, doc Document) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(doc.Content) == "" {
		return ErrEmptyDocument
	}
	err := p.core.UpdateDocument(ctx, doc)
	if errors.Is(err, application.ErrDeleteNotSupported) {
		return ErrDeleteNotSupported
	}
	return err
}

// Retrieve выполняет поиск по вопросу с заданным topK и возвращает RetrievalResult.
// Удобен для прямой передачи в eval.Run (реализует eval.RetrievalRunner).
// Для цепочки с фильтрами, hybrid и другими параметрами используйте Search builder.
func (p *Pipeline) Retrieve(ctx context.Context, question string, topK int) (RetrievalResult, error) {
	return p.Search(question).TopK(topK).Retrieve(ctx)
}

// DeleteDocument удаляет документ и все его чанки по ID документа (ParentID).
// Требует, чтобы VectorStore реализовывал DocumentStore capability.
// Если store не поддерживает — возвращает ErrDeleteNotSupported.
func (p *Pipeline) DeleteDocument(ctx context.Context, docID string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(docID) == "" {
		return ErrEmptyDocumentID
	}
	err := p.core.DeleteDocument(ctx, docID)
	if errors.Is(err, application.ErrDeleteNotSupported) {
		return ErrDeleteNotSupported
	}
	return err
}

// ErrStreamingNotSupported возвращается, если streaming-метод вызван,
// но underlying LLMProvider не поддерживает StreamingLLMProvider capability.
var ErrStreamingNotSupported = errors.New("LLM provider does not support streaming")

// ErrHybridNotSupported возвращается, если метод гибридного поиска вызван,
// но underlying VectorStore не поддерживает HybridSearcher capability.
var ErrHybridNotSupported = errors.New("vector store does not support hybrid search")

// ErrDeleteNotSupported возвращается, если DeleteDocument вызван,
// но underlying VectorStore не реализует DocumentStore capability.
var ErrDeleteNotSupported = errors.New("vector store does not support DeleteByParentID")

// ErrEmptyDocumentID возвращается, если передан пустой ID документа.
var ErrEmptyDocumentID = errors.New("document ID must not be empty")

// IndexBatch индексирует документы параллельно с ограничением concurrency.
//
// В отличие от Index, IndexBatch обрабатывает документы конкурентно (batchSize workers)
// и возвращает partial results: успешно проиндексированные документы и ошибки отдельно.
// Ошибка одного документа не прерывает обработку остальных.
//
// batchSize — количество параллельных workers (0 → значение по умолчанию 4).
// Для управления concurrency и rate limiting используйте PipelineOptions.IndexConcurrency,
// PipelineOptions.IndexBatchRateLimit и PipelineOptions.IndexBatchRateLimitPerWorker
// при создании Pipeline.
func (p *Pipeline) IndexBatch(ctx context.Context, docs []Document, batchSize int) (*IndexBatchResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if strings.TrimSpace(doc.Content) == "" {
			return nil, ErrEmptyDocument
		}
	}

	result, err := p.core.IndexBatch(ctx, docs, batchSize)
	return result, mapAppError(err)
}

// mapAppError — единая точка маппинга application/domain ошибок на публичные sentinel'ы.
//
// @sk-task api-consistency-pass#T2.2: rename mapValidationErr → mapAppError + расширение маппинга (RQ-003, AC-005, AC-006)
// @sk-task hardening-2026q2#T3.2: Упрощение mapValidationErr (AC-010)
func mapAppError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, application.ErrStreamingNotSupported):
		return ErrStreamingNotSupported
	case errors.Is(err, application.ErrHybridNotSupported):
		return ErrHybridNotSupported
	case errors.Is(err, application.ErrFiltersNotSupported):
		return ErrFiltersNotSupported
	case errors.Is(err, application.ErrDeleteNotSupported):
		return ErrDeleteNotSupported
	case errors.Is(err, domain.ErrEmptyQueryText):
		return ErrEmptyQuery
	case errors.Is(err, domain.ErrInvalidQueryTopK):
		return ErrInvalidTopK
	case errors.Is(err, domain.ErrEmptyDocumentContent):
		return ErrEmptyDocument
	case errors.Is(err, domain.ErrEmbeddingDimensionMismatch):
		return ErrEmbeddingDimensionMismatch
	case errors.Is(err, domain.ErrUpdateNotAtomic):
		return ErrUpdateNotAtomic
	}
	return err
}
