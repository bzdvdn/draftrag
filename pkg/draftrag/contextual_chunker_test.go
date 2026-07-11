package draftrag

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type stubBaseChunker struct{}

func (stubBaseChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return []domain.Chunk{
		{ID: doc.ID + ":0", Content: doc.Content, ParentID: doc.ID, Position: 0},
	}, nil
}

// @sk-test contextual-chunking#T2.2: TestNewContextualChunker_InvalidConfig (RQ-005)
func TestNewContextualChunker_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		opts ContextualChunkerOptions
	}{
		{"nil base", ContextualChunkerOptions{Base: nil, ContextKey: "title", Template: "[CONTEXT] {context}\n{content}"}},
		{"empty context key", ContextualChunkerOptions{Base: stubBaseChunker{}, ContextKey: "", Template: "[CONTEXT] {context}\n{content}"}},
		{"empty template", ContextualChunkerOptions{Base: stubBaseChunker{}, ContextKey: "title", Template: ""}},
		{"missing content placeholder", ContextualChunkerOptions{Base: stubBaseChunker{}, ContextKey: "title", Template: "{context} only"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewContextualChunker(tc.opts)
			if !errors.Is(err, ErrInvalidChunkerConfig) {
				t.Fatalf("expected ErrInvalidChunkerConfig, got %v", err)
			}
		})
	}
}

// @sk-test contextual-chunking#T2.2: TestNewContextualChunker_ValidConfig (RQ-005)
func TestNewContextualChunker_ValidConfig(t *testing.T) {
	ch, err := NewContextualChunker(ContextualChunkerOptions{
		Base:       stubBaseChunker{},
		ContextKey: "title",
		Template:   "[CONTEXT] {context}\n{content}",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil chunker")
	}

	doc := domain.Document{
		ID:       "doc-1",
		Content:  "test content",
		Metadata: map[string]string{"title": "Test Doc"},
	}
	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !strings.HasPrefix(chunks[0].Content, "[CONTEXT] Test Doc\n") {
		t.Fatalf("expected context prefix, got %q", chunks[0].Content)
	}
}

type ctxAwareEmbedder struct {
	keyword string
	match   []float64
	other   []float64
}

func (e *ctxAwareEmbedder) Health(_ context.Context) error { return nil }

func (e *ctxAwareEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	if strings.Contains(text, e.keyword) {
		return e.match, nil
	}
	return e.other, nil
}

type noopLLM struct{}

func (noopLLM) Health(_ context.Context) error { return nil }

func (noopLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "mock response", nil
}

// @sk-test contextual-chunking#T3.1: TestContextualChunker_SearchByContextWord (AC-005)
func TestContextualChunker_SearchByContextWord(t *testing.T) {
	ctx := context.Background()

	basicCh := NewBasicChunker(BasicChunkerOptions{ChunkSize: 1000, Overlap: 0, MaxChunks: 0})

	contextualCh, err := NewContextualChunker(ContextualChunkerOptions{
		Base:       basicCh,
		ContextKey: "title",
		Template:   "[CONTEXT] {context}\n{content}",
	})
	if err != nil {
		t.Fatalf("contextual chunker: %v", err)
	}

	store := vectorstore.NewInMemoryStore()
	embedder := &ctxAwareEmbedder{
		keyword: "Research",
		match:   []float64{1, 0},
		other:   []float64{0, 1},
	}

	p, err := NewPipelineWithChunker(store, noopLLM{}, embedder, contextualCh)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	docWithContext := domain.Document{
		ID:       "doc-1",
		Content:  "pure financial data without context keyword",
		Metadata: map[string]string{"title": "Research Paper"},
	}
	if err := p.Index(ctx, []domain.Document{docWithContext}); err != nil {
		t.Fatalf("index: %v", err)
	}

	result, err := p.Retrieve(ctx, "Research", 5)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if result.TotalFound == 0 || len(result.Chunks) == 0 {
		t.Fatalf("expected at least 1 result, got total=%d len=%d", result.TotalFound, len(result.Chunks))
	}

	found := false
	for _, ch := range result.Chunks {
		if strings.HasPrefix(ch.Chunk.Content, "[CONTEXT] Research Paper\n") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected chunk with context prefix [CONTEXT] Research Paper\\n, none found")
	}
}
