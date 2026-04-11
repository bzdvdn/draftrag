package draftrag

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var _ = PipelineOptions{}
var _ = NewPipelineWithOptions

type topKStore struct {
	gotTopK int
}

func (s *topKStore) Upsert(ctx context.Context, chunk domain.Chunk) error { return nil }
func (s *topKStore) Delete(ctx context.Context, id string) error          { return nil }
func (s *topKStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	s.gotTopK = topK
	return domain.RetrievalResult{}, nil
}

type okEmbedder struct{}

func (okEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{1, 2}, nil
}

type okLLM struct{}

func (okLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "ok", nil
}

func TestPipelineOptions_DefaultTopK_AppliesToQueryAndAnswer(t *testing.T) {
	store := &topKStore{}
	p := NewPipelineWithOptions(store, okLLM{}, okEmbedder{}, PipelineOptions{
		DefaultTopK: 3,
	})

	_, _ = p.Query(context.Background(), "q")
	if store.gotTopK != 3 {
		t.Fatalf("Query: expected topK=3, got %d", store.gotTopK)
	}

	_, _ = p.Answer(context.Background(), "q")
	if store.gotTopK != 3 {
		t.Fatalf("Answer: expected topK=3, got %d", store.gotTopK)
	}
}
