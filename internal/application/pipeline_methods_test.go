package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Дополнительные mock для тестирования методов

type errorVectorStore struct{}

func (m *errorVectorStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	return errors.New("upsert failed")
}

func (m *errorVectorStore) Delete(ctx context.Context, id string) error {
	return errors.New("delete failed")
}

func (m *errorVectorStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, errors.New("search failed")
}

type errorLLMProvider struct{}

func (m *errorLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "", errors.New("generate failed")
}

type errorEmbedder struct{}

func (m *errorEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return nil, errors.New("embed failed")
}

func TestPipeline_Index_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err := p.Index(context.Background(), docs)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_Index_UpsertError(t *testing.T) {
	store := &errorVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err := p.Index(context.Background(), docs)
	if err == nil {
		t.Fatal("expected error for upsert failure, got nil")
	}
}

func TestPipeline_Index_WithoutChunker(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err := p.Index(context.Background(), docs)
	// Без chunker документ должен индексироваться как один чанк
	if err != nil {
		t.Fatalf("expected no error for index without chunker, got %v", err)
	}
}

func TestPipeline_Index_WithChunker(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}
	chunker := &testChunker{}

	p := NewPipelineWithConfig(store, llm, embedder, PipelineConfig{
		Chunker: chunker,
	})

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err := p.Index(context.Background(), docs)
	if err != nil {
		t.Fatalf("expected no error for index with chunker, got %v", err)
	}
}

func TestPipeline_Index_ContextCancellation(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err := p.Index(ctx, docs)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestPipeline_Query_SearchError(t *testing.T) {
	store := &errorVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.Query(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for search failure, got nil")
	}
}

func TestPipeline_Query_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.Query(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_Query_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	result, err := p.Query(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// mockVectorStore возвращает пустой result
	if len(result.Chunks) == 0 {
		// это ожидаемо для mock
	}
}

func TestPipeline_Answer_GenerateError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &errorLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.Answer(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for generate failure, got nil")
	}
}

func TestPipeline_Answer_EmptyResult(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	// mockVectorStore возвращает пустой result
	_, err := p.Answer(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error for empty result, got %v", err)
	}
}

func TestPipeline_Answer_ContextCancellation(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.Answer(ctx, "test query", 5)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}
