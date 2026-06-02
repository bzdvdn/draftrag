package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var (
	ErrFiltersNotSupported = errors.New("vector store does not support filters")

	ErrStreamingNotSupported = errors.New("LLM provider does not support streaming")

	ErrDeleteNotSupported = errors.New("vector store does not support DeleteByParentID")
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
type PipelineConfig struct {
	SystemPrompt        string
	Chunker             domain.Chunker
	MaxContextChars     int
	MaxContextChunks    int
	DedupByParentID     bool
	MMREnabled          bool
	MMRLambda           float64
	MMRCandidatePool    int
	Hooks               domain.Hooks
	IndexConcurrency    int
	IndexBatchRateLimit int
	Reranker            domain.Reranker
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
type Pipeline struct {
	store               domain.VectorStore
	llm                 domain.LLMProvider
	embedder            domain.Embedder
	chunker             domain.Chunker
	systemPrompt        string
	maxContextChars     int
	maxContextChunks    int
	dedupByParentID     bool
	mmrEnabled          bool
	mmrLambda           float64
	mmrCandidatePool    int
	hooks               domain.Hooks
	indexConcurrency    int
	indexBatchRateLimit int
	reranker            domain.Reranker
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func NewPipeline(store domain.VectorStore, llm domain.LLMProvider, embedder domain.Embedder) *Pipeline {
	if store == nil {
		panic("nil store")
	}
	if llm == nil {
		panic("nil llm")
	}
	if embedder == nil {
		panic("nil embedder")
	}

	return &Pipeline{
		store:        store,
		llm:          llm,
		embedder:     embedder,
		chunker:      nil,
		systemPrompt: defaultSystemPromptV1,
	}
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func NewPipelineWithConfig(
	store domain.VectorStore,
	llm domain.LLMProvider,
	embedder domain.Embedder,
	cfg PipelineConfig,
) *Pipeline {
	p := NewPipeline(store, llm, embedder)
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
	p.reranker = cfg.Reranker
	return p
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func NewPipelineWithChunker(
	store domain.VectorStore,
	llm domain.LLMProvider,
	embedder domain.Embedder,
	chunker domain.Chunker,
) *Pipeline {
	if chunker == nil {
		panic("nil chunker")
	}

	return NewPipelineWithConfig(store, llm, embedder, PipelineConfig{Chunker: chunker})
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) indexChunks(ctx context.Context, operation string, chunks []domain.Chunk) error {
	for _, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return err
		}

		embedStart := time.Now()
		p.hookStart(ctx, operation, domain.HookStageEmbed)
		embedding, err := p.embedder.Embed(ctx, chunk.Content)
		p.hookEnd(ctx, operation, domain.HookStageEmbed, embedStart, err)
		if err != nil {
			return err
		}
		chunk.Embedding = embedding

		if err := chunk.Validate(); err != nil {
			return err
		}
		if err := p.store.Upsert(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) Index(ctx context.Context, docs []domain.Document) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, doc := range docs {
		if err := doc.Validate(); err != nil {
			return err
		}

		if p.chunker != nil {
			chunkStart := time.Now()
			p.hookStart(ctx, "Index", domain.HookStageChunking)
			chunks, err := p.chunker.Chunk(ctx, doc)
			p.hookEnd(ctx, "Index", domain.HookStageChunking, chunkStart, err)
			if err != nil {
				return err
			}
			if err := p.indexChunks(ctx, "Index", chunks); err != nil {
				return err
			}
			continue
		}

		embedStart := time.Now()
		p.hookStart(ctx, "Index", domain.HookStageEmbed)
		embedding, err := p.embedder.Embed(ctx, doc.Content)
		p.hookEnd(ctx, "Index", domain.HookStageEmbed, embedStart, err)
		if err != nil {
			return err
		}

		chunk := domain.Chunk{
			ID:        fmt.Sprintf("%s#%d", doc.ID, 0),
			Content:   doc.Content,
			ParentID:  doc.ID,
			Embedding: embedding,
			Position:  0,
		}
		if err := chunk.Validate(); err != nil {
			return err
		}

		if err := p.store.Upsert(ctx, chunk); err != nil {
			return err
		}
	}

	return nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) DeleteDocument(ctx context.Context, docID string) error {
	ds, ok := p.store.(domain.DocumentStore)
	if !ok {
		return ErrDeleteNotSupported
	}
	return ds.DeleteByParentID(ctx, docID)
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) UpdateDocument(ctx context.Context, doc domain.Document) error {
	if err := p.DeleteDocument(ctx, doc.ID); err != nil {
		return err
	}
	return p.Index(ctx, []domain.Document{doc})
}
