package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// Дополнительные mock для тестирования методов

type errorVectorStore struct{}

func (m *errorVectorStore) Health(_ context.Context) error { return nil }
func (m *errorVectorStore) Upsert(_ context.Context, _ domain.Chunk) error {
	return errors.New("upsert failed")
}

func (m *errorVectorStore) Delete(_ context.Context, _ string) error {
	return errors.New("delete failed")
}

func (m *errorVectorStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, errors.New("search failed")
}

type errorLLMProvider struct{}

func (m *errorLLMProvider) Health(_ context.Context) error { return nil }
func (m *errorLLMProvider) Generate(_ context.Context, _, _ string) (string, error) {
	return "", errors.New("generate failed")
}

type errorEmbedder struct{}

func (m *errorEmbedder) Health(_ context.Context) error { return nil }
func (m *errorEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return nil, errors.New("embed failed")
}

func TestPipeline_Index_EmbedError(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err = p.Index(context.Background(), docs)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_Index_UpsertError(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &errorVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err = p.Index(context.Background(), docs)
	if err == nil {
		t.Fatal("expected error for upsert failure, got nil")
	}
}

func TestPipeline_Index_WithoutChunker(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err = p.Index(context.Background(), docs)
	// Без chunker документ должен индексироваться как один чанк
	if err != nil {
		t.Fatalf("expected no error for index without chunker, got %v", err)
	}
}

func TestPipeline_Index_WithChunker(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}
	chunker := &testChunker{}

	p, err := NewPipelineWithConfig(store, llm, embedder, PipelineOptions{
		Chunker: chunker,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err = p.Index(context.Background(), docs)
	if err != nil {
		t.Fatalf("expected no error for index with chunker, got %v", err)
	}
}

func TestPipeline_Index_ContextCancellation(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем

	docs := []domain.Document{
		{
			ID:      "doc1",
			Content: "test content",
		},
	}

	err = p.Index(ctx, docs)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestPipeline_Query_SearchError(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &errorVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Query(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for search failure, got nil")
	}
}

func TestPipeline_Query_EmbedError(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Query(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_Query_Success(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.Query(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// mockVectorStore возвращает пустой result
	if len(result.Chunks) == 0 {
		t.Log("пустой result ожидаем для mockVectorStore")
	}
}

func TestPipeline_Answer_GenerateError(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &errorLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for generate failure, got nil")
	}
}

func TestPipeline_Answer_EmptyResult(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	// mockVectorStore возвращает пустой result
	_, err = p.Answer(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error for empty result, got %v", err)
	}
}

func TestPipeline_Answer_ContextCancellation(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p, err := NewPipeline(store, llm, embedder)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = p.Answer(ctx, "test query", 5)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}
