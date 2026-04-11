package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Компиляционные проверки наличия методов.
var (
	_ = (*SearchBuilder).Cite
)

type panicStore2 struct{}

func (panicStore2) Upsert(ctx context.Context, chunk domain.Chunk) error {
	panic("should not be called")
}
func (panicStore2) Delete(ctx context.Context, id string) error { panic("should not be called") }
func (panicStore2) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	panic("should not be called")
}

type panicEmbedder2 struct{}

func (panicEmbedder2) Embed(ctx context.Context, text string) ([]float64, error) {
	panic("should not be called")
}

type panicLLM2 struct{}

func (panicLLM2) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	panic("should not be called")
}

func TestPipeline_Cite_Validation(t *testing.T) {
	p := NewPipeline(panicStore2{}, panicLLM2{}, panicEmbedder2{})

	_, _, err := p.Search("   ").TopK(5).Cite(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}

	_, _, err = p.Search("q").TopK(0).Cite(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}
