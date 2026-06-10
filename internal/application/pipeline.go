package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Sentinel errors returned by Pipeline operations.
var (
	ErrFiltersNotSupported = errors.New("vector store does not support filters")

	ErrStreamingNotSupported = errors.New("LLM provider does not support streaming")

	ErrDeleteNotSupported = errors.New("vector store does not support DeleteByParentID")
)

// PipelineOptions configures a Pipeline behaviour.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task arch-quality-pass#T3.2: единый struct конфигурации (AC-004)
type PipelineOptions struct {
	SystemPrompt                 string
	Chunker                      domain.Chunker
	MaxContextChars              int
	MaxContextChunks             int
	DedupByParentID              bool
	MMREnabled                   bool
	MMRLambda                    float64
	MMRCandidatePool             int
	Hooks                        domain.Hooks
	IndexConcurrency             int
	IndexBatchRateLimit          int
	IndexBatchRateLimitPerWorker bool
	StreamBufferSize             int
	Reranker                     domain.Reranker
}

// Pipeline is the core RAG pipeline coordinating store, LLM, and embedder.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
type Pipeline struct {
	store                        domain.VectorStore
	llm                          domain.LLMProvider
	embedder                     domain.Embedder
	chunker                      domain.Chunker
	systemPrompt                 string
	maxContextChars              int
	maxContextChunks             int
	dedupByParentID              bool
	mmrEnabled                   bool
	mmrLambda                    float64
	mmrCandidatePool             int
	hooks                        domain.Hooks
	indexConcurrency             int
	indexBatchRateLimit          int
	indexBatchRateLimitPerWorker bool
	streamBufferSize             int
	reranker                     domain.Reranker
}

// NewPipeline creates a Pipeline with required dependencies.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task arch-quality-pass#T2.1: error return вместо panic для конфигурации (AC-002)
func NewPipeline(store domain.VectorStore, llm domain.LLMProvider, embedder domain.Embedder) (*Pipeline, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}
	if llm == nil {
		return nil, errors.New("nil llm")
	}
	if embedder == nil {
		return nil, errors.New("nil embedder")
	}

	return &Pipeline{
		store:        store,
		llm:          llm,
		embedder:     embedder,
		chunker:      nil,
		systemPrompt: defaultSystemPromptV1,
	}, nil
}

// NewPipelineWithConfig creates a Pipeline with the given configuration.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task arch-quality-pass#T2.1: error return вместо panic для конфигурации (AC-002)
// @sk-task arch-quality-pass#T3.2: принимает PipelineOptions вместо PipelineConfig (AC-004)
func NewPipelineWithConfig(
	store domain.VectorStore,
	llm domain.LLMProvider,
	embedder domain.Embedder,
	cfg PipelineOptions,
) (*Pipeline, error) {
	if cfg.StreamBufferSize < 0 {
		return nil, fmt.Errorf("StreamBufferSize must be >= 0, got %d", cfg.StreamBufferSize)
	}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.SystemPrompt) != "" {
		p.systemPrompt = cfg.SystemPrompt
	}
	p.chunker = cfg.Chunker
	p.maxContextChars = cfg.MaxContextChars
	p.maxContextChunks = cfg.MaxContextChunks
	p.dedupByParentID = cfg.DedupByParentID
	p.mmrEnabled = cfg.MMREnabled
	p.mmrLambda = cfg.MMRLambda
	if p.mmrEnabled && p.mmrLambda == 0 {
		p.mmrLambda = 0.5
	}
	p.mmrCandidatePool = cfg.MMRCandidatePool
	p.hooks = cfg.Hooks
	p.indexConcurrency = cfg.IndexConcurrency
	if p.indexConcurrency <= 0 {
		p.indexConcurrency = 4
	}
	p.indexBatchRateLimit = cfg.IndexBatchRateLimit
	if p.indexBatchRateLimit <= 0 {
		p.indexBatchRateLimit = 10
	}
	p.indexBatchRateLimitPerWorker = cfg.IndexBatchRateLimitPerWorker
	p.streamBufferSize = cfg.StreamBufferSize
	p.reranker = cfg.Reranker
	return p, nil
}

// NewPipelineWithChunker creates a Pipeline with an optional chunker.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task arch-quality-pass#T2.1: error return вместо panic для конфигурации (AC-002)
func NewPipelineWithChunker(
	store domain.VectorStore,
	llm domain.LLMProvider,
	embedder domain.Embedder,
	chunker domain.Chunker,
) (*Pipeline, error) {
	if chunker == nil {
		return nil, errors.New("nil chunker")
	}

	return NewPipelineWithConfig(store, llm, embedder, PipelineOptions{Chunker: chunker})
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T3.1: shared doc-processor между Index и IndexBatch (DEC-004, RQ-004, AC-006)
//
// processDocumentOp — общий путь индексации одного документа: optional chunking
// → embedding → upsert всех чанков. operationName используется в hook-вызовах
// и метриках, чтобы различать вызовы из Index vs IndexBatch.
//
// Заменяет processDocumentForBatch из T1.2: единый helper для обоих entry-points
// (Index и IndexBatch) вместо двух почти-копий.
//
// T3.2: делегирует chunking+embedding в produceChunks, оставляя здесь только
// store.Upsert каждого чанка. Это позволяет updateDocumentAtomic повторно
// использовать chunk+embed (через produceChunks) без двойной логики.
func (p *Pipeline) processDocumentOp(ctx context.Context, operationName string, doc domain.Document) error {
	chunks, err := p.produceChunks(ctx, operationName, doc)
	if err != nil {
		return err
	}
	for _, ch := range chunks {
		if err := p.store.Upsert(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

// @sk-task api-consistency-pass#T3.2: shared chunk+embed helper для atomic update (DEC-005, AC-008)
// @sk-task arch-quality-pass#T1.2: use context from hookStart (AC-001)
// @sk-task arch-quality-pass#T3.1: использует context из hookStart для embed; передаёт его же в hookEnd (AC-001, AC-005)
//
// produceChunks выполняет chunking + embedding + validation для одного документа
// без персистенции. Вызывающий код отвечает за upsert (в store или в tx).
// Используется:
// - processDocumentOp (T3.1) — store.Upsert каждого чанка;
// - updateDocumentAtomicTransactional (T3.2) — tx.Upsert каждого чанка;
// - updateDocumentAtomicBestEffort (T3.2) — косвенно через processDocumentOp.
func (p *Pipeline) produceChunks(ctx context.Context, operationName string, doc domain.Document) ([]domain.Chunk, error) {
	if p.chunker != nil {
		chunkStart := time.Now()
		traceCtx := p.hookStart(ctx, operationName, domain.HookStageChunking)
		chunks, err := p.chunker.Chunk(traceCtx, doc)
		p.hookEnd(traceCtx, operationName, domain.HookStageChunking, chunkStart, err)
		if err != nil {
			return nil, err
		}
		for i := range chunks {
			embedStart := time.Now()
			traceCtx := p.hookStart(ctx, operationName, domain.HookStageEmbed)
			embedding, err := p.embedder.Embed(traceCtx, chunks[i].Content)
			p.hookEnd(traceCtx, operationName, domain.HookStageEmbed, embedStart, err)
			if err != nil {
				return nil, err
			}
			chunks[i].Embedding = embedding
			if err := chunks[i].Validate(); err != nil {
				return nil, err
			}
		}
		return chunks, nil
	}

	embedStart := time.Now()
	traceCtx := p.hookStart(ctx, operationName, domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(traceCtx, doc.Content)
	p.hookEnd(traceCtx, operationName, domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return nil, err
	}

	chunk := domain.Chunk{
		ID:        fmt.Sprintf("%s#%d", doc.ID, 0),
		Content:   doc.Content,
		ParentID:  doc.ID,
		Embedding: embedding,
		Position:  0,
	}
	if err := chunk.Validate(); err != nil {
		return nil, err
	}
	return []domain.Chunk{chunk}, nil
}

// Index индексирует набор документов параллельно с ограничением
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T3.1: параллельная обработка Index через processDocsConcurrently (DEC-004, RQ-004, AC-006)
//
// Index индексирует набор документов параллельно с ограничением
// p.indexConcurrency и rateLimit p.indexBatchRateLimit.
//
// Семантика fail-fast: при первой ошибке обработки cancel-ит in-flight siblings
// и возвращает оригинальную ошибку (не context.Canceled). Документы, не
// прошедшие Validate, также прерывают выполнение и возвращают первую такую
// ошибку.
//
// Параллелизм: реализовано через processDocsConcurrently (T1.2). На каждую
// документную goroutine — отдельный семафор слот и общий rate-limiter.
func (p *Pipeline) Index(ctx context.Context, docs []domain.Document) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, failed, _ := processDocsConcurrently(
		cancelCtx,
		docs,
		p.indexConcurrency,
		p.indexBatchRateLimit,
		p.indexBatchRateLimitPerWorker,
		func(procCtx context.Context, doc domain.Document) error {
			err := p.processDocumentOp(procCtx, "Index", doc)
			if err != nil {
				cancel()
			}
			return err
		},
	)

	if len(failed) > 0 {
		for _, fe := range failed {
			if !errors.Is(fe.Error, context.Canceled) && !errors.Is(fe.Error, context.DeadlineExceeded) {
				return fe.Error
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDocument deletes all chunks belonging to a document by its ID.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) DeleteDocument(ctx context.Context, docID string) error {
	ds, ok := p.store.(domain.DocumentStore)
	if !ok {
		return ErrDeleteNotSupported
	}
	return ds.DeleteByParentID(ctx, docID)
}

// UpdateDocument выполняет атомарное обновление документа через
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T3.2: делегирует в updateDocumentAtomic (DEC-005, RQ-005, AC-008, AC-009)
//
// UpdateDocument выполняет атомарное обновление документа через
// updateDocumentAtomic, который выбирает transactional или best-effort путь
// в зависимости от capability underlying store.
func (p *Pipeline) UpdateDocument(ctx context.Context, doc domain.Document) error {
	return p.updateDocumentAtomic(ctx, doc)
}
