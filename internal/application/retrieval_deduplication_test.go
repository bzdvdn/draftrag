package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type fixedSearchStoreDedup struct {
	result domain.RetrievalResult
}

func (fixedSearchStoreDedup) Upsert(ctx context.Context, chunk domain.Chunk) error { return nil }
func (fixedSearchStoreDedup) Delete(ctx context.Context, id string) error          { return nil }
func (s fixedSearchStoreDedup) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	return s.result, nil
}

type fixedEmbedderDedup struct{}

func (fixedEmbedderDedup) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{1}, nil
}

type panicLLMDedup struct{}

func (panicLLMDedup) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	panic("should not be called")
}

func TestPipeline_Query_DedupByParentID_Enabled(t *testing.T) {
	input := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{ParentID: "A", Content: "A1"}, Score: 0.8},
			{Chunk: domain.Chunk{ParentID: "B", Content: "B1"}, Score: 0.7},
			{Chunk: domain.Chunk{ParentID: "A", Content: "A2"}, Score: 0.9},
		},
		TotalFound: 3,
	}

	p := NewPipelineWithConfig(
		fixedSearchStoreDedup{result: input},
		panicLLMDedup{},
		fixedEmbedderDedup{},
		PipelineConfig{DedupByParentID: true},
	)

	got, err := p.Query(context.Background(), "Q", 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(got.Chunks) != 2 {
		t.Fatalf("expected 2 chunks after dedup, got %#v", got.Chunks)
	}
	if got.Chunks[0].Chunk.ParentID != "A" || got.Chunks[0].Chunk.Content != "A2" {
		t.Fatalf("expected best chunk for A first, got %#v", got.Chunks[0])
	}
	if got.Chunks[1].Chunk.ParentID != "B" || got.Chunks[1].Chunk.Content != "B1" {
		t.Fatalf("expected chunk for B second, got %#v", got.Chunks[1])
	}
	if got.QueryText != "Q" {
		t.Fatalf("expected QueryText=Q, got %q", got.QueryText)
	}
	if got.TotalFound != 3 {
		t.Fatalf("expected TotalFound unchanged, got %d", got.TotalFound)
	}
}

func TestPipeline_Query_DedupByParentID_Disabled_NoChanges(t *testing.T) {
	input := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{ParentID: "A", Content: "A1"}, Score: 0.8},
			{Chunk: domain.Chunk{ParentID: "B", Content: "B1"}, Score: 0.7},
			{Chunk: domain.Chunk{ParentID: "A", Content: "A2"}, Score: 0.9},
		},
		TotalFound: 3,
	}

	p := NewPipelineWithConfig(
		fixedSearchStoreDedup{result: input},
		panicLLMDedup{},
		fixedEmbedderDedup{},
		PipelineConfig{DedupByParentID: false},
	)

	got, err := p.Query(context.Background(), "Q", 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(got.Chunks) != len(input.Chunks) {
		t.Fatalf("expected chunks unchanged, got %#v", got.Chunks)
	}
	for i := range input.Chunks {
		if got.Chunks[i].Chunk.ParentID != input.Chunks[i].Chunk.ParentID ||
			got.Chunks[i].Chunk.Content != input.Chunks[i].Chunk.Content ||
			got.Chunks[i].Score != input.Chunks[i].Score {
			t.Fatalf("expected chunk %d unchanged, got=%#v want=%#v", i, got.Chunks[i], input.Chunks[i])
		}
	}
}
