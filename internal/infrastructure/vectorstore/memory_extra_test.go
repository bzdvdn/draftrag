package vectorstore

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestMatchesMetadataFilter_EmptyFields(t *testing.T) {
	metadata := map[string]string{"source": "wiki", "lang": "ru"}
	fields := map[string]string{}

	if !matchesMetadataFilter(metadata, fields) {
		t.Error("expected true for empty fields")
	}
}

func TestMatchesMetadataFilter_Match(t *testing.T) {
	metadata := map[string]string{"source": "wiki", "lang": "ru"}
	fields := map[string]string{"source": "wiki"}

	if !matchesMetadataFilter(metadata, fields) {
		t.Error("expected true for matching field")
	}
}

func TestMatchesMetadataFilter_MultipleFieldsMatch(t *testing.T) {
	metadata := map[string]string{"source": "wiki", "lang": "ru"}
	fields := map[string]string{"source": "wiki", "lang": "ru"}

	if !matchesMetadataFilter(metadata, fields) {
		t.Error("expected true for matching all fields")
	}
}

func TestMatchesMetadataFilter_NoMatch(t *testing.T) {
	metadata := map[string]string{"source": "wiki", "lang": "ru"}
	fields := map[string]string{"source": "docs"}

	if matchesMetadataFilter(metadata, fields) {
		t.Error("expected false for non-matching field")
	}
}

func TestMatchesMetadataFilter_PartialMatch(t *testing.T) {
	metadata := map[string]string{"source": "wiki", "lang": "ru"}
	fields := map[string]string{"source": "wiki", "lang": "en"}

	if matchesMetadataFilter(metadata, fields) {
		t.Error("expected false for partial match (AND logic)")
	}
}

func TestMatchesMetadataFilter_NilMetadata(t *testing.T) {
	var metadata map[string]string
	fields := map[string]string{"source": "wiki"}

	if matchesMetadataFilter(metadata, fields) {
		t.Error("expected false for nil metadata with non-empty fields")
	}
}

func TestMatchesMetadataFilter_NilFields(t *testing.T) {
	metadata := map[string]string{"source": "wiki"}
	var fields map[string]string

	if !matchesMetadataFilter(metadata, fields) {
		t.Error("expected true for nil fields")
	}
}

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := []float64{1.0, 0.0, 0.0}
	b := []float64{1.0, 0.0, 0.0}

	result := cosineSimilarity(a, b)
	if result < 0.99 || result > 1.01 {
		t.Errorf("expected ~1.0 for identical vectors, got %f", result)
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	a := []float64{1.0, 0.0}
	b := []float64{0.0, 1.0}

	result := cosineSimilarity(a, b)
	if result < -0.01 || result > 0.01 {
		t.Errorf("expected ~0.0 for orthogonal vectors, got %f", result)
	}
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	a := []float64{1.0, 0.0}
	b := []float64{-1.0, 0.0}

	result := cosineSimilarity(a, b)
	if result < -1.01 || result > -0.99 {
		t.Errorf("expected ~-1.0 for opposite vectors, got %f", result)
	}
}

func TestCosineSimilarity_DifferentDimensions(t *testing.T) {
	a := []float64{1.0, 0.0}
	b := []float64{1.0, 0.0, 0.0}

	result := cosineSimilarity(a, b)
	if result != 0 {
		t.Errorf("expected 0 for different dimensions, got %f", result)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float64{0.0, 0.0}
	b := []float64{1.0, 0.0}

	result := cosineSimilarity(a, b)
	if result != 0 {
		t.Errorf("expected 0 for zero vector, got %f", result)
	}
}

func TestCosineSimilarity_Clamped(t *testing.T) {
	// Проверяем, что значение зажимается в [-1, 1]
	a := []float64{1.0, 0.0}
	b := []float64{1.0, 0.0}

	result := cosineSimilarity(a, b)
	if result > 1.0 {
		t.Errorf("expected result <= 1.0, got %f", result)
	}
	if result < -1.0 {
		t.Errorf("expected result >= -1.0, got %f", result)
	}
}

func TestInMemoryStore_SearchWithFilter_EmbeddingNil(t *testing.T) {
	store := NewInMemoryStore()
	
	chunk1 := domain.Chunk{
		ID:        "c1",
		ParentID:  "doc1",
		Embedding: nil, // без embedding
	}
	
	store.Upsert(context.Background(), chunk1)
	
	embedding := []float64{1.0, 0.0}
	result, err := store.Search(context.Background(), embedding, 5)
	
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Chunks) != 0 {
		t.Error("expected 0 chunks for nil embedding")
	}
}

func TestInMemoryStore_SearchWithMetadataFilter_EmbeddingNil(t *testing.T) {
	store := NewInMemoryStore()
	
	chunk1 := domain.Chunk{
		ID:        "c1",
		ParentID:  "doc1",
		Metadata:  map[string]string{"source": "wiki"},
		Embedding: nil, // без embedding
	}
	
	store.Upsert(context.Background(), chunk1)
	
	embedding := []float64{1.0, 0.0}
	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	result, err := store.SearchWithMetadataFilter(context.Background(), embedding, 5, filter)
	
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Chunks) != 0 {
		t.Error("expected 0 chunks for nil embedding with metadata filter")
	}
}
