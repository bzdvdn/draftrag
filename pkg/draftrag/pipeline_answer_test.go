package draftrag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Компиляционные проверки наличия методов.
var (
	_ = (*Pipeline).Answer
	_ = (*Pipeline).Search
)

type panicStore struct{}

func (panicStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	panic("should not be called")
}
func (panicStore) Delete(ctx context.Context, id string) error { panic("should not be called") }
func (panicStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	panic("should not be called")
}

type panicEmbedder struct{}

func (panicEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	panic("should not be called")
}

type panicLLM struct{}

func (panicLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	panic("should not be called")
}

func TestPipeline_Answer_Validation(t *testing.T) {
	p := NewPipeline(panicStore{}, panicLLM{}, panicEmbedder{})

	_, err := p.Search("   ").TopK(5).Answer(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}

	_, err = p.Search("q").TopK(0).Answer(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

func TestPipeline_Answer_ContextCancelFast(t *testing.T) {
	p := NewPipeline(panicStore{}, panicLLM{}, panicEmbedder{})

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	_, err := p.Search("q").TopK(5).Answer(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}
