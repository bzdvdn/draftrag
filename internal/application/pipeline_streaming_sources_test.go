package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestPipeline_AnswerStreamWithParentIDs_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerStreamWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerStreamWithParentIDs_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, err := p.AnswerStreamWithParentIDs(context.Background(), "test query", 5, []string{"doc1"})
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

func TestPipeline_AnswerStreamWithMetadataFilter_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	_, err := p.AnswerStreamWithMetadataFilter(context.Background(), "test query", 5, filter)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerStreamWithMetadataFilter_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	stream, err := p.AnswerStreamWithMetadataFilter(context.Background(), "test query", 5, filter)
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

func TestPipeline_AnswerHyDEStreamWithSources_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerHyDEStreamWithSources(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHyDEStreamWithSources_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, result, err := p.AnswerHyDEStreamWithSources(context.Background(), "test query", 5)
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
	_ = result
}

func TestPipeline_AnswerMultiStreamWithSources_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerMultiStreamWithSources(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerMultiStreamWithSources_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, result, err := p.AnswerMultiStreamWithSources(context.Background(), "test query", 3, 5)
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
	_ = result
}

func TestPipeline_AnswerHybridStreamWithSources_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, err := p.AnswerHybridStreamWithSources(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHybridStreamWithSources_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	stream, result, err := p.AnswerHybridStreamWithSources(context.Background(), "test query", 5, config)
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
	_ = result
}

func TestPipeline_AnswerHybridStreamWithSources_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, err := p.AnswerHybridStreamWithSources(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}

func TestPipeline_AnswerStreamWithParentIDsWithSources_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerStreamWithParentIDsWithSources(context.Background(), "test query", 5, []string{"doc1"})
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerStreamWithParentIDsWithSources_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, result, err := p.AnswerStreamWithParentIDsWithSources(context.Background(), "test query", 5, []string{"doc1"})
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
	_ = result
}

func TestPipeline_AnswerStreamWithMetadataFilterWithSources_EmbedError(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	_, _, err := p.AnswerStreamWithMetadataFilterWithSources(context.Background(), "test query", 5, filter)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerStreamWithMetadataFilterWithSources_Success(t *testing.T) {
	store := &vectorStoreWithFilters{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	stream, result, err := p.AnswerStreamWithMetadataFilterWithSources(context.Background(), "test query", 5, filter)
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
	_ = result
}

func TestPipeline_AnswerHyDEStreamWithInlineCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerHyDEStreamWithInlineCitations(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHyDEStreamWithInlineCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, result, citations, err := p.AnswerHyDEStreamWithInlineCitations(context.Background(), "test query", 5)
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
	_ = result
	_ = citations
}

func TestPipeline_AnswerMultiStreamWithInlineCitations_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, _, err := p.AnswerMultiStreamWithInlineCitations(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerMultiStreamWithInlineCitations_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, result, citations, err := p.AnswerMultiStreamWithInlineCitations(context.Background(), "test query", 3, 5)
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
	_ = result
	_ = citations
}

func TestPipeline_AnswerHybridStreamWithInlineCitations_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, _, err := p.AnswerHybridStreamWithInlineCitations(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHybridStreamWithInlineCitations_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	stream, result, citations, err := p.AnswerHybridStreamWithInlineCitations(context.Background(), "test query", 5, config)
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
	_ = result
	_ = citations
}

func TestPipeline_AnswerHybridStreamWithInlineCitations_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, _, _, err := p.AnswerHybridStreamWithInlineCitations(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}
