package draftrag

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/chunker"
)

// SemanticChunkerOptions задаёт параметры семантического чанкера.
type SemanticChunkerOptions struct {
	// Embedder — интерфейс для вычисления эмбеддингов (обязательно).
	Embedder Embedder
	// SimilarityThreshold — порог косинусного сходства [0.0–1.0].
	// При сходстве ниже порога начинается новый чанк.
	SimilarityThreshold float64
	// MinChunkSize — минимальный размер чанка в символах (>= 0).
	// 0 означает без минимума.
	MinChunkSize int
	// MaxChunkSize — максимальный размер чанка в символах (> MinChunkSize или 0).
	// 0 означает без максимума.
	MaxChunkSize int
}

type semanticChunkerImpl struct {
	inner *chunker.SemanticChunker
}

// NewSemanticChunker создаёт SemanticChunker.
// @sk-task chunker-semantic#T2.2: public constructor + validation (AC-007)
func NewSemanticChunker(opts SemanticChunkerOptions) (Chunker, error) {
	if err := validateSemanticChunkerOptions(opts); err != nil {
		return nil, err
	}

	inner := chunker.NewSemanticChunker(chunker.SemanticChunkerOptions{
		Embedder:            opts.Embedder,
		SimilarityThreshold: opts.SimilarityThreshold,
		MinChunkSize:        opts.MinChunkSize,
		MaxChunkSize:        opts.MaxChunkSize,
	})

	return &semanticChunkerImpl{inner: inner}, nil
}

func (w *semanticChunkerImpl) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return w.inner.Chunk(ctx, doc)
}

func validateSemanticChunkerOptions(opts SemanticChunkerOptions) error {
	if opts.Embedder == nil {
		return fmt.Errorf("%w: Embedder is required for semantic chunker", ErrInvalidChunkerConfig)
	}
	if opts.SimilarityThreshold < 0 || opts.SimilarityThreshold > 1.0 {
		return fmt.Errorf("%w: SimilarityThreshold must be in [0.0, 1.0], got %f", ErrInvalidChunkerConfig, opts.SimilarityThreshold)
	}
	if opts.MinChunkSize < 0 {
		return fmt.Errorf("%w: MinChunkSize must be >= 0", ErrInvalidChunkerConfig)
	}
	if opts.MaxChunkSize < 0 {
		return fmt.Errorf("%w: MaxChunkSize must be >= 0", ErrInvalidChunkerConfig)
	}
	if opts.MaxChunkSize > 0 && opts.MinChunkSize > opts.MaxChunkSize {
		return fmt.Errorf("%w: MinChunkSize (%d) must be <= MaxChunkSize (%d)", ErrInvalidChunkerConfig, opts.MinChunkSize, opts.MaxChunkSize)
	}

	return nil
}
