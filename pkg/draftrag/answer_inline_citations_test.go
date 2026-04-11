package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Компиляционные проверки наличия методов.
var (
	_ = (*SearchBuilder).InlineCite
)

type panicStoreInline struct{}

func (panicStoreInline) Upsert(ctx context.Context, chunk domain.Chunk) error {
	panic("should not be called")
}
func (panicStoreInline) Delete(ctx context.Context, id string) error { panic("should not be called") }
func (panicStoreInline) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	panic("should not be called")
}

type panicEmbedderInline struct{}

func (panicEmbedderInline) Embed(ctx context.Context, text string) ([]float64, error) {
	panic("should not be called")
}

type panicLLMInline struct{}

func (panicLLMInline) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	panic("should not be called")
}

func TestPipeline_InlineCite_Validation(t *testing.T) {
	p := NewPipeline(panicStoreInline{}, panicLLMInline{}, panicEmbedderInline{})

	_, _, _, err := p.Search("   ").TopK(5).InlineCite(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}

	_, _, _, err = p.Search("q").TopK(0).InlineCite(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}
