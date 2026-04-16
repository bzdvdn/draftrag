package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestSelectMMR_InvalidLambda(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{0.1, 0.2}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{0.3, 0.4},
			},
			Score: 0.9,
		},
	}

	// Lambda < 0
	_, err := selectMMR(ctx, queryEmbedding, candidates, 5, -0.1)
	if err == nil {
		t.Fatal("expected error for lambda < 0, got nil")
	}
	if err != errMMRInvalidLambda {
		t.Errorf("expected errMMRInvalidLambda, got %v", err)
	}

	// Lambda > 1
	_, err = selectMMR(ctx, queryEmbedding, candidates, 5, 1.1)
	if err == nil {
		t.Fatal("expected error for lambda > 1, got nil")
	}
	if err != errMMRInvalidLambda {
		t.Errorf("expected errMMRInvalidLambda, got %v", err)
	}
}

func TestSelectMMR_EmptyQueryVector(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{0.3, 0.4},
			},
			Score: 0.9,
		},
	}

	_, err := selectMMR(ctx, queryEmbedding, candidates, 5, 0.5)
	if err == nil {
		t.Fatal("expected error for empty query embedding, got nil")
	}
	if err != errMMREmptyQueryVector {
		t.Errorf("expected errMMREmptyQueryVector, got %v", err)
	}
}

func TestSelectMMR_EmbeddingMissing(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{0.1, 0.2}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{}, // пустой embedding
			},
			Score: 0.9,
		},
	}

	_, err := selectMMR(ctx, queryEmbedding, candidates, 5, 0.5)
	if err == nil {
		t.Fatal("expected error for missing chunk embedding, got nil")
	}
	if err != errMMREmbeddingMissing {
		t.Errorf("expected errMMREmbeddingMissing, got %v", err)
	}
}

func TestSelectMMR_DimMismatch(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{0.1, 0.2}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{0.3, 0.4, 0.5}, // другая размерность
			},
			Score: 0.9,
		},
	}

	_, err := selectMMR(ctx, queryEmbedding, candidates, 5, 0.5)
	if err == nil {
		t.Fatal("expected error for dimension mismatch, got nil")
	}
	if err != errMMRDimMismatch {
		t.Errorf("expected errMMRDimMismatch, got %v", err)
	}
}

func TestSelectMMR_EmptyCandidates(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{0.1, 0.2}
	candidates := []domain.RetrievedChunk{}

	result, err := selectMMR(ctx, queryEmbedding, candidates, 5, 0.5)
	if err != nil {
		t.Fatalf("expected no error for empty candidates, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty candidates, got %v", result)
	}
}

func TestSelectMMR_TopKZero(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{0.1, 0.2}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{0.3, 0.4},
			},
			Score: 0.9,
		},
	}

	_, err := selectMMR(ctx, queryEmbedding, candidates, 0, 0.5)
	if err == nil {
		t.Fatal("expected error for topK <= 0, got nil")
	}
}

func TestSelectMMR_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем контекст

	queryEmbedding := []float64{0.1, 0.2}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{0.3, 0.4},
			},
			Score: 0.9,
		},
	}

	_, err := selectMMR(ctx, queryEmbedding, candidates, 5, 0.5)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestSelectMMR_Success(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{1.0, 0.0}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{1.0, 0.0}, // похож на query
			},
			Score: 0.9,
		},
		{
			Chunk: domain.Chunk{
				ID:        "c2",
				Embedding: []float64{0.0, 1.0}, // ортогонален
			},
			Score: 0.8,
		},
	}

	result, err := selectMMR(ctx, queryEmbedding, candidates, 2, 0.5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestSelectMMR_TopKLargerThanCandidates(t *testing.T) {
	ctx := context.Background()
	queryEmbedding := []float64{1.0, 0.0}
	candidates := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:        "c1",
				Embedding: []float64{1.0, 0.0},
			},
			Score: 0.9,
		},
	}

	result, err := selectMMR(ctx, queryEmbedding, candidates, 10, 0.5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 result (limited by candidates), got %d", len(result))
	}
}

func TestCosine(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{"identical vectors", []float64{1.0, 0.0}, []float64{1.0, 0.0}, 1.0},
		{"orthogonal vectors", []float64{1.0, 0.0}, []float64{0.0, 1.0}, 0.0},
		{"opposite vectors", []float64{1.0, 0.0}, []float64{-1.0, 0.0}, -1.0},
		{"45 degrees", []float64{1.0, 0.0}, []float64{0.707, 0.707}, 0.707},
		{"empty vectors", []float64{}, []float64{}, 0.0},
		{"dimension mismatch", []float64{1.0}, []float64{1.0, 0.0}, 0.0},
		{"zero vector a", []float64{0.0, 0.0}, []float64{1.0, 0.0}, 0.0},
		{"zero vector b", []float64{1.0, 0.0}, []float64{0.0, 0.0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosine(tt.a, tt.b)
			// Допускаем небольшую погрешность для floating point
			if result < tt.expected-0.01 || result > tt.expected+0.01 {
				t.Errorf("cosine() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaxCosineToSelected(t *testing.T) {
	selected := []mmrCandidate{
		{
			rc: domain.RetrievedChunk{
				Chunk: domain.Chunk{
					Embedding: []float64{1.0, 0.0},
				},
			},
		},
	}

	tests := []struct {
		name      string
		embedding []float64
		expected  float64
	}{
		{"identical to selected", []float64{1.0, 0.0}, 1.0},
		{"orthogonal to selected", []float64{0.0, 1.0}, 0.0},
		{"45 degrees to selected", []float64{0.707, 0.707}, 0.707},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxCosineToSelected(tt.embedding, selected)
			// Допускаем небольшую погрешность
			if result < tt.expected-0.01 || result > tt.expected+0.01 {
				t.Errorf("maxCosineToSelected() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaxCosineToSelected_EmptySelected(t *testing.T) {
	embedding := []float64{1.0, 0.0}
	result := maxCosineToSelected(embedding, []mmrCandidate{})
	if result != 0 {
		t.Errorf("expected 0 for empty selected, got %v", result)
	}
}
