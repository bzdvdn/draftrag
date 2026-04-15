package vectorstore

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestInMemoryStore_Integration_FullWorkflow(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Добавляем чанки
	chunks := []domain.Chunk{
		{
			ID:        "c1",
			Content:   "first chunk",
			ParentID:  "doc1",
			Position:  0,
			Embedding: []float64{1.0, 0.0, 0.0},
			Metadata:  map[string]string{"source": "wiki"},
		},
		{
			ID:        "c2",
			Content:   "second chunk",
			ParentID:  "doc1",
			Position:  1,
			Embedding: []float64{0.0, 1.0, 0.0},
			Metadata:  map[string]string{"source": "docs"},
		},
		{
			ID:        "c3",
			Content:   "third chunk",
			ParentID:  "doc2",
			Position:  0,
			Embedding: []float64{0.0, 0.0, 1.0},
			Metadata:  map[string]string{"source": "wiki"},
		},
	}

	for _, chunk := range chunks {
		err := store.Upsert(ctx, chunk)
		if err != nil {
			t.Fatalf("failed to upsert chunk %s: %v", chunk.ID, err)
		}
	}

	// Тестируем Search
	queryEmbedding := []float64{1.0, 0.0, 0.0}
	result, err := store.Search(ctx, queryEmbedding, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(result.Chunks))
	}
	if result.TotalFound != 3 {
		t.Errorf("expected TotalFound=3, got %d", result.TotalFound)
	}

	// Тестируем SearchWithFilter
	filter := domain.ParentIDFilter{ParentIDs: []string{"doc1"}}
	filteredResult, err := store.SearchWithFilter(ctx, queryEmbedding, 2, filter)
	if err != nil {
		t.Fatalf("SearchWithFilter failed: %v", err)
	}
	// doc1 имеет c1 (embedding [1,0,0]) и c2 (embedding [0,1,0])
	// queryEmbedding [1,0,0] ближе к c1, но c2 тоже имеет parentID doc1
	if len(filteredResult.Chunks) != 2 {
		t.Errorf("expected 2 chunks with parentID doc1, got %d", len(filteredResult.Chunks))
	}

	// Тестируем SearchWithMetadataFilter
	metadataFilter := domain.MetadataFilter{Fields: map[string]string{"source": "wiki"}}
	metadataResult, err := store.SearchWithMetadataFilter(ctx, queryEmbedding, 2, metadataFilter)
	if err != nil {
		t.Fatalf("SearchWithMetadataFilter failed: %v", err)
	}
	// c1 и c3 имеют source=wiki, queryEmbedding [1,0,0] ближе к c1
	if len(metadataResult.Chunks) != 2 {
		t.Errorf("expected 2 chunks with source=wiki, got %d", len(metadataResult.Chunks))
	}

	// Тестируем Delete
	err = store.Delete(ctx, "c1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Проверяем, что чанк удалён
	_, err = store.Search(ctx, queryEmbedding, 5)
	if err != nil {
		t.Fatalf("Search after delete failed: %v", err)
	}

	// Тестируем DeleteByParentID
	err = store.DeleteByParentID(ctx, "doc1")
	if err != nil {
		t.Fatalf("DeleteByParentID failed: %v", err)
	}
}

func TestInMemoryStore_Integration_EmptyStore(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	queryEmbedding := []float64{1.0, 0.0, 0.0}
	result, err := store.Search(ctx, queryEmbedding, 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Chunks) != 0 {
		t.Errorf("expected 0 chunks from empty store, got %d", len(result.Chunks))
	}
	if result.TotalFound != 0 {
		t.Errorf("expected TotalFound=0, got %d", result.TotalFound)
	}
}

func TestInMemoryStore_Integration_UpdateChunk(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "original content",
		ParentID:  "doc1",
		Position:  0,
		Embedding: []float64{1.0, 0.0, 0.0},
	}

	err := store.Upsert(ctx, chunk)
	if err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}

	// Обновляем чанк
	chunk.Content = "updated content"
	chunk.Embedding = []float64{0.5, 0.5, 0.0}

	err = store.Upsert(ctx, chunk)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	// Проверяем, что чанк обновлён
	queryEmbedding := []float64{0.5, 0.5, 0.0}
	result, err := store.Search(ctx, queryEmbedding, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result.Chunks))
	}
	if result.Chunks[0].Chunk.Content != "updated content" {
		t.Errorf("expected updated content, got %s", result.Chunks[0].Chunk.Content)
	}
}

func TestInMemoryStore_Integration_SearchWithFilter_Delegation(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "test content",
		ParentID:  "doc1",
		Position:  0,
		Embedding: []float64{1.0, 0.0, 0.0},
	}

	err := store.Upsert(ctx, chunk)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Пустой filter должен делегироваться в базовый Search
	queryEmbedding := []float64{1.0, 0.0, 0.0}
	filter := domain.ParentIDFilter{ParentIDs: []string{}}
	result, err := store.SearchWithFilter(ctx, queryEmbedding, 5, filter)
	if err != nil {
		t.Fatalf("SearchWithFilter with empty ParentIDs failed: %v", err)
	}
	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk with empty filter, got %d", len(result.Chunks))
	}
}

func TestInMemoryStore_Integration_SearchWithMetadataFilter_Delegation(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "test content",
		ParentID:  "doc1",
		Position:  0,
		Embedding: []float64{1.0, 0.0, 0.0},
		Metadata:  map[string]string{"source": "wiki"},
	}

	err := store.Upsert(ctx, chunk)
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Пустой filter должен делегироваться в базовый Search
	queryEmbedding := []float64{1.0, 0.0, 0.0}
	filter := domain.MetadataFilter{Fields: map[string]string{}}
	result, err := store.SearchWithMetadataFilter(ctx, queryEmbedding, 5, filter)
	if err != nil {
		t.Fatalf("SearchWithMetadataFilter with empty Fields failed: %v", err)
	}
	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk with empty filter, got %d", len(result.Chunks))
	}
}

func TestInMemoryStore_Integration_Sorting(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunks := []domain.Chunk{
		{
			ID:        "c1",
			Content:   "low score",
			ParentID:  "doc1",
			Position:  0,
			Embedding: []float64{0.1, 0.9, 0.0}, // ортогонален к query
		},
		{
			ID:        "c2",
			Content:   "high score",
			ParentID:  "doc1",
			Position:  1,
			Embedding: []float64{1.0, 0.0, 0.0}, // идентичен query
		},
		{
			ID:        "c3",
			Content:   "medium score",
			ParentID:  "doc1",
			Position:  2,
			Embedding: []float64{0.5, 0.5, 0.0}, // 45 градусов к query
		},
	}

	for _, chunk := range chunks {
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

	// Проверяем, что результаты отсортированы по убыванию score
	if len(result.Chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result.Chunks))
	}
	if result.Chunks[0].Chunk.ID != "c2" {
		t.Errorf("expected c2 first (highest similarity), got %s", result.Chunks[0].Chunk.ID)
	}
	if result.Chunks[1].Chunk.ID != "c3" {
		t.Errorf("expected c3 second (medium similarity), got %s", result.Chunks[1].Chunk.ID)
	}
	if result.Chunks[2].Chunk.ID != "c1" {
		t.Errorf("expected c1 third (lowest similarity), got %s", result.Chunks[2].Chunk.ID)
	}
}
