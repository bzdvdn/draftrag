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

	// @sk-task sub-query-decomposition#T3.3: sentinel for nil decomposer guard (AC-005, AC-006)
	ErrSubDecomposeNotSupported = errors.New("sub-query decomposition not supported: no QueryDecomposer configured")
)

// PipelineOptions configures a Pipeline behaviour.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task arch-quality-pass#T3.2: единый struct конфигурации (AC-004)
// @sk-task pii-guardrails#T2.1: PipelineOptions.PIIDetector (RQ-001, RQ-002)
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
	PIIDetector                  domain.PIIDetector
	ParentContextEnabled         *bool
	Middleware                   []domain.Middleware
}

// Pipeline is the core RAG pipeline coordinating store, LLM, and embedder.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task pii-guardrails#T2.1: Pipeline.PIIDetector (RQ-001, RQ-002)
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
	piidetector                  domain.PIIDetector
	parentContextEnabled         bool
	middleware                   []domain.Middleware
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
		store:                store,
		llm:                  llm,
		embedder:             embedder,
		chunker:              nil,
		systemPrompt:         defaultSystemPromptV1,
		parentContextEnabled: true,
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
	p.piidetector = cfg.PIIDetector
	p.middleware = cfg.Middleware
	p.parentContextEnabled = true
	if cfg.ParentContextEnabled != nil {
		p.parentContextEnabled = *cfg.ParentContextEnabled
	}
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

// @sk-task hierarchical-indices#T3.1: parent save in processDocumentOp (AC-001, RQ-001, DEC-003)
//
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
//
// T3.1: после upsert чанков сохраняет parent-документ в ParentDocumentStore,
// если parentContextEnabled=true и store поддерживает capability.
// Parent embedding вычисляется из doc.Content (DEC-003); при отсутствии chunker'а
// переиспользует embedding единственного чанка во избежание дублирования вызова.
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

	if p.parentContextEnabled {
		if ps, ok := p.store.(domain.ParentDocumentStore); ok {
			parentEmbedding := p.parentEmbeddingOrEmbed(ctx, operationName, doc)
			if parentEmbedding == nil && len(chunks) > 0 {
				parentEmbedding = chunks[0].Embedding
			}
			if parentEmbedding != nil {
				if err := ps.UpsertParent(ctx, doc, parentEmbedding); err != nil {
					return err
				}
			}
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
		var chunks []domain.Chunk
		_, err := p.execWithStageMiddleware(ctx, domain.HookStageChunking, operationName, domain.StageData{Document: doc}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
			chunkStart := time.Now()
			traceCtx := p.hookStart(ctx, operationName, domain.HookStageChunking)
			var chunkErr error
			chunks, chunkErr = p.chunker.Chunk(traceCtx, d.Document)
			p.hookEnd(traceCtx, operationName, domain.HookStageChunking, chunkStart, chunkErr)
			if chunkErr != nil {
				return d, chunkErr
			}
			for i := range chunks {
				embedData, eErr := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, operationName, domain.StageData{Query: chunks[i].Content}, func(ctx context.Context, ed domain.StageData) (domain.StageData, error) {
					embedStart := time.Now()
					traceCtx := p.hookStart(ctx, operationName, domain.HookStageEmbed)
					embedding, embedErr := p.embedder.Embed(traceCtx, ed.Query)
					p.hookEnd(traceCtx, operationName, domain.HookStageEmbed, embedStart, embedErr)
					if embedErr != nil {
						return ed, embedErr
					}
					ed.Embedding = embedding
					return ed, nil
				})
				if eErr != nil {
					return d, eErr
				}
				chunks[i].Embedding = embedData.Embedding
				if err := chunks[i].Validate(); err != nil {
					return d, err
				}
			}
			return d, nil
		})
		if err != nil {
			return nil, err
		}
		return chunks, nil
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, operationName, domain.StageData{Query: doc.Content}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		traceCtx := p.hookStart(ctx, operationName, domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(traceCtx, d.Query)
		p.hookEnd(traceCtx, operationName, domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
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

// @sk-task hierarchical-indices#T3.1: parentEmbeddingOrEmbed helper (AC-001, DEC-003)
//
// parentEmbeddingOrEmbed возвращает embedding для parent-документа.
// При отсутствии chunker'а возвращает nil — вызывающий код должен использовать
// embedding единственного чанка. При наличии chunker'а вызывает embedder
// для полного текста документа.
func (p *Pipeline) parentEmbeddingOrEmbed(ctx context.Context, operationName string, doc domain.Document) []float64 {
	if p.chunker == nil {
		return nil
	}
	embedStart := time.Now()
	traceCtx := p.hookStart(ctx, operationName, domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(traceCtx, doc.Content)
	p.hookEnd(traceCtx, operationName, domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return nil
	}
	return embedding
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

// @sk-task hierarchical-indices#T3.2: maybeAttachParentContent helper (AC-002, AC-003, AC-004, DM-001)
//
// maybeAttachParentContent загружает parent-документы для каждого уникального
// ParentID среди найденных чанков и заполняет RetrievedChunk.ParentContent.
//
// Graceful degradation:
// - store не реализует ParentDocumentStore → return (AC-003)
// - parentContextEnabled=false → return (AC-004)
// - GetParentDocument вернул nil → ParentContent остаётся пустой строкой
//
// Group by parentID: для N чанков с одним parentID выполняется ровно один
// GetParentDocument, чтобы избежать N+1 round-trip на remote store.
func (p *Pipeline) maybeAttachParentContent(ctx context.Context, result domain.RetrievalResult) domain.RetrievalResult {
	if !p.parentContextEnabled {
		return result
	}
	ps, ok := p.store.(domain.ParentDocumentStore)
	if !ok {
		return result
	}

	parentCache := make(map[string]string, len(result.Chunks))
	for i, rc := range result.Chunks {
		parentID := rc.Chunk.ParentID
		if parentID == "" {
			continue
		}
		content, cached := parentCache[parentID]
		if !cached {
			parentDoc, err := ps.GetParentDocument(ctx, parentID)
			if err != nil || parentDoc == nil {
				parentCache[parentID] = ""
				continue
			}
			content = parentDoc.Content
			parentCache[parentID] = content
		}
		result.Chunks[i].ParentContent = content
	}
	return result
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
