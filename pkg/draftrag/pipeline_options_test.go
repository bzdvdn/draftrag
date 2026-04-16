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

func (s *topKStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (s *topKStore) Delete(_ context.Context, _ string) error       { return nil }
func (s *topKStore) Search(_ context.Context, _ []float64, topK int) (domain.RetrievalResult, error) {
	s.gotTopK = topK
	return domain.RetrievalResult{}, nil
}

type okEmbedder struct{}

func (okEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{1, 2}, nil
}

type okLLM struct{}

func (okLLM) Generate(_ context.Context, _, _ string) (string, error) {
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
