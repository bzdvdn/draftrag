package chunker

import (
	"context"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task contextual-chunking#T1.1: ContextualChunker decorator (AC-001–AC-006)
type ContextualChunker struct {
	base       domain.Chunker
	contextKey string
	template   string
}

func NewContextualChunker(base domain.Chunker, contextKey, template string) *ContextualChunker {
	return &ContextualChunker{
		base:       base,
		contextKey: contextKey,
		template:   template,
	}
}

func (c *ContextualChunker) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	chunks, err := c.base.Chunk(ctx, doc)
	if err != nil {
		return nil, err
	}

	contextStr := doc.Metadata[c.contextKey]
	if contextStr == "" {
		return chunks, nil
	}

	result := make([]domain.Chunk, len(chunks))
	for i, ch := range chunks {
		content := strings.ReplaceAll(c.template, "{context}", contextStr)
		content = strings.ReplaceAll(content, "{content}", ch.Content)
		result[i] = ch
		result[i].Content = content
	}

	return result, nil
}
