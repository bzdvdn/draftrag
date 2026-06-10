package vectorstore

import (
	"context"
	"fmt"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// StoreFactory создаёт чистое (пустое) хранилище для contract-теста.
//
// @sk-task contract-tests-stores#T1.1,contract-tests-stores#T2.1: StoreFactory type + 8 VectorStore scenarios
type StoreFactory func() domain.VectorStore

// runVectorStoreContract запускает все contract-сценарии VectorStore.
//
// @sk-test contract-tests-stores#T2.1: upsert_and_search, upsert_and_search_topk_clipping, search_empty_collection,
// delete_and_search, delete_nonexistent_id, upsert_duplicate_id, nil_embedding_skipped, topk_zero_or_negative
func runVectorStoreContract(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("upsert_and_search", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		chunk := domain.Chunk{
			ID: "doc-1#0", Content: "cat", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
		}
		if err := s.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		result, err := s.Search(ctx, []float64{1, 0, 0}, 5)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) == 0 {
			t.Fatal("expected non-empty results")
		}
		if result.Chunks[0].Chunk.ID != chunk.ID {
			t.Fatalf("expected id %s, got %s", chunk.ID, result.Chunks[0].Chunk.ID)
		}
		if result.Chunks[0].Score <= 0 {
			t.Fatalf("expected positive score, got %v", result.Chunks[0].Score)
		}
	})
	t.Run("upsert_and_search_topk_clipping", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			chunk := domain.Chunk{
				ID: fmt.Sprintf("doc-1#%d", i), Content: "text", ParentID: "doc-1",
				Embedding: []float64{1, 0, 0},
			}
			if err := s.Upsert(ctx, chunk); err != nil {
				t.Fatalf("upsert %d: %v", i, err)
			}
		}

		result, err := s.Search(ctx, []float64{1, 0, 0}, 1)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
		}
		// TotalFound может быть 3 (InMemoryStore: до clipping) или 1 (другие store),
		// поэтому проверяем только корректность Chunks
	})
	t.Run("search_empty_collection", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		result, err := s.Search(ctx, []float64{1, 0, 0}, 5)
		if err != nil {
			t.Fatalf("search on empty store: %v", err)
		}
		if len(result.Chunks) != 0 {
			t.Fatalf("expected empty result, got %d chunks", len(result.Chunks))
		}
	})
	t.Run("delete_and_search", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		chunk := domain.Chunk{
			ID: "doc-1#0", Content: "cat", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
		}
		if err := s.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		result, err := s.Search(ctx, []float64{1, 0, 0}, 5)
		if err != nil {
			t.Fatalf("search before delete: %v", err)
		}
		if len(result.Chunks) == 0 {
			t.Fatal("expected result before delete")
		}

		if err := s.Delete(ctx, chunk.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}

		result, err = s.Search(ctx, []float64{1, 0, 0}, 5)
		if err != nil {
			t.Fatalf("search after delete: %v", err)
		}
		if len(result.Chunks) != 0 {
			t.Fatalf("expected empty after delete, got %d", len(result.Chunks))
		}
	})
	t.Run("delete_nonexistent_id", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		err := s.Delete(ctx, "nonexistent-id")
		if err != nil {
			t.Fatalf("delete nonexistent id should be idempotent, got: %v", err)
		}
	})
	t.Run("upsert_duplicate_id", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		first := domain.Chunk{
			ID: "doc-1#0", Content: "original", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
		}
		if err := s.Upsert(ctx, first); err != nil {
			t.Fatalf("first upsert: %v", err)
		}

		second := domain.Chunk{
			ID: "doc-1#0", Content: "overwritten", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
		}
		if err := s.Upsert(ctx, second); err != nil {
			t.Fatalf("second upsert: %v", err)
		}

		result, err := s.Search(ctx, []float64{1, 0, 0}, 5)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
		}
		if result.Chunks[0].Chunk.Content != "overwritten" {
			t.Fatalf("expected overwritten content, got %s", result.Chunks[0].Chunk.Content)
		}
	})
	t.Run("nil_embedding_skipped", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		withEmbedding := domain.Chunk{
			ID: "doc-1#0", Content: "visible", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
		}
		if err := s.Upsert(ctx, withEmbedding); err != nil {
			t.Fatalf("upsert with embedding: %v", err)
		}

		withoutEmbedding := domain.Chunk{
			ID: "doc-1#1", Content: "hidden", ParentID: "doc-1",
			Embedding: nil,
		}
		err := s.Upsert(ctx, withoutEmbedding)
		if err != nil {
			// Некоторые store (с размерностью) возвращают ошибку dimension mismatch — это допустимо
			return
		}

		result, err := s.Search(ctx, []float64{1, 0, 0}, 5)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 1 {
			t.Fatalf("expected 1 chunk (nil embedding skipped), got %d", len(result.Chunks))
		}
		if result.Chunks[0].Chunk.ID != "doc-1#0" {
			t.Fatalf("expected doc-1#0, got %s", result.Chunks[0].Chunk.ID)
		}
	})
	t.Run("topk_zero_or_negative", func(t *testing.T) {
		s := newStore()
		ctx := context.Background()

		_, err := s.Search(ctx, []float64{1, 0, 0}, 0)
		if err == nil {
			t.Fatal("expected error for topK=0")
		}

		_, err = s.Search(ctx, []float64{1, 0, 0}, -1)
		if err == nil {
			t.Fatal("expected error for topK=-1")
		}
	})
}

// storeWithFilters возвращает VectorStoreWithFilters или пропускает тест.
//
// @sk-test contract-tests-stores#T3.1: search_with_filter_single_parent, search_with_filter_multi_parent,
// search_with_filter_empty_delegates, search_with_metadata_filter_exact, search_with_metadata_filter_multi_field,
// search_with_metadata_filter_no_match, search_with_metadata_filter_empty_delegates
func storeWithFilters(t *testing.T, newStore StoreFactory) domain.VectorStoreWithFilters {
	t.Helper()
	s, ok := newStore().(domain.VectorStoreWithFilters)
	if !ok {
		t.Skip("store does not implement VectorStoreWithFilters")
	}
	return s
}

// runFilterContract запускает все contract-сценарии VectorStoreWithFilters.
func runFilterContract(t *testing.T, newStore StoreFactory) {
	t.Helper()

	t.Run("search_with_filter_single_parent", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunks := []domain.Chunk{
			{ID: "doc-a#0", Content: "a1", ParentID: "doc-a", Embedding: []float64{1, 0, 0}},
			{ID: "doc-b#0", Content: "b1", ParentID: "doc-b", Embedding: []float64{1, 0, 0}},
		}
		for _, c := range chunks {
			if err := s.Upsert(ctx, c); err != nil {
				t.Fatalf("upsert: %v", err)
			}
		}

		result, err := s.SearchWithFilter(ctx, []float64{1, 0, 0}, 10, domain.ParentIDFilter{
			ParentIDs: []string{"doc-a"},
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
		}
		if result.Chunks[0].Chunk.ParentID != "doc-a" {
			t.Fatalf("expected parent doc-a, got %s", result.Chunks[0].Chunk.ParentID)
		}
	})
	t.Run("search_with_filter_multi_parent", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunks := []domain.Chunk{
			{ID: "doc-a#0", Content: "a1", ParentID: "doc-a", Embedding: []float64{1, 0, 0}},
			{ID: "doc-b#0", Content: "b1", ParentID: "doc-b", Embedding: []float64{1, 0, 0}},
			{ID: "doc-c#0", Content: "c1", ParentID: "doc-c", Embedding: []float64{1, 0, 0}},
		}
		for _, c := range chunks {
			if err := s.Upsert(ctx, c); err != nil {
				t.Fatalf("upsert: %v", err)
			}
		}

		result, err := s.SearchWithFilter(ctx, []float64{1, 0, 0}, 10, domain.ParentIDFilter{
			ParentIDs: []string{"doc-a", "doc-c"},
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(result.Chunks))
		}
	})
	t.Run("search_with_filter_empty_delegates", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunk := domain.Chunk{
			ID: "doc-1#0", Content: "hello", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
		}
		if err := s.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		base, err := s.Search(ctx, []float64{1, 0, 0}, 10)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		filtered, err := s.SearchWithFilter(ctx, []float64{1, 0, 0}, 10, domain.ParentIDFilter{})
		if err != nil {
			t.Fatalf("filter search: %v", err)
		}
		if len(base.Chunks) != len(filtered.Chunks) {
			t.Fatalf("expected same count: base=%d filtered=%d", len(base.Chunks), len(filtered.Chunks))
		}
	})
	t.Run("search_with_metadata_filter_exact", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunks := []domain.Chunk{
			{ID: "doc-1#0", Content: "legal", ParentID: "doc-1", Embedding: []float64{1, 0, 0}, Metadata: map[string]string{"category": "legal"}},
			{ID: "doc-2#0", Content: "finance", ParentID: "doc-2", Embedding: []float64{1, 0, 0}, Metadata: map[string]string{"category": "finance"}},
		}
		for _, c := range chunks {
			if err := s.Upsert(ctx, c); err != nil {
				t.Fatalf("upsert: %v", err)
			}
		}

		result, err := s.SearchWithMetadataFilter(ctx, []float64{1, 0, 0}, 10, domain.MetadataFilter{
			Fields: map[string]string{"category": "legal"},
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
		}
		if result.Chunks[0].Chunk.ID != "doc-1#0" {
			t.Fatalf("expected doc-1#0, got %s", result.Chunks[0].Chunk.ID)
		}
	})
	t.Run("search_with_metadata_filter_multi_field", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunks := []domain.Chunk{
			{ID: "doc-1#0", Content: "match", ParentID: "doc-1", Embedding: []float64{1, 0, 0}, Metadata: map[string]string{"category": "legal", "lang": "en"}},
			{ID: "doc-2#0", Content: "no-match", ParentID: "doc-2", Embedding: []float64{1, 0, 0}, Metadata: map[string]string{"category": "legal", "lang": "fr"}},
		}
		for _, c := range chunks {
			if err := s.Upsert(ctx, c); err != nil {
				t.Fatalf("upsert: %v", err)
			}
		}

		result, err := s.SearchWithMetadataFilter(ctx, []float64{1, 0, 0}, 10, domain.MetadataFilter{
			Fields: map[string]string{"category": "legal", "lang": "en"},
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(result.Chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(result.Chunks))
		}
		if result.Chunks[0].Chunk.ID != "doc-1#0" {
			t.Fatalf("expected doc-1#0, got %s", result.Chunks[0].Chunk.ID)
		}
	})
	t.Run("search_with_metadata_filter_no_match", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunk := domain.Chunk{
			ID: "doc-1#0", Content: "hello", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
			Metadata:  map[string]string{"category": "legal"},
		}
		if err := s.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		result, err := s.SearchWithMetadataFilter(ctx, []float64{1, 0, 0}, 10, domain.MetadataFilter{
			Fields: map[string]string{"category": "nonexistent"},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(result.Chunks) != 0 {
			t.Fatalf("expected 0 chunks, got %d", len(result.Chunks))
		}
	})
	t.Run("search_with_metadata_filter_empty_delegates", func(t *testing.T) {
		s := storeWithFilters(t, newStore)
		ctx := context.Background()

		chunk := domain.Chunk{
			ID: "doc-1#0", Content: "hello", ParentID: "doc-1",
			Embedding: []float64{1, 0, 0},
			Metadata:  map[string]string{"category": "legal"},
		}
		if err := s.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		base, err := s.Search(ctx, []float64{1, 0, 0}, 10)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		filtered, err := s.SearchWithMetadataFilter(ctx, []float64{1, 0, 0}, 10, domain.MetadataFilter{})
		if err != nil {
			t.Fatalf("filter search: %v", err)
		}
		if len(base.Chunks) != len(filtered.Chunks) {
			t.Fatalf("expected same count: base=%d filtered=%d", len(base.Chunks), len(filtered.Chunks))
		}
	})
}
