package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestPipeline_QueryHyDE_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.QueryHyDE(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_QueryHyDE_SearchError(t *testing.T) {
	store := &errorVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.QueryHyDE(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for search failure, got nil")
	}
}

func TestPipeline_QueryHyDE_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	result, err := p.QueryHyDE(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_AnswerHyDE_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerHyDE(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHyDE_GenerateError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &errorLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerHyDE(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for generate failure, got nil")
	}
}

func TestPipeline_AnswerHyDE_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, err := p.AnswerHyDE(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestPipeline_QueryMulti_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.QueryMulti(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_QueryMulti_SearchError(t *testing.T) {
	store := &errorVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.QueryMulti(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for search failure, got nil")
	}
}

func TestPipeline_QueryMulti_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	result, err := p.QueryMulti(context.Background(), "test query", 3, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_AnswerMulti_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerMulti(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerMulti_GenerateError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &errorLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerMulti(context.Background(), "test query", 3, 5)
	if err == nil {
		t.Fatal("expected error for generate failure, got nil")
	}
}

func TestPipeline_AnswerMulti_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	answer, err := p.AnswerMulti(context.Background(), "test query", 3, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

type hybridSearcher struct {
	mockVectorStore
}

func (m *hybridSearcher) SearchHybrid(_ context.Context, _ string, _ []float64, _ int, _ domain.HybridConfig) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

func TestPipeline_QueryHybrid_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, err := p.QueryHybrid(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_QueryHybrid_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, err := p.QueryHybrid(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}

func TestPipeline_QueryHybrid_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	result, err := p.QueryHybrid(context.Background(), "test query", 5, config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_ = result
}

func TestPipeline_AnswerHybrid_EmbedError(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, err := p.AnswerHybrid(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerHybrid_NotSupported(t *testing.T) {
	store := &mockVectorStore{} // не реализует HybridSearcher
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	_, err := p.AnswerHybrid(context.Background(), "test query", 5, config)
	if err == nil {
		t.Fatal("expected error for unsupported operation, got nil")
	}
}

func TestPipeline_AnswerHybrid_Success(t *testing.T) {
	store := &hybridSearcher{}
	llm := &mockLLMProvider{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	config := domain.DefaultHybridConfig()
	answer, err := p.AnswerHybrid(context.Background(), "test query", 5, config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}
