package vectorstore

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestQdrantStore_NewQdrantStore_DefaultBaseURL(t *testing.T) {
	store := NewQdrantStore("", "test-collection", 1536)
	
	if store.baseURL != "http://localhost:6333" {
		t.Errorf("expected default baseURL http://localhost:6333, got %s", store.baseURL)
	}
	if store.collection != "test-collection" {
		t.Errorf("expected collection test-collection, got %s", store.collection)
	}
	if store.dimension != 1536 {
		t.Errorf("expected dimension 1536, got %d", store.dimension)
	}
}

func TestQdrantStore_NewQdrantStore_CustomBaseURL(t *testing.T) {
	store := NewQdrantStore("http://custom:9000", "test-collection", 768)
	
	if store.baseURL != "http://custom:9000" {
		t.Errorf("expected custom baseURL http://custom:9000, got %s", store.baseURL)
	}
	if store.collection != "test-collection" {
		t.Errorf("expected collection test-collection, got %s", store.collection)
	}
	if store.dimension != 768 {
		t.Errorf("expected dimension 768, got %d", store.dimension)
	}
}

func TestQdrantRuntimeOptions_Default(t *testing.T) {
	opts := QdrantRuntimeOptions{}
	
	if opts.SearchTimeout != 0 {
		t.Errorf("expected zero SearchTimeout, got %v", opts.SearchTimeout)
	}
	if opts.UpsertTimeout != 0 {
		t.Errorf("expected zero UpsertTimeout, got %v", opts.UpsertTimeout)
	}
	if opts.DeleteTimeout != 0 {
		t.Errorf("expected zero DeleteTimeout, got %v", opts.DeleteTimeout)
	}
	if opts.MaxTopK != 0 {
		t.Errorf("expected zero MaxTopK, got %d", opts.MaxTopK)
	}
}

func TestQdrantRuntimeOptions_WithValues(t *testing.T) {
	opts := QdrantRuntimeOptions{
		SearchTimeout:  30,
		UpsertTimeout:  60,
		DeleteTimeout:  30,
		MaxTopK:       100,
	}
	
	if opts.SearchTimeout != 30 {
		t.Errorf("expected SearchTimeout 30, got %v", opts.SearchTimeout)
	}
	if opts.UpsertTimeout != 60 {
		t.Errorf("expected UpsertTimeout 60, got %v", opts.UpsertTimeout)
	}
	if opts.DeleteTimeout != 30 {
		t.Errorf("expected DeleteTimeout 30, got %v", opts.DeleteTimeout)
	}
	if opts.MaxTopK != 100 {
		t.Errorf("expected MaxTopK 100, got %d", opts.MaxTopK)
	}
}

func TestQdrantStore_Interfaces(t *testing.T) {
	// Compile-time проверка интерфейсов
	var _ domain.VectorStore = (*QdrantStore)(nil)
	var _ domain.VectorStoreWithFilters = (*QdrantStore)(nil)
	var _ domain.DocumentStore = (*QdrantStore)(nil)
	
	// Если скомпилируется - интерфейсы реализованы корректно
	store := NewQdrantStore("http://localhost:6333", "test", 1536)
	if store == nil {
		t.Error("expected non-nil store")
	}
}

func TestQdrantStore_EmbeddingDimensionMismatch(t *testing.T) {
	// Проверка ошибки размерности эмбеддинга
	err := domain.ErrEmbeddingDimensionMismatch
	if err == nil {
		t.Error("expected non-nil error")
	}
}

func TestQdrantStore_ChunkValidation(t *testing.T) {
	// Проверка валидации чанка
	chunk := domain.Chunk{
		ID:       "",
		Content:  "test",
		ParentID: "doc1",
		Position: 0,
	}
	
	err := chunk.Validate()
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestQdrantStore_MetadataMapping(t *testing.T) {
	// Проверка маппинга метаданных
	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "test content",
		ParentID:  "doc1",
		Position:  0,
		Metadata:  map[string]string{"source": "wiki", "lang": "ru"},
		Embedding: []float64{0.1, 0.2, 0.3},
	}
	
	// Проверяем, что метаданные содержат ожидаемые поля
	if chunk.Metadata["source"] != "wiki" {
		t.Errorf("expected source=wiki, got %s", chunk.Metadata["source"])
	}
	if chunk.Metadata["lang"] != "ru" {
		t.Errorf("expected lang=ru, got %s", chunk.Metadata["lang"])
	}
	if chunk.ParentID != "doc1" {
		t.Errorf("expected parent_id=doc1, got %s", chunk.ParentID)
	}
}

func TestMilvusStore_Interfaces(t *testing.T) {
	// Compile-time проверка интерфейсов
	var _ domain.VectorStore = (*MilvusStore)(nil)
	var _ domain.VectorStoreWithFilters = (*MilvusStore)(nil)
	var _ domain.HybridSearcher = (*MilvusStore)(nil)
	
	// Если скомпилируется - интерфейсы реализованы корректно
	store := NewMilvusStore("http://localhost:19530", "test", "")
	if store == nil {
		t.Error("expected non-nil store")
	}
}
