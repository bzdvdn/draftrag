package vectorstore

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestCalculateRRF(t *testing.T) {
	// Тестовые данные
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
		{Chunk: domain.Chunk{ID: "c"}, Score: 0.7},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.95}, // b есть в обоих
		{Chunk: domain.Chunk{ID: "d"}, Score: 0.85},
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.75}, // a есть в обоих
	}

	result := calculateRRF(semantic, bm25, 60)

	// Проверяем что все уникальные ID присутствуют
	if len(result) != 4 {
		t.Errorf("expected 4 unique results, got %d", len(result))
	}

	// Проверяем что результаты отсортированы по score desc
	for i := 1; i < len(result); i++ {
		if result[i].score > result[i-1].score {
			t.Errorf("results not sorted by score at index %d", i)
		}
	}

	// Проверяем что b и a имеют более высокие score (так как есть в обоих списках)
	idToScore := make(map[string]float64)
	for _, r := range result {
		idToScore[r.chunk.ID] = r.score
	}

	// b и a должны иметь score > чем c и d (так как получили score от двух источников)
	if idToScore["b"] <= idToScore["c"] {
		t.Error("expected 'b' to have higher score than 'c' (appears in both lists)")
	}
	if idToScore["a"] <= idToScore["d"] {
		t.Error("expected 'a' to have higher score than 'd' (appears in both lists)")
	}
}

func TestCalculateRRF_DefaultK(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
	}

	// k=0 должно использовать default 60
	result := calculateRRF(semantic, bm25, 0)
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestCalculateWeightedScore(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.95}, // b в обоих
		{Chunk: domain.Chunk{ID: "c"}, Score: 0.85},
	}

	// wSemantic = 0.7, wBM25 = 0.3
	result := calculateWeightedScore(semantic, bm25, 0.7)

	// Проверяем количество
	if len(result) != 3 {
		t.Errorf("expected 3 unique results, got %d", len(result))
	}

	// Проверяем сортировку
	for i := 1; i < len(result); i++ {
		if result[i].score > result[i-1].score {
			t.Errorf("results not sorted by score at index %d", i)
		}
	}

	// b должен быть первым (0.7*0.8 + 0.3*0.95 = 0.56 + 0.285 = 0.845)
	if result[0].chunk.ID != "b" {
		t.Errorf("expected 'b' first, got %s", result[0].chunk.ID)
	}
}

func TestCalculateWeightedScore_OnlySemantic(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "c"}, Score: 0.95},
	}

	// wSemantic = 1.0 (только semantic)
	result := calculateWeightedScore(semantic, bm25, 1.0)

	// a и b должны иметь score, c должен иметь score 0 (так как wBM25 = 0)
	idToScore := make(map[string]float64)
	for _, r := range result {
		idToScore[r.chunk.ID] = r.score
	}

	if idToScore["a"] == 0 {
		t.Error("expected 'a' to have non-zero score")
	}
	if idToScore["c"] != 0 {
		t.Errorf("expected 'c' to have score 0 (wBM25=0), got %f", idToScore["c"])
	}
}

func TestCalculateWeightedScore_OnlyBM25(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
	}

	// wSemantic = 0.0 (только BM25)
	result := calculateWeightedScore(semantic, bm25, 0.0)

	idToScore := make(map[string]float64)
	for _, r := range result {
		idToScore[r.chunk.ID] = r.score
	}

	if idToScore["a"] != 0 {
		t.Errorf("expected 'a' to have score 0 (wSemantic=0), got %f", idToScore["a"])
	}
	if idToScore["b"] == 0 {
		t.Error("expected 'b' to have non-zero score")
	}
}

func TestFuseResults_RRF(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "c"}, Score: 0.95},
	}

	config := domain.HybridConfig{
		UseRRF:   true,
		RRFK:     60,
		BMFinalK: 2, // вернуть только top 2
	}

	result := fuseResults(semantic, bm25, config)

	if len(result) != 2 {
		t.Errorf("expected 2 results (BMFinalK=2), got %d", len(result))
	}

	// Проверяем сортировку
	for i := 1; i < len(result); i++ {
		if result[i].Score > result[i-1].Score {
			t.Errorf("results not sorted by score at index %d", i)
		}
	}
}

func TestFuseResults_Weighted(t *testing.T) {
	semantic := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a"}, Score: 0.9},
	}
	bm25 := []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "b"}, Score: 0.8},
	}

	config := domain.HybridConfig{
		UseRRF:         false,
		SemanticWeight: 0.6,
		BMFinalK:       10, // явно запрашиваем больше результатов
	}

	result := fuseResults(semantic, bm25, config)

	// Должны вернуться все уникальные результаты
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestFuseResults_Empty(t *testing.T) {
	// Пустые результаты
	semantic := []domain.RetrievedChunk{}
	bm25 := []domain.RetrievedChunk{}

	config := domain.HybridConfig{
		UseRRF: true,
		RRFK:   60,
	}

	result := fuseResults(semantic, bm25, config)

	if len(result) != 0 {
		t.Errorf("expected 0 results for empty inputs, got %d", len(result))
	}
}
