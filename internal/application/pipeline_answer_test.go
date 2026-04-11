package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type recordingEmbedder struct {
	calls *[]string
}

func (e recordingEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	*e.calls = append(*e.calls, "embed:"+text)
	return []float64{1, 2, 3}, nil
}

type recordingStore struct {
	calls  *[]string
	result domain.RetrievalResult
}

func (s recordingStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	*s.calls = append(*s.calls, "upsert")
	return nil
}

func (s recordingStore) Delete(ctx context.Context, id string) error {
	*s.calls = append(*s.calls, "delete")
	return nil
}

func (s recordingStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	*s.calls = append(*s.calls, "search")
	return s.result, nil
}

type recordingLLM struct {
	calls        *[]string
	systemPrompt string
	userMessage  string
}

func (l *recordingLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	*l.calls = append(*l.calls, "generate")
	l.systemPrompt = systemPrompt
	l.userMessage = userMessage
	return "answer", nil
}

func TestPipeline_Answer_CallsOrderAndReturnsAnswer(t *testing.T) {
	var calls []string
	llm := &recordingLLM{calls: &calls}

	p := NewPipeline(
		recordingStore{
			calls: &calls,
			result: domain.RetrievalResult{
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{Content: "chunk-1"}},
					{Chunk: domain.Chunk{Content: "chunk-2"}},
				},
			},
		},
		llm,
		recordingEmbedder{calls: &calls},
	)

	got, err := p.Answer(context.Background(), "what?", 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != "answer" {
		t.Fatalf("expected %q, got %q", "answer", got)
	}

	wantCalls := []string{"embed:what?", "search", "generate"}
	if len(calls) != len(wantCalls) {
		t.Fatalf("expected %d calls, got %d: %#v", len(wantCalls), len(calls), calls)
	}
	for i := range wantCalls {
		if calls[i] != wantCalls[i] {
			t.Fatalf("call[%d]: expected %q, got %q", i, wantCalls[i], calls[i])
		}
	}
}

func TestPipeline_Answer_PromptContractV1(t *testing.T) {
	var calls []string
	llm := &recordingLLM{calls: &calls}

	p := NewPipeline(
		recordingStore{
			calls: &calls,
			result: domain.RetrievalResult{
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{Content: "A"}},
					{Chunk: domain.Chunk{Content: "B"}},
				},
			},
		},
		llm,
		recordingEmbedder{calls: &calls},
	)

	_, err := p.Answer(context.Background(), "Q", 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if llm.systemPrompt != defaultSystemPromptV1 {
		t.Fatalf("unexpected system prompt: %q", llm.systemPrompt)
	}
	wantUser := "Контекст:\nA\nB\n\nВопрос:\nQ"
	if llm.userMessage != wantUser {
		t.Fatalf("unexpected user message:\nwant=%q\ngot=%q", wantUser, llm.userMessage)
	}
}

func TestPipeline_Answer_ContextCanceledFastAndNoCalls(t *testing.T) {
	var calls []string
	llm := &recordingLLM{calls: &calls}

	p := NewPipeline(
		recordingStore{calls: &calls},
		llm,
		recordingEmbedder{calls: &calls},
	)

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	_, err := p.Answer(ctx, "Q", 2)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
	if len(calls) != 0 {
		t.Fatalf("expected no dependency calls, got %#v", calls)
	}
}
