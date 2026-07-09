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

func (panicStoreInline) Health(_ context.Context) error { return nil }
func (panicStoreInline) Upsert(_ context.Context, _ domain.Chunk) error {
	panic("should not be called")
}
func (panicStoreInline) Delete(_ context.Context, _ string) error { panic("should not be called") }
func (panicStoreInline) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	panic("should not be called")
}

type panicEmbedderInline struct{}

func (panicEmbedderInline) Health(_ context.Context) error { return nil }
func (panicEmbedderInline) Embed(_ context.Context, _ string) ([]float64, error) {
	panic("should not be called")
}

type panicLLMInline struct{}

func (panicLLMInline) Health(_ context.Context) error { return nil }
func (panicLLMInline) Generate(_ context.Context, _, _ string) (string, error) {
	panic("should not be called")
}

func TestPipeline_InlineCite_Validation(t *testing.T) {
	p, err := NewPipeline(panicStoreInline{}, panicLLMInline{}, panicEmbedderInline{})
	if err != nil {
		t.Fatal(err)
	}

	_, _, _, err = p.Search("   ").TopK(5).InlineCite(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}

	_, _, _, err = p.Search("q").TopK(0).InlineCite(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}
