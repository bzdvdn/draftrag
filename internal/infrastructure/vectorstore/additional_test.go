package vectorstore

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestInMemoryStore_ContextCancellation_Search(t *testing.T) {
	store := NewInMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	_, err := store.Search(ctx, queryEmbedding, 5)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestInMemoryStore_ContextCancellation_Upsert(t *testing.T) {
	store := NewInMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "test",
		ParentID:  "doc1",
		Position:  0,
		Embedding: []float64{1.0, 0.0, 0.0},
	}

	err := store.Upsert(ctx, chunk)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestInMemoryStore_ContextCancellation_Delete(t *testing.T) {
	store := NewInMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.Delete(ctx, "c1")
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestInMemoryStore_ContextCancellation_DeleteByParentID(t *testing.T) {
	store := NewInMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.DeleteByParentID(ctx, "doc1")
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestInMemoryStore_ContextCancellation_SearchWithFilter(t *testing.T) {
	store := NewInMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	filter := domain.ParentIDFilter{ParentIDs: []string{"doc1"}}
	_, err := store.SearchWithFilter(ctx, queryEmbedding, 5, filter)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestInMemoryStore_ContextCancellation_SearchWithMetadataFilter(t *testing.T) {
	store := NewInMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	_, err := store.SearchWithMetadataFilter(ctx, queryEmbedding, 5, filter)
	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
}

func TestInMemoryStore_NilContext_Panic(t *testing.T) {
	store := NewInMemoryStore()

	// Проверяем, что nil context вызывает panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context, got nil")
		}
	}()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	//nolint:staticcheck // Нам нужно передать nil context, чтобы проверить, что метод паникует.
	_, _ = store.Search(nil, queryEmbedding, 5)
}

func TestCosineSimilarity_ZeroVectors(t *testing.T) {
	a := []float64{0.0, 0.0, 0.0}
	b := []float64{0.0, 0.0, 0.0}

	result := cosineSimilarity(a, b)
	if result != 0 {
		t.Errorf("expected 0 for zero vectors, got %f", result)
	}
}

func TestCosineSimilarity_SingleDimension(t *testing.T) {
	a := []float64{1.0}
	b := []float64{1.0}

	result := cosineSimilarity(a, b)
	if result < 0.99 || result > 1.01 {
		t.Errorf("expected ~1.0 for identical single-dimension vectors, got %f", result)
	}
}

func TestCosineSimilarity_NegativeValues(t *testing.T) {
	a := []float64{-1.0, 0.0}
	b := []float64{-1.0, 0.0}

	result := cosineSimilarity(a, b)
	if result < 0.99 || result > 1.01 {
		t.Errorf("expected ~1.0 for identical negative vectors, got %f", result)
	}
}
