package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestPipeline_AnswerWithInlineCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerWithInlineCitations(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerWithInlineCitations_GenerateError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &errorLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerWithInlineCitations(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for generate failure, got nil")
	}
}

func TestPipeline_AnswerWithInlineCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, citations, err := p.AnswerWithInlineCitations(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
	_ = citations
}

func TestPipeline_AnswerWithInlineCitationsWithParentIDs_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerWithInlineCitationsWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerWithInlineCitationsWithParentIDs_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, citations, err := p.AnswerWithInlineCitationsWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
	_ = citations
}

func TestPipeline_AnswerWithInlineCitationsWithMetadataFilter_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	_, _, _, err := p.AnswerWithInlineCitationsWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerWithInlineCitationsWithMetadataFilter_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	answer, result, citations, err := p.AnswerWithInlineCitationsWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
	_ = citations
}

func TestPipeline_AnswerHyDEWithInlineCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerHyDEWithInlineCitations(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHyDEWithInlineCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, citations, err := p.AnswerHyDEWithInlineCitations(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
	_ = citations
}

func TestPipeline_AnswerMultiWithInlineCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerMultiWithInlineCitations(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerMultiWithInlineCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, citations, err := p.AnswerMultiWithInlineCitations(context.Background(), "test query", 3, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
	_ = citations
}

func TestPipeline_AnswerHybridWithInlineCitations_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, _, err := p.AnswerHybridWithInlineCitations(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHybridWithInlineCitations_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	answer, result, citations, err := p.AnswerHybridWithInlineCitations(context.Background(), "test query", 5, config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
	_ = citations
}

func TestPipeline_AnswerHybridWithInlineCitations_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, _, err := p.AnswerHybridWithInlineCitations(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}
