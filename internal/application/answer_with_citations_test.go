package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type fixedSearchStore2 struct {
	result domain.RetrievalResult
}

func (fixedSearchStore2) Upsert(ctx context.Context, chunk domain.Chunk) error { return nil }
func (fixedSearchStore2) Delete(ctx context.Context, id string) error          { return nil }
func (s fixedSearchStore2) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	return s.result, nil
}

type fixedEmbedder2 struct{}

func (fixedEmbedder2) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{1}, nil
}

type okLLM2 struct{}

func (okLLM2) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "ok", nil
}

type errLLM struct {
	err error
}

func (l errLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "", l.err
}

func TestPipeline_AnswerWithCitations_ReturnsAnswerAndRetrieval(t *testing.T) {
	expected := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A"}, Score: 0.9},
		},
		TotalFound: 1,
	}

	p := NewPipelineWithConfig(
		fixedSearchStore2{result: expected},
		okLLM2{},
		fixedEmbedder2{},
		PipelineConfig{},
	)

	answer, gotRetrieval, err := p.AnswerWithCitations(context.Background(), "Q", 3)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if answer != "ok" {
		t.Fatalf("expected answer %q, got %q", "ok", answer)
	}
	if gotRetrieval.TotalFound != 1 || len(gotRetrieval.Chunks) != 1 {
		t.Fatalf("unexpected retrieval: %#v", gotRetrieval)
	}
	if gotRetrieval.Chunks[0].Chunk.Content != "A" {
		t.Fatalf("unexpected chunk content: %#v", gotRetrieval)
	}
	if gotRetrieval.QueryText != "Q" {
		t.Fatalf("expected QueryText=%q, got %q", "Q", gotRetrieval.QueryText)
	}
}

func TestPipeline_AnswerWithCitations_PartialResultOnGenerateError(t *testing.T) {
	expected := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A"}, Score: 0.9},
		},
		TotalFound: 1,
	}
	genErr := errors.New("generate failed")

	p := NewPipelineWithConfig(
		fixedSearchStore2{result: expected},
		errLLM{err: genErr},
		fixedEmbedder2{},
		PipelineConfig{},
	)

	answer, gotRetrieval, err := p.AnswerWithCitations(context.Background(), "Q", 3)
	if !errors.Is(err, genErr) {
		t.Fatalf("expected genErr, got %v", err)
	}
	if answer != "" {
		t.Fatalf("expected empty answer on generate error, got %q", answer)
	}
	if len(gotRetrieval.Chunks) != 1 || gotRetrieval.QueryText != "Q" {
		t.Fatalf("expected partial retrieval result, got %#v", gotRetrieval)
	}
}
