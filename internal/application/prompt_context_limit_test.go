package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type fixedSearchStore struct {
	result domain.RetrievalResult
}

func (fixedSearchStore) Upsert(ctx context.Context, chunk domain.Chunk) error { return nil }
func (fixedSearchStore) Delete(ctx context.Context, id string) error          { return nil }
func (s fixedSearchStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	return s.result, nil
}

type fixedEmbedder struct{}

func (fixedEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{1, 2, 3}, nil
}

type captureUserMessageLLM struct {
	userMessage string
}

func (l *captureUserMessageLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	l.userMessage = userMessage
	return "ok", nil
}

func TestPromptContextLimit_MaxContextChunks(t *testing.T) {
	llm := &captureUserMessageLLM{}
	store := fixedSearchStore{
		result: domain.RetrievalResult{
			Chunks: []domain.RetrievedChunk{
				{Chunk: domain.Chunk{Content: "AAA"}},
				{Chunk: domain.Chunk{Content: "BBB"}},
				{Chunk: domain.Chunk{Content: "CCC"}},
			},
		},
	}

	p := NewPipelineWithConfig(store, llm, fixedEmbedder{}, PipelineConfig{MaxContextChunks: 1})
	_, err := p.Answer(context.Background(), "Q", 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := "Контекст:\nAAA\n\nВопрос:\nQ"
	if llm.userMessage != want {
		t.Fatalf("unexpected user message:\nwant=%q\ngot=%q", want, llm.userMessage)
	}
}

func TestPromptContextLimit_MaxContextChars(t *testing.T) {
	llm := &captureUserMessageLLM{}
	store := fixedSearchStore{
		result: domain.RetrievalResult{
			Chunks: []domain.RetrievedChunk{
				{Chunk: domain.Chunk{Content: "AAA"}},
				{Chunk: domain.Chunk{Content: "BBB"}},
				{Chunk: domain.Chunk{Content: "CCC"}},
			},
		},
	}

	p := NewPipelineWithConfig(store, llm, fixedEmbedder{}, PipelineConfig{MaxContextChars: 2})
	_, err := p.Answer(context.Background(), "Q", 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := "Контекст:\nAA\n\nВопрос:\nQ"
	if llm.userMessage != want {
		t.Fatalf("unexpected user message:\nwant=%q\ngot=%q", want, llm.userMessage)
	}
}

func TestPromptContextLimit_BothLimits(t *testing.T) {
	llm := &captureUserMessageLLM{}
	store := fixedSearchStore{
		result: domain.RetrievalResult{
			Chunks: []domain.RetrievedChunk{
				{Chunk: domain.Chunk{Content: "AAA"}},
				{Chunk: domain.Chunk{Content: "BBB"}},
				{Chunk: domain.Chunk{Content: "CCC"}},
			},
		},
	}

	p := NewPipelineWithConfig(store, llm, fixedEmbedder{}, PipelineConfig{
		MaxContextChunks: 2,
		MaxContextChars:  5,
	})
	_, err := p.Answer(context.Background(), "Q", 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := "Контекст:\nAAA\nB\n\nВопрос:\nQ"
	if llm.userMessage != want {
		t.Fatalf("unexpected user message:\nwant=%q\ngot=%q", want, llm.userMessage)
	}
}
