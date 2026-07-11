package vectorstore

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task T4.1: Unit-тесты InMemoryStore.SearchWithMetadataFilter (AC-002, AC-004, AC-005)

// TestInMemoryStore_SearchWithMetadataFilter_Filters проверяет, что фильтр по метаданным
// возвращает только совпадающие чанки (AC-001, AC-005).
func TestInMemoryStore_SearchWithMetadataFilter_Filters(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	legal := domain.Chunk{
		ID:        "doc-legal#0",
		Content:   "legal document",
		ParentID:  "doc-legal",
		Embedding: []float64{1, 0},
		Metadata:  map[string]string{"category": "legal"},
	}
	finance := domain.Chunk{
		ID:        "doc-finance#0",
		Content:   "finance document",
		ParentID:  "doc-finance",
		Embedding: []float64{1, 0},
		Metadata:  map[string]string{"category": "finance"},
	}

	for _, c := range []domain.Chunk{legal, finance} {
		if err := store.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	result, err := store.SearchWithMetadataFilter(ctx, []float64{1, 0}, 10, domain.MetadataFilter{
		Fields: map[string]string{"category": "legal"},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
	}
	if result.Chunks[0].Chunk.ID != "doc-legal#0" {
		t.Fatalf("expected doc-legal#0, got %s", result.Chunks[0].Chunk.ID)
	}
}

// TestInMemoryStore_SearchWithMetadataFilter_EmptyFilter проверяет, что пустой фильтр
// возвращает тот же результат, что и Search без фильтра (AC-002).
func TestInMemoryStore_SearchWithMetadataFilter_EmptyFilter(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunk := domain.Chunk{
		ID:        "doc-1#0",
		Content:   "hello",
		ParentID:  "doc-1",
		Embedding: []float64{1, 0},
		Metadata:  map[string]string{"category": "legal"},
	}
	if err := store.Upsert(ctx, chunk); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	base, err := store.Search(ctx, []float64{1, 0}, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	filtered, err := store.SearchWithMetadataFilter(ctx, []float64{1, 0}, 10, domain.MetadataFilter{})
	if err != nil {
		t.Fatalf("filter search: %v", err)
	}
	if len(base.Chunks) != len(filtered.Chunks) {
		t.Fatalf("expected same count: base=%d filtered=%d", len(base.Chunks), len(filtered.Chunks))
	}
}

// TestInMemoryStore_SearchWithMetadataFilter_NoMatch проверяет, что несуществующий фильтр
// возвращает пустой результат без ошибки (AC-004).
func TestInMemoryStore_SearchWithMetadataFilter_NoMatch(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunk := domain.Chunk{
		ID:        "doc-1#0",
		Content:   "hello",
		ParentID:  "doc-1",
		Embedding: []float64{1, 0},
		Metadata:  map[string]string{"category": "legal"},
	}
	if err := store.Upsert(ctx, chunk); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	result, err := store.SearchWithMetadataFilter(ctx, []float64{1, 0}, 10, domain.MetadataFilter{
		Fields: map[string]string{"category": "nonexistent"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(result.Chunks) != 0 {
		t.Fatalf("expected 0 chunks, got %d", len(result.Chunks))
	}
}

// TestInMemoryStore_ImplementsVectorStoreWithFilters проверяет, что InMemoryStore
// реализует интерфейс VectorStoreWithFilters (AC-005).
func TestInMemoryStore_ImplementsVectorStoreWithFilters(t *testing.T) {
	t.Parallel()
	var _ domain.VectorStoreWithFilters = (*InMemoryStore)(nil)
}

func TestInMemoryStore_BasicSearch(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	chunk := domain.Chunk{
		ID:        "doc-1#0",
		Content:   "cat",
		ParentID:  "doc-1",
		Embedding: []float64{1, 0, 0},
		Position:  0,
	}

	if err := store.Upsert(ctx, chunk); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	result, err := store.Search(ctx, []float64{1, 0, 0}, 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Chunks) == 0 {
		t.Fatalf("expected non-empty results")
	}
	if result.Chunks[0].Score <= 0 {
		t.Fatalf("expected score > 0, got %v", result.Chunks[0].Score)
	}
	if result.Chunks[0].Score > 1 || result.Chunks[0].Score < -1 {
		t.Fatalf("expected score in [-1, 1], got %v", result.Chunks[0].Score)
	}
}

// @sk-test hierarchical-indices#T5.2: TestInMemoryStoreParentDocumentStore (AC-001, DEC-001, DEC-002, DM-002)
func TestInMemoryStoreParentDocumentStore(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	doc := domain.Document{
		ID:      "doc-1",
		Content: "parent document content for testing",
	}
	embedding := []float64{0.1, 0.2, 0.3}

	var vs domain.VectorStore = store
	ps, ok := vs.(domain.ParentDocumentStore)
	if !ok {
		t.Fatal("InMemoryStore does not implement ParentDocumentStore")
	}

	if err := ps.UpsertParent(ctx, doc, embedding); err != nil {
		t.Fatalf("UpsertParent: %v", err)
	}

	got, err := ps.GetParentDocument(ctx, "doc-1")
	if err != nil {
		t.Fatalf("GetParentDocument: %v", err)
	}
	if got == nil {
		t.Fatal("expected parent document, got nil")
	}
	if got.Content != "parent document content for testing" {
		t.Fatalf("expected content %q, got %q", "parent document content for testing", got.Content)
	}

	if err := ps.DeleteParent(ctx, "doc-1"); err != nil {
		t.Fatalf("DeleteParent: %v", err)
	}

	got, err = ps.GetParentDocument(ctx, "doc-1")
	if err != nil {
		t.Fatalf("GetParentDocument after delete: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

// @sk-test hierarchical-indices#T5.2: TestInMemoryStoreParentDocumentStoreNotFound (DEC-002)
func TestInMemoryStoreParentDocumentStoreNotFound(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	var vs domain.VectorStore = store
	ps, ok := vs.(domain.ParentDocumentStore)
	if !ok {
		t.Fatal("InMemoryStore does not implement ParentDocumentStore")
	}

	got, err := ps.GetParentDocument(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetParentDocument: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nonexistent document")
	}
}

// @sk-test hierarchical-indices#T5.2: TestInMemoryStoreParentDocumentStoreDeleteIdempotent (DM-002)
func TestInMemoryStoreParentDocumentStoreDeleteIdempotent(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	var vs domain.VectorStore = store
	ps, ok := vs.(domain.ParentDocumentStore)
	if !ok {
		t.Fatal("InMemoryStore does not implement ParentDocumentStore")
	}

	if err := ps.DeleteParent(ctx, "nonexistent"); err != nil {
		t.Fatalf("DeleteParent on nonexistent should be idempotent: %v", err)
	}
}
