package draftrag

import (
	"context"
	"fmt"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/chunker"
)

// ContextualChunkerOptions задаёт параметры контекстного чанкера.
type ContextualChunkerOptions struct {
	// Base — базовый Chunker, чьи чанки будут обогащены контекстом (обязательно).
	Base Chunker
	// ContextKey — ключ в Document.Metadata, откуда берётся контекст (обязательно).
	ContextKey string
	// Template — шаблон, содержащий {context} и {content} плейсхолдеры (обязательно).
	Template string
}

type contextualChunkerImpl struct {
	inner *chunker.ContextualChunker
}

// @sk-task contextual-chunking#T1.2: NewContextualChunker constructor + validation (RQ-005)
func NewContextualChunker(opts ContextualChunkerOptions) (Chunker, error) {
	if err := validateContextualChunkerOptions(opts); err != nil {
		return nil, err
	}

	inner := chunker.NewContextualChunker(opts.Base, opts.ContextKey, opts.Template)
	return &contextualChunkerImpl{inner: inner}, nil
}

func (w *contextualChunkerImpl) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return w.inner.Chunk(ctx, doc)
}

func validateContextualChunkerOptions(opts ContextualChunkerOptions) error {
	if opts.Base == nil {
		return fmt.Errorf("%w: Base is required for contextual chunker", ErrInvalidChunkerConfig)
	}
	if opts.ContextKey == "" {
		return fmt.Errorf("%w: ContextKey is required", ErrInvalidChunkerConfig)
	}
	if opts.Template == "" {
		return fmt.Errorf("%w: Template is required", ErrInvalidChunkerConfig)
	}
	if !strings.Contains(opts.Template, "{content}") {
		return fmt.Errorf("%w: Template must contain {content} placeholder", ErrInvalidChunkerConfig)
	}

	return nil
}
