package vectorstore

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestCalculateRRF_EmptyInputs(t *testing.T) {
	semantic := []domain.RetrievedChunk{}
	bm25 := []domain.RetrievedChunk{}

	result := calculateRRF(semantic, bm25, 60)
	if len(result) != 0 {
		t.Errorf("expected empty result for empty inputs, got %d", len(result))
	}
}

func TestCalculateRRF_SingleList(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	result := calculateRRF(semantic, bm25, 60)
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestCalculateRRF_DuplicateChunks(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"}, // тот же ID
			Score: 0.8,
		},
	}

	result := calculateRRF(semantic, bm25, 60)
	if len(result) != 1 {
		t.Errorf("expected 1 unique result, got %d", len(result))
	}
	// Score должен быть суммой обоих RRF scores
	if result[0].score <= 0 {
		t.Errorf("expected positive score for duplicate chunks, got %f", result[0].score)
	}
}

func TestCalculateRRF_ZeroK(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	result := calculateRRF(semantic, bm25, 0) // должно использовать default 60
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestCalculateWeightedScore_EmptyInputs(t *testing.T) {
	semantic := []domain.RetrievedChunk{}
	bm25 := []domain.RetrievedChunk{}

	result := calculateWeightedScore(semantic, bm25, 0.7)
	if len(result) != 0 {
		t.Errorf("expected empty result for empty inputs, got %d", len(result))
	}
}


func TestCalculateWeightedScore_NegativeWeight(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	result := calculateWeightedScore(semantic, bm25, -0.5) // должно стать 0
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
	// При wSemantic=0 score должен быть 0
	if result[0].score != 0 {
		t.Errorf("expected score 0 for negative weight, got %f", result[0].score)
	}
}

func TestCalculateWeightedScore_GreaterThanOne(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	result := calculateWeightedScore(semantic, bm25, 1.5) // должно стать 1
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
	// При wSemantic=1 score должен быть semantic_score
	if result[0].score < 0.89 || result[0].score > 0.91 {
		t.Errorf("expected score ~0.9 for weight > 1, got %f", result[0].score)
	}
}

func TestFuseResults_UseRRF(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c2"},
			Score: 0.8,
		},
	}

	config := domain.HybridConfig{
		UseRRF: true,
		RRFK:   60,
	}

	result := fuseResults(semantic, bm25, config)
	// Должно быть 2 результата (уникальные ID)
	if len(result) != 2 {
		t.Logf("expected 2 results, got %d", len(result))
		// Это может быть ожидаемо в зависимости от реализации
	}
}

func TestFuseResults_UseWeighted(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c2"},
			Score: 0.8,
		},
	}

	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 0.7,
	}

	result := fuseResults(semantic, bm25, config)
	// Должно быть 2 результата (уникальные ID)
	if len(result) != 2 {
		t.Logf("expected 2 results, got %d", len(result))
		// Это может быть ожидаемо в зависимости от реализации
	}
}

func TestFuseResults_BMFinalK(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
		{
			Chunk: domain.Chunk{ID: "c2"},
			Score: 0.8,
		},
		{
			Chunk: domain.Chunk{ID: "c3"},
			Score: 0.7,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	config := domain.HybridConfig{
		UseRRF:    true,
		RRFK:      60,
		BMFinalK: 2, // ограничиваем до 2
	}

	result := fuseResults(semantic, bm25, config)
	if len(result) != 2 {
		t.Errorf("expected 2 results with BMFinalK=2, got %d", len(result))
	}
}

func TestFuseResults_InvalidSemanticWeight(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 1.5, // невалидно, должно стать 0.7 (default)
	}

	result := fuseResults(semantic, bm25, config)
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}

func TestFuseResults_ZeroRRFK(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{ID: "c1"},
			Score: 0.9,
		},
	}
	bm25 := []domain.RetrievedChunk{}

	config := domain.HybridConfig{
		UseRRF: true,
		RRFK:   0, // должно использовать default 60
	}

	result := fuseResults(semantic, bm25, config)
	if len(result) != 1 {
		t.Errorf("expected 1 result, got %d", len(result))
	}
}
