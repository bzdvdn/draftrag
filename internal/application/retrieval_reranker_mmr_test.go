package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type mmrStore struct {
	candidates []domain.RetrievedChunk
	lastTopK   int
}

func (s *mmrStore) Upsert(ctx context.Context, chunk domain.Chunk) error { return nil }
func (s *mmrStore) Delete(ctx context.Context, id string) error          { return nil }

func (s *mmrStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	s.lastTopK = topK
	out := s.candidates
	if topK < len(out) {
		out = out[:topK]
	}
	return domain.RetrievalResult{Chunks: out, TotalFound: len(s.candidates)}, nil
}

type fixedEmbedderMMR struct {
	v []float64
}

func (e fixedEmbedderMMR) Embed(ctx context.Context, text string) ([]float64, error) { return e.v, nil }

type okLLM struct{}

func (okLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "ok", nil
}

func TestPipeline_MMR_ChangesSelectionWhenEnabled(t *testing.T) {
	store := &mmrStore{
		candidates: []domain.RetrievedChunk{
			// Кластер A (очень похожи друг на друга), высокие score.
			{Chunk: domain.Chunk{Content: "A1", ParentID: "A1", Embedding: []float64{1, 0}}, Score: 0.95},
			{Chunk: domain.Chunk{Content: "A2", ParentID: "A2", Embedding: []float64{0.99, 0.01}}, Score: 0.94},
			// Кластер B, более низкие score.
			{Chunk: domain.Chunk{Content: "B1", ParentID: "B1", Embedding: []float64{0, 1}}, Score: 0.60},
			{Chunk: domain.Chunk{Content: "B2", ParentID: "B2", Embedding: []float64{0.01, 0.99}}, Score: 0.59},
		},
	}

	p := NewPipelineWithConfig(
		store,
		okLLM{},
		fixedEmbedderMMR{v: []float64{1, 0}},
		PipelineConfig{
			MMREnabled:       true,
			MMRLambda:        0.5,
			MMRCandidatePool: 4,
		},
	)

	_, retrieval, err := p.AnswerWithCitations(context.Background(), "Q", 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if store.lastTopK != 4 {
		t.Fatalf("expected candidate pool topK=4, got %d", store.lastTopK)
	}
	if len(retrieval.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %#v", retrieval.Chunks)
	}

	got0 := retrieval.Chunks[0].Chunk.Content
	got1 := retrieval.Chunks[1].Chunk.Content
	if got0 == "A1" && got1 == "A2" {
		t.Fatalf("expected diversified selection (not only cluster A), got %q and %q", got0, got1)
	}
	// Должен быть выбран хотя бы один из кластера B.
	if got0 != "B1" && got0 != "B2" && got1 != "B1" && got1 != "B2" {
		t.Fatalf("expected at least one B* chunk selected, got %q and %q", got0, got1)
	}
}

func TestPipeline_MMR_DisabledDoesNotChangeBehavior(t *testing.T) {
	store := &mmrStore{
		candidates: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A1", ParentID: "A1", Embedding: []float64{1, 0}}, Score: 0.95},
			{Chunk: domain.Chunk{Content: "A2", ParentID: "A2", Embedding: []float64{0.99, 0.01}}, Score: 0.94},
			{Chunk: domain.Chunk{Content: "B1", ParentID: "B1", Embedding: []float64{0, 1}}, Score: 0.60},
		},
	}

	p := NewPipelineWithConfig(
		store,
		okLLM{},
		fixedEmbedderMMR{v: []float64{1, 0}},
		PipelineConfig{
			MMREnabled:       false,
			MMRLambda:        0.5,
			MMRCandidatePool: 10,
		},
	)

	_, retrieval, err := p.AnswerWithCitations(context.Background(), "Q", 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if store.lastTopK != 2 {
		t.Fatalf("expected topK=2 when MMR disabled, got %d", store.lastTopK)
	}
	if len(retrieval.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %#v", retrieval.Chunks)
	}
	if retrieval.Chunks[0].Chunk.Content != "A1" || retrieval.Chunks[1].Chunk.Content != "A2" {
		t.Fatalf("expected baseline top-2 by score (A1,A2), got %q,%q", retrieval.Chunks[0].Chunk.Content, retrieval.Chunks[1].Chunk.Content)
	}
}

func TestPipeline_MMR_EnabledRequiresEmbeddings(t *testing.T) {
	store := &mmrStore{
		candidates: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A1", ParentID: "A1", Embedding: []float64{1, 0}}, Score: 0.95},
			{Chunk: domain.Chunk{Content: "A2", ParentID: "A2", Embedding: nil}, Score: 0.94},
		},
	}

	p := NewPipelineWithConfig(
		store,
		okLLM{},
		fixedEmbedderMMR{v: []float64{1, 0}},
		PipelineConfig{
			MMREnabled:       true,
			MMRLambda:        0.5,
			MMRCandidatePool: 2,
		},
	)

	_, _, err := p.AnswerWithCitations(context.Background(), "Q", 2)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, errMMREmbeddingMissing) {
		t.Fatalf("expected errMMREmbeddingMissing, got %v", err)
	}
}
