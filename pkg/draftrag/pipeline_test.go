package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type testEmbedder struct{}

func (testEmbedder) Embed(ctx context.Context, _ string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	return []float64{1, 0}, nil
}

type testLLM struct{}

func (testLLM) Generate(ctx context.Context, _, _ string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	return "ok", nil
}

func TestPipeline_ValidationErrors(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	ctx := context.Background()

	if err := p.Index(ctx, []domain.Document{{ID: "doc-1", Content: ""}}); !errors.Is(err, ErrEmptyDocument) {
		t.Fatalf("expected ErrEmptyDocument, got %v", err)
	}

	if _, err := p.Query(ctx, ""); !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}

	if _, err := p.Search("cat").TopK(0).Retrieve(ctx); !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}
