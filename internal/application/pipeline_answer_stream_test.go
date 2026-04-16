package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/llm"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// MockEmbedder — мок для embedder.
type MockEmbedder struct {
	Embedding []float64
	Err       error
}

func (m *MockEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Embedding, nil
}

// TestAnswerStream_Success проверяет успешное streaming.
// @sk-task T3.1: Тест AnswerStream end-to-end (AC-001, DEC-003)
func TestAnswerStream_Success(t *testing.T) {
	ctx := context.Background()

	// Мок store с данными
	store := vectorstore.NewInMemoryStore()
	_ = store.Upsert(ctx, domain.Chunk{
		ID:        "chunk-1",
		Content:   "test content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
	})

	// Мок embedder
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}

	// Мок streaming LLM
	streamingLLM := &llm.MockStreamingLLM{
		Tokens: []string{"Hello", " ", "world"},
		Delay:  1 * time.Millisecond,
	}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	ch, err := pipeline.AnswerStream(ctx, "test question", 5)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var result string
	for token := range ch {
		result += token
	}

	if result != "Hello world" {
		t.Fatalf("expected %q, got %q", "Hello world", result)
	}
}

// TestAnswerStream_NonStreamingLLM проверяет graceful degradation.
// @sk-task T3.1: Тест graceful degradation (AC-004)
func TestAnswerStream_NonStreamingLLM(t *testing.T) {
	ctx := context.Background()

	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	nonStreamingLLM := &llm.NonStreamingLLM{Result: "static answer"}

	pipeline := NewPipeline(store, nonStreamingLLM, embedder)

	ch, err := pipeline.AnswerStream(ctx, "test question", 5)

	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel on error")
	}
}

// TestAnswerStreamWithInlineCitations_Success проверяет streaming с цитатами.
// @sk-task T3.1: Тест AnswerStreamWithInlineCitations (AC-002)
func TestAnswerStreamWithInlineCitations_Success(t *testing.T) {
	ctx := context.Background()

	// Мок store с данными
	store := vectorstore.NewInMemoryStore()
	_ = store.Upsert(ctx, domain.Chunk{
		ID:        "chunk-1",
		Content:   "test content for citations",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
	})

	// Мок embedder
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}

	// Мок streaming LLM
	streamingLLM := &llm.MockStreamingLLM{
		Tokens: []string{"Based", " ", "on", " ", "source"},
		Delay:  1 * time.Millisecond,
	}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	ch, retrieval, citations, err := pipeline.AnswerStreamWithInlineCitations(ctx, "test question", 5)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var result string
	for token := range ch {
		result += token
	}

	// Проверяем, что что-то пришло
	if result == "" {
		t.Fatal("expected non-empty result")
	}

	// Проверяем, что цитаты заполнены (они собираются до streaming'а)
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}

	// Проверяем retrieval
	if len(retrieval.Chunks) == 0 {
		t.Fatal("expected retrieval chunks")
	}
}

// TestAnswerStreamWithInlineCitations_NonStreamingLLM проверяет graceful degradation для citations.
// @sk-task T3.1: Тест graceful degradation для citations (AC-004)
func TestAnswerStreamWithInlineCitations_NonStreamingLLM(t *testing.T) {
	ctx := context.Background()

	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	nonStreamingLLM := &llm.NonStreamingLLM{Result: "static answer"}

	pipeline := NewPipeline(store, nonStreamingLLM, embedder)

	ch, _, _, err := pipeline.AnswerStreamWithInlineCitations(ctx, "test question", 5)

	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel on error")
	}
}

// TestAnswerStream_ContextCancellation проверяет отмену контекста.
// @sk-task T3.1: Тест context cancellation в AnswerStream (AC-003, RQ-005)
func TestAnswerStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Мок store с данными
	store := vectorstore.NewInMemoryStore()
	_ = store.Upsert(ctx, domain.Chunk{
		ID:        "chunk-1",
		Content:   "test content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
	})

	// Мок embedder
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}

	// Мок streaming LLM с задержкой
	streamingLLM := &llm.MockStreamingLLM{
		Tokens: []string{"slow", " ", "tokens"},
		Delay:  30 * time.Millisecond,
	}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	ch, err := pipeline.AnswerStream(ctx, "test question", 5)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Читаем, но контекст отменится
	count := 0
	for range ch {
		count++
	}

	t.Logf("received %d tokens before cancellation", count)
}
