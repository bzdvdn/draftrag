package vectorstore

import (
	"sort"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// fusionResult внутренняя структура для хранения промежуточных результатов fusion.
type fusionResult struct {
	chunk domain.Chunk
	score float64
}

// calculateRRF выполняет Reciprocal Rank Fusion (RRF) двух списков результатов.
// Формула: score = Σ 1/(k + rank)
// где k = rrfK (обычно 60), rank = позиция в списке (начиная с 1).
func calculateRRF(semantic, bm25 []domain.RetrievedChunk, rrfK int) []fusionResult {
	if rrfK < 1 {
		rrfK = 60 // default
	}

	// Карта для агрегации score по chunk.ID
	scores := make(map[string]*fusionResult)

	// Обрабатываем semantic результаты
	for rank, rc := range semantic {
		id := rc.Chunk.ID
		if _, exists := scores[id]; !exists {
			scores[id] = &fusionResult{chunk: rc.Chunk, score: 0}
		}
		// RRF score: 1/(k + rank), rank начинается с 1
		scores[id].score += 1.0 / float64(rrfK+rank+1)
	}

	// Обрабатываем BM25 результаты
	for rank, rc := range bm25 {
		id := rc.Chunk.ID
		if _, exists := scores[id]; !exists {
			scores[id] = &fusionResult{chunk: rc.Chunk, score: 0}
		}
		scores[id].score += 1.0 / float64(rrfK+rank+1)
	}

	// Преобразуем map в slice
	result := make([]fusionResult, 0, len(scores))
	for _, fr := range scores {
		result = append(result, *fr)
	}

	// Сортируем по score (убывание)
	sort.Slice(result, func(i, j int) bool {
		return result[i].score > result[j].score
	})

	return result
}

// calculateWeightedScore выполняет weighted fusion двух списков результатов.
// Формула: score = w_semantic * norm(semantic_score) + w_bm25 * norm(bm25_score)
// semantic_score уже нормализован [0,1] (cosine similarity)
// bm25_score нормализуется через ts_rank_cd [0,1].
func calculateWeightedScore(semantic, bm25 []domain.RetrievedChunk, wSemantic float64) []fusionResult {
	if wSemantic < 0 {
		wSemantic = 0
	}
	if wSemantic > 1 {
		wSemantic = 1
	}
	wBM25 := 1.0 - wSemantic

	// Карта для агрегации score по chunk.ID
	scores := make(map[string]*fusionResult)

	// Обрабатываем semantic результаты
	for _, rc := range semantic {
		id := rc.Chunk.ID
		scores[id] = &fusionResult{
			chunk: rc.Chunk,
			score: wSemantic * rc.Score, // semantic score уже [0,1]
		}
	}

	// Обрабатываем BM25 результаты
	for _, rc := range bm25 {
		id := rc.Chunk.ID
		if _, exists := scores[id]; exists {
			// Документ найден в обоих списках
			scores[id].score += wBM25 * rc.Score
		} else {
			// Документ только в BM25
			scores[id] = &fusionResult{
				chunk: rc.Chunk,
				score: wBM25 * rc.Score,
			}
		}
	}

	// Преобразуем map в slice
	result := make([]fusionResult, 0, len(scores))
	for _, fr := range scores {
		result = append(result, *fr)
	}

	// Сортируем по score (убывание)
	sort.Slice(result, func(i, j int) bool {
		return result[i].score > result[j].score
	})

	return result
}

// fuseResults объединяет результаты semantic и BM25 поиска согласно конфигурации.
func fuseResults(semantic, bm25 []domain.RetrievedChunk, config domain.HybridConfig) []domain.RetrievedChunk {
	var fused []fusionResult

	if config.UseRRF {
		rrfK := config.RRFK
		if rrfK < 1 {
			rrfK = 60
		}
		fused = calculateRRF(semantic, bm25, rrfK)
	} else {
		wSemantic := config.SemanticWeight
		if wSemantic < 0 || wSemantic > 1 {
			wSemantic = 0.7 // default
		}
		fused = calculateWeightedScore(semantic, bm25, wSemantic)
	}

	// Определяем сколько результатов вернуть
	finalK := config.BMFinalK
	if finalK <= 0 {
		// Если не задано явно, используем исходный topK
		// (берём из длины semantic или bm25, whichever is larger)
		finalK = len(semantic)
		if len(bm25) > finalK {
			finalK = len(bm25)
		}
	}

	// Обрезаем до finalK
	if len(fused) > finalK {
		fused = fused[:finalK]
	}

	// Преобразуем в RetrievedChunk
	result := make([]domain.RetrievedChunk, len(fused))
	for i, fr := range fused {
		result[i] = domain.RetrievedChunk{
			Chunk: fr.chunk,
			Score: fr.score,
		}
	}

	return result
}
