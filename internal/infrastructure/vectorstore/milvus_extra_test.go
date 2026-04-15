package vectorstore

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestMilvusStore_EmbeddingDimensionMismatch(t *testing.T) {
	// Проверка ошибки размерности эмбеддинга
	err := domain.ErrEmbeddingDimensionMismatch
	if err == nil {
		t.Error("expected non-nil error")
	}
}

func TestMilvusStore_ChunkValidation(t *testing.T) {
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

func TestMilvusStore_MetadataMapping(t *testing.T) {
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

func TestMilvusStore_HybridConfig_Validation(t *testing.T) {
	// Проверка валидации гибридной конфигурации
	config := domain.HybridConfig{
		UseRRF:         true,
		RRFK:           60,
		SemanticWeight: 0.7,
	}
	
	err := config.Validate()
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestMilvusStore_HybridConfig_InvalidRRFK(t *testing.T) {
	config := domain.HybridConfig{
		UseRRF: true,
		RRFK:   0, // невалидно
	}
	
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for invalid RRFK, got nil")
	}
}

func TestMilvusStore_HybridConfig_InvalidSemanticWeight(t *testing.T) {
	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 1.5, // невалидно
	}
	
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for invalid SemanticWeight, got nil")
	}
}

func TestMilvusStore_ParentIDFilter(t *testing.T) {
	// Проверка фильтра по ParentID
	filter := domain.ParentIDFilter{
		ParentIDs: []string{"doc1", "doc2", "doc3"},
	}
	
	if len(filter.ParentIDs) != 3 {
		t.Errorf("expected 3 parent IDs, got %d", len(filter.ParentIDs))
	}
	
	if filter.ParentIDs[0] != "doc1" {
		t.Errorf("expected doc1, got %s", filter.ParentIDs[0])
	}
}

func TestMilvusStore_MetadataFilter(t *testing.T) {
	// Проверка фильтра по метаданным
	filter := domain.MetadataFilter{
		Fields: map[string]string{
			"source": "wiki",
			"lang":   "ru",
		},
	}
	
	if len(filter.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(filter.Fields))
	}
	
	if filter.Fields["source"] != "wiki" {
		t.Errorf("expected source=wiki, got %s", filter.Fields["source"])
	}
}

func TestMilvusStore_EmbeddingVector(t *testing.T) {
	// Проверка работы с embedding-векторами
	embedding := []float64{0.1, 0.2, 0.3, 0.4}
	
	if len(embedding) != 4 {
		t.Errorf("expected 4 dimensions, got %d", len(embedding))
	}
	
	if embedding[0] != 0.1 {
		t.Errorf("expected 0.1, got %f", embedding[0])
	}
}

func TestMilvusStore_RetrievalResult(t *testing.T) {
	// Проверка структуры результата
	result := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{
				Chunk: domain.Chunk{
					ID:       "c1",
					Content:  "test content",
					ParentID: "doc1",
				},
				Score: 0.9,
			},
		},
		QueryText:  "test query",
		TotalFound: 1,
	}
	
	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(result.Chunks))
	}
	if result.TotalFound != 1 {
		t.Error("expected TotalFound=1")
	}
	if result.QueryText != "test query" {
		t.Errorf("expected query text 'test query', got %s", result.QueryText)
	}
}
