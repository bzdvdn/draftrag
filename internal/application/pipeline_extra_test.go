package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestPipeline_AnswerWithCitationsWithParentIDs_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerWithCitationsWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerWithCitationsWithParentIDs_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, err := p.AnswerWithCitationsWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
}

func TestPipeline_AnswerWithCitationsWithMetadataFilter_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	_, _, err := p.AnswerWithCitationsWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerWithCitationsWithMetadataFilter_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	answer, result, err := p.AnswerWithCitationsWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
}

func TestPipeline_AnswerHyDEWithCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerHyDEWithCitations(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHyDEWithCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, err := p.AnswerHyDEWithCitations(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
}

func TestPipeline_AnswerMultiWithCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerMultiWithCitations(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerMultiWithCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, result, err := p.AnswerMultiWithCitations(context.Background(), "test query", 3, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
}

func TestPipeline_AnswerHybridWithCitations_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, err := p.AnswerHybridWithCitations(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHybridWithCitations_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	answer, result, err := p.AnswerHybridWithCitations(context.Background(), "test query", 5, config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
	_ = result
}

func TestPipeline_AnswerHybridWithCitations_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, err := p.AnswerHybridWithCitations(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}

func TestPipeline_AnswerHyDEStream_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerHyDEStream(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHyDEStream_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, err := p.AnswerHyDEStream(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tokens := []string{}
	for token := range stream {
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		t.Error("expected at least one token")
	}
}

func TestPipeline_AnswerMultiStream_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerMultiStream(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerMultiStream_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, err := p.AnswerMultiStream(context.Background(), "test query", 3, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tokens := []string{}
	for token := range stream {
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		t.Error("expected at least one token")
	}
}

func TestPipeline_AnswerHybridStream_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, err := p.AnswerHybridStream(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHybridStream_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	stream, err := p.AnswerHybridStream(context.Background(), "test query", 5, config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tokens := []string{}
	for token := range stream {
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		t.Error("expected at least one token")
	}
}

func TestPipeline_AnswerHybridStream_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, err := p.AnswerHybridStream(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}
