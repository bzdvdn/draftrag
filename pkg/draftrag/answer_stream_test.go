package draftrag

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

// TestPipeline_AnswerStream_Success проверяет streaming через public API.
// @sk-task T3.1: Тест AnswerStream в public API (AC-001)
func TestPipeline_AnswerStream_Success(t *testing.T) {
	ctx := context.Background()

	// Инициализация компонентов
	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens: []string{"Hello", " ", "from", " ", "API"},
		Delay:  1 * time.Millisecond,
	}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	// Добавляем тестовые данные
	_ = store.Upsert(ctx, domain.Chunk{
		ID:        "chunk-1",
		Content:   "test content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
	})

	ch, err := pipeline.Search("test").TopK(5).Stream(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var result string
	for token := range ch {
		result += token
	}

	if result != "Hello from API" {
		t.Fatalf("expected %q, got %q", "Hello from API", result)
	}
}

// TestPipeline_AnswerStream_Validation проверяет валидацию входных данных.
// @sk-task T3.1: Тест валидации входных данных AnswerStream (AC-001)
func TestPipeline_AnswerStream_Validation(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"ok"}}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	tests := []struct {
		name     string
		question string
		topK     int
		wantErr  error
	}{
		{
			name:     "empty question",
			question: "",
			topK:     5,
			wantErr:  ErrEmptyQuery,
		},
		{
			name:     "invalid topK",
			question: "valid",
			topK:     0,
			wantErr:  ErrInvalidTopK,
		},
		{
			name:     "negative topK",
			question: "valid",
			topK:     -1,
			wantErr:  ErrInvalidTopK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ch, err := pipeline.Search(tt.question).TopK(tt.topK).Stream(ctx)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if ch != nil {
				t.Fatal("expected nil channel on error")
			}
		})
	}
}

// TestPipeline_AnswerStreamWithInlineCitations_Success проверяет streaming с цитатами.
// @sk-task T3.1: Тест AnswerStreamWithInlineCitations в public API (AC-002)
func TestPipeline_AnswerStreamWithInlineCitations_Success(t *testing.T) {
	ctx := context.Background()

	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens: []string{"Result", " ", "with", " ", "citations"},
		Delay:  1 * time.Millisecond,
	}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	// Добавляем тестовые данные
	_ = store.Upsert(ctx, domain.Chunk{
		ID:        "chunk-1",
		Content:   "citation source content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
	})

	ch, retrieval, citations, err := pipeline.Search("test").TopK(5).StreamCite(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var result string
	for token := range ch {
		result += token
	}

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	if len(citations) == 0 {
		t.Fatal("expected citations")
	}

	if len(retrieval.Chunks) == 0 {
		t.Fatal("expected retrieval chunks")
	}
}

// TestPipeline_AnswerStreamWithInlineCitations_Validation проверяет валидацию.
// @sk-task T3.1: Тест валидации AnswerStreamWithInlineCitations (AC-002)
func TestPipeline_AnswerStreamWithInlineCitations_Validation(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"ok"}}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	tests := []struct {
		name     string
		question string
		topK     int
		wantErr  error
	}{
		{
			name:     "empty question",
			question: "",
			topK:     5,
			wantErr:  ErrEmptyQuery,
		},
		{
			name:     "zero topK",
			question: "valid",
			topK:     0,
			wantErr:  ErrInvalidTopK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ch, _, _, err := pipeline.Search(tt.question).TopK(tt.topK).StreamCite(ctx)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if ch != nil {
				t.Fatal("expected nil channel on error")
			}
		})
	}
}

// TestPipeline_AnswerStream_NonStreamingLLM проверяет graceful degradation.
// @sk-task T3.1: Тест graceful degradation в public API (AC-004)
func TestPipeline_AnswerStream_NonStreamingLLM(t *testing.T) {
	ctx := context.Background()

	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	nonStreamingLLM := &llm.NonStreamingLLM{Result: "static"}

	pipeline := NewPipeline(store, nonStreamingLLM, embedder)

	ch, err := pipeline.Search("test").TopK(5).Stream(ctx)

	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel on error")
	}
}

// TestPipeline_AnswerStreamWithInlineCitations_NonStreamingLLM проверяет graceful degradation для citations.
// @sk-task T3.1: Тест graceful degradation для citations в public API (AC-004)
func TestPipeline_AnswerStreamWithInlineCitations_NonStreamingLLM(t *testing.T) {
	ctx := context.Background()

	store := vectorstore.NewInMemoryStore()
	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	nonStreamingLLM := &llm.NonStreamingLLM{Result: "static"}

	pipeline := NewPipeline(store, nonStreamingLLM, embedder)

	ch, _, _, err := pipeline.Search("test").TopK(5).StreamCite(ctx)

	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel on error")
	}
}

// TestPipeline_AnswerStream_ContextCancellation проверяет отмену контекста.
// @sk-task T3.1: Тест context cancellation в public API (AC-003, RQ-005)
func TestPipeline_AnswerStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	store := vectorstore.NewInMemoryStore()
	_ = store.Upsert(ctx, domain.Chunk{
		ID:        "chunk-1",
		Content:   "test",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
	})

	embedder := &MockEmbedder{Embedding: []float64{0.1, 0.2, 0.3}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens: []string{"slow", " ", "stream"},
		Delay:  30 * time.Millisecond,
	}

	pipeline := NewPipeline(store, streamingLLM, embedder)

	ch, err := pipeline.Search("test").TopK(5).Stream(ctx)
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
