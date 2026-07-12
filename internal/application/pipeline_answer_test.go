package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type recordingEmbedder struct {
	calls *[]string
}

func (e recordingEmbedder) Health(_ context.Context) error { return nil }
func (e recordingEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	*e.calls = append(*e.calls, "embed:"+text)
	return []float64{1, 2, 3}, nil
}

type recordingStore struct {
	mu     sync.Mutex
	calls  *[]string
	result domain.RetrievalResult
}

func (s *recordingStore) Health(_ context.Context) error { return nil }
func (s *recordingStore) Upsert(_ context.Context, _ domain.Chunk) error {
	s.mu.Lock()
	*s.calls = append(*s.calls, "upsert")
	s.mu.Unlock()
	return nil
}

func (s *recordingStore) Delete(_ context.Context, _ string) error {
	s.mu.Lock()
	*s.calls = append(*s.calls, "delete")
	s.mu.Unlock()
	return nil
}

func (s *recordingStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	s.mu.Lock()
	*s.calls = append(*s.calls, "search")
	s.mu.Unlock()
	return s.result, nil
}

type recordingLLM struct {
	calls        *[]string
	systemPrompt string
	userMessage  string
}

func (l *recordingLLM) Health(_ context.Context) error { return nil }
func (l *recordingLLM) Generate(_ context.Context, systemPrompt, userMessage string) (string, error) {
	*l.calls = append(*l.calls, "generate")
	l.systemPrompt = systemPrompt
	l.userMessage = userMessage
	return "answer", nil
}

func TestPipeline_Answer_CallsOrderAndReturnsAnswer(t *testing.T) {
	var calls []string
	llm := &recordingLLM{calls: &calls}

	p, err := NewPipeline(
		&recordingStore{
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
	if err != nil {
		t.Fatal(err)
	}

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

	p, err := NewPipeline(
		&recordingStore{
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
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "Q", 2)
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

	p, err := NewPipeline(
		&recordingStore{calls: &calls},
		llm,
		recordingEmbedder{calls: &calls},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	_, err = p.Answer(ctx, "Q", 2)
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
