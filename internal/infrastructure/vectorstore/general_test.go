package vectorstore

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestInMemoryStore_DeleteNonExistent(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Удаление несуществующего чанка не должно вызывать ошибку
	err := store.Delete(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("expected no error for deleting non-existent chunk, got %v", err)
	}
}

func TestInMemoryStore_DeleteByParentID_Empty(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Удаление по несуществующему parentID не должно вызывать ошибку
	err := store.DeleteByParentID(ctx, "non-existent-parent-id")
	if err != nil {
		t.Fatalf("expected no error for deleting non-existent parentID, got %v", err)
	}
}

func TestInMemoryStore_Search_InvalidTopK(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	_, err := store.Search(ctx, queryEmbedding, 0)
	if err == nil {
		t.Fatal("expected error for topK=0, got nil")
	}
}

func TestInMemoryStore_SearchWithFilter_InvalidTopK(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	filter := domain.ParentIDFilter{ParentIDs: []string{"doc1"}}
	_, err := store.SearchWithFilter(ctx, queryEmbedding, 0, filter)
	if err == nil {
		t.Fatal("expected error for topK=0, got nil")
	}
}

func TestInMemoryStore_SearchWithMetadataFilter_InvalidTopK(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	filter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	_, err := store.SearchWithMetadataFilter(ctx, queryEmbedding, 0, filter)
	if err == nil {
		t.Fatal("expected error for topK=0, got nil")
	}
}

func TestInMemoryStore_Search_TopK_Limit(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Добавляем 5 чанков
	ids := []string{"c0", "c1", "c2", "c3", "c4"}
	for i, id := range ids {
		chunk := domain.Chunk{
			ID:        id,
			Content:   "test",
			ParentID:  "doc1",
			Position:  i,
			Embedding: []float64{float64(i) * 0.2, 0.0, 0.0},
		}
		err := store.Upsert(ctx, chunk)
		if err != nil {
			t.Fatalf("upsert failed: %v", err)
		}
	}

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	result, err := store.Search(ctx, queryEmbedding, 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Chunks) != 3 {
		t.Errorf("expected 3 chunks (topK=3), got %d", len(result.Chunks))
	}
	if result.TotalFound != 5 {
		t.Errorf("expected TotalFound=5, got %d", result.TotalFound)
	}
}

func TestInMemoryStore_Upsert_DimensionMismatch(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Чанк с неправильной размерностью эмбеддинга не вызовет ошибку в InMemoryStore
	// так как он не проверяет размерность при upsert
	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "test",
		ParentID:  "doc1",
		Position:  0,
		Embedding: []float64{1.0, 0.0}, // 2D вместо ожидаемого
	}

	err := store.Upsert(ctx, chunk)
	// InMemoryStore не проверяет размерность, поэтому ошибки не будет
	if err != nil {
		t.Fatalf("unexpected error for dimension mismatch in InMemoryStore: %v", err)
	}
}

func TestHybridConfig_DefaultValues(t *testing.T) {
	config := domain.DefaultHybridConfig()

	// Проверяем дефолтные значения
	if config.UseRRF != true {
		t.Errorf("expected UseRRF=true, got %v", config.UseRRF)
	}
	if config.RRFK != 60 {
		t.Errorf("expected RRFK=60, got %d", config.RRFK)
	}
	if config.SemanticWeight != 0.7 {
		t.Errorf("expected SemanticWeight=0.7, got %f", config.SemanticWeight)
	}
	if config.BMFinalK != 0 {
		t.Errorf("expected BMFinalK=0, got %d", config.BMFinalK)
	}
}

func TestParentIDFilter_EmptySlice(t *testing.T) {
	filter := domain.ParentIDFilter{
		ParentIDs: []string{},
	}

	if len(filter.ParentIDs) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(filter.ParentIDs))
	}
}

func TestMetadataFilter_NilFields(t *testing.T) {
	filter := domain.MetadataFilter{
		Fields: nil,
	}

	if len(filter.Fields) != 0 {
		t.Errorf("expected nil fields to be treated as empty, got %d", len(filter.Fields))
	}
}

func TestRetrievedChunk_Empty(t *testing.T) {
	chunk := domain.RetrievedChunk{
		Chunk: domain.Chunk{},
		Score: 0.0,
	}

	if chunk.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", chunk.Score)
	}
}

func TestChunk_EmptyEmbedding(t *testing.T) {
	chunk := domain.Chunk{
		ID:       "c1",
		Content:  "test",
		ParentID: "doc1",
		Position: 0,
		Embedding: []float64{},
	}

	// Пустой embedding допустим в некоторых случаях
	if len(chunk.Embedding) != 0 {
		t.Errorf("expected empty embedding, got %d elements", len(chunk.Embedding))
	}
}

func TestDocument_Empty(t *testing.T) {
	doc := domain.Document{
		ID:      "",
		Content: "",
	}

	err := doc.Validate()
	if err == nil {
		t.Fatal("expected error for empty document, got nil")
	}
}
