package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type vectorStoreWithFilters struct {
	mockVectorStore
}

func (m *vectorStoreWithFilters) SearchWithFilter(_ context.Context, _ []float64, _ int, _ domain.ParentIDFilter) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

func (m *vectorStoreWithFilters) SearchWithMetadataFilter(_ context.Context, _ []float64, _ int, _ domain.MetadataFilter) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

func TestPipeline_QueryWithParentIDs_EmptyParentIDs(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	// Пустой parentIDs должен делегироваться в обычный Query
	result, err := p.QueryWithParentIDs(context.Background(), "test query", 5, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_QueryWithParentIDs_WithParentIDs(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	parentIDs := []string{"doc1", "doc2"}
	result, err := p.QueryWithParentIDs(context.Background(), "test query", 5, parentIDs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_QueryWithParentIDs_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.QueryWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_QueryWithMetadataFilter_EmptyFields(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{}
	// Пустые fields должны делегироваться в обычный Query
	result, err := p.QueryWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_QueryWithMetadataFilter_WithFields(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{
		Fields: map[string]string{"source": "wiki"},
	}
	result, err := p.QueryWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_QueryWithMetadataFilter_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{
		Fields: map[string]string{"source": "wiki"},
	}
	_, err := p.QueryWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerWithParentIDs_EmptyParentIDs(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	// Пустой parentIDs должен делегироваться в обычный Answer
	answer, err := p.AnswerWithParentIDs(context.Background(), "test query", 5, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestPipeline_AnswerWithParentIDs_WithParentIDs(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	parentIDs := []string{"doc1", "doc2"}
	answer, err := p.AnswerWithParentIDs(context.Background(), "test query", 5, parentIDs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestPipeline_AnswerWithMetadataFilter_EmptyFields(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{}
	answer, err := p.AnswerWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestPipeline_AnswerWithMetadataFilter_WithFields(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{
		Fields: map[string]string{"source": "wiki"},
	}
	answer, err := p.AnswerWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

type documentStore struct {
	mockVectorStore
}

func (m *documentStore) DeleteByParentID(_ context.Context, _ string) error {
	return nil
}

func TestPipeline_DeleteDocument_Success(t *testing.T) {
	store := &documentStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	err := p.DeleteDocument(context.Background(), "doc1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPipeline_DeleteDocument_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует DocumentStore
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	err := p.DeleteDocument(context.Background(), "doc1")
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}

func TestPipeline_UpdateDocument_Success(t *testing.T) {
	store := &documentStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	doc := domain.Document{
		ID:      "doc1",
		Content: "updated content",
	}

	err := p.UpdateDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPipeline_UpdateDocument_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует DocumentStore
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	doc := domain.Document{
		ID:      "doc1",
		Content: "updated content",
	}

	err := p.UpdateDocument(context.Background(), doc)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}

func TestPipeline_AnswerWithCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, err := p.AnswerWithCitations(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
}

func TestPipeline_AnswerWithCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerWithCitations(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}
