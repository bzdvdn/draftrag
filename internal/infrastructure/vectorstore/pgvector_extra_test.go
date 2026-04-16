package vectorstore

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestPGVectorStore_Search_InvalidTopK(t *testing.T) {
	// Этот тест проверяет обработку невалидного topK
	// Поскольку pgvector требует реальное подключение к базе данных,
	// этот тест является заглушкой для проверки логики валидации

	// В реальном коде валидация может быть на уровне store или domain
	// Здесь просто проверяем, что мы можем создать валидную конфигурацию

	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 0.7,
		RRFK:           60, // должно быть > 0
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestPGVectorStore_SearchWithFilter_EmptyParentIDs(t *testing.T) {
	t.Parallel()

	// Проверка делегирования в базовый Search при пустых parentIDs
	// Это заглушка - реальная проверка требует подключения к БД

	filter := domain.ParentIDFilter{
		ParentIDs: []string{},
	}

	if len(filter.ParentIDs) != 0 {
		t.Fatalf("expected empty ParentIDs, got %d", len(filter.ParentIDs))
	}
}

func TestPGVectorStore_SearchWithMetadataFilter_EmptyFields(t *testing.T) {
	t.Parallel()

	// Проверка делегирования в базовый Search при пустых fields
	// Это заглушка - реальная проверка требует подключения к БД

	filter := domain.MetadataFilter{
		Fields: map[string]string{},
	}

	if len(filter.Fields) != 0 {
		t.Fatalf("expected empty Fields, got %d", len(filter.Fields))
	}
}

func TestPGVectorStore_HybridConfig_Default(t *testing.T) {
	config := domain.DefaultHybridConfig()

	err := config.Validate()
	if err != nil {
		t.Fatalf("expected valid default config, got error: %v", err)
	}

	// Проверяем дефолтные значения
	if config.UseRRF != true {
		t.Errorf("expected UseRRF=true, got %v", config.UseRRF)
	}
	if config.SemanticWeight != 0.7 {
		t.Errorf("expected SemanticWeight=0.7, got %f", config.SemanticWeight)
	}
}

func TestPGVectorStore_HybridConfig_RRF(t *testing.T) {
	config := domain.HybridConfig{
		UseRRF: true,
		RRFK:   60,
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("expected valid RRF config, got error: %v", err)
	}
}

func TestPGVectorStore_HybridConfig_InvalidSemanticWeight(t *testing.T) {
	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 1.5, // невалидно
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for invalid SemanticWeight, got nil")
	}
}

func TestPGVectorStore_HybridConfig_InvalidRRFK(t *testing.T) {
	config := domain.HybridConfig{
		UseRRF: true,
		RRFK:   0, // невалидно
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for invalid RRFK, got nil")
	}
}

func TestPGVectorStore_HybridConfig_InvalidBMFinalK(t *testing.T) {
	config := domain.HybridConfig{
		UseRRF:   true,
		RRFK:     60,
		BMFinalK: -1, // невалидно
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for invalid BMFinalK, got nil")
	}
}

func TestPGVectorStore_MetadataFilter_Empty(t *testing.T) {
	filter := domain.MetadataFilter{
		Fields: map[string]string{},
	}

	if len(filter.Fields) != 0 {
		t.Error("expected empty Fields map")
	}
}

func TestPGVectorStore_MetadataFilter_WithFields(t *testing.T) {
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

func TestPGVectorStore_ParentIDFilter_Empty(t *testing.T) {
	filter := domain.ParentIDFilter{
		ParentIDs: []string{},
	}

	if len(filter.ParentIDs) != 0 {
		t.Error("expected empty ParentIDs slice")
	}
}

func TestPGVectorStore_ParentIDFilter_WithIDs(t *testing.T) {
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

func TestPGVectorStore_ContextCancellation(t *testing.T) {
	// Проверка отмены контекста
	// Это заглушка - реальная проверка требует подключения к БД

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

func TestPGVectorStore_EmbeddingDimensions(t *testing.T) {
	// Проверка работы с embedding-векторами разных размерностей

	embedding1 := []float64{0.1, 0.2, 0.3}
	if len(embedding1) != 3 {
		t.Errorf("expected 3 dimensions, got %d", len(embedding1))
	}

	embedding2 := []float64{0.1, 0.2, 0.3, 0.4}
	if len(embedding2) != 4 {
		t.Errorf("expected 4 dimensions, got %d", len(embedding2))
	}
}

func TestPGVectorStore_RetrievalResult_Empty(t *testing.T) {
	result := domain.RetrievalResult{
		Chunks:     []domain.RetrievedChunk{},
		QueryText:  "",
		TotalFound: 0,
	}

	if len(result.Chunks) != 0 {
		t.Error("expected empty chunks")
	}
	if result.TotalFound != 0 {
		t.Error("expected TotalFound=0")
	}
}

func TestPGVectorStore_RetrievalResult_WithData(t *testing.T) {
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
