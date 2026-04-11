package application

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var (
	errMMRInvalidLambda    = errors.New("mmr lambda must be in [0..1]")
	errMMREmptyQueryVector = errors.New("mmr requires non-empty query embedding")
	errMMREmbeddingMissing = errors.New("mmr requires chunk embeddings in retrieval results")
	errMMRDimMismatch      = errors.New("mmr embedding dimension mismatch")
)

type mmrCandidate struct {
	rc        domain.RetrievedChunk
	ix        int
	relevance float64
}

func selectMMR(
	ctx context.Context,
	queryEmbedding []float64,
	candidates []domain.RetrievedChunk,
	topK int,
	lambda float64,
) ([]domain.RetrievedChunk, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if topK <= 0 {
		return nil, errors.New("topK must be > 0")
	}
	if lambda < 0 || lambda > 1 {
		return nil, errMMRInvalidLambda
	}
	if len(queryEmbedding) == 0 {
		return nil, errMMREmptyQueryVector
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// Подготовим кандидатов: relevance = исходный score (как baseline retrieval),
	// diversity = cosine(chunk, selectedChunk).
	prepared := make([]mmrCandidate, 0, len(candidates))
	for i, rc := range candidates {
		emb := rc.Chunk.Embedding
		if len(emb) == 0 {
			return nil, errMMREmbeddingMissing
		}
		if len(emb) != len(queryEmbedding) {
			return nil, errMMRDimMismatch
		}
		prepared = append(prepared, mmrCandidate{
			rc:        rc,
			ix:        i,
			relevance: rc.Score,
		})
	}

	// Детерминизм: при равной relevance выбираем более ранний кандидат.
	sort.SliceStable(prepared, func(i, j int) bool {
		if prepared[i].relevance == prepared[j].relevance {
			return prepared[i].ix < prepared[j].ix
		}
		return prepared[i].relevance > prepared[j].relevance
	})

	target := topK
	if target > len(prepared) {
		target = len(prepared)
	}

	selected := make([]mmrCandidate, 0, target)
	remaining := prepared

	for len(selected) < target {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		bestIdx := 0
		bestScore := math.Inf(-1)

		for i := range remaining {
			// Diversity = max cosine(candidate, any selected).
			div := 0.0
			if len(selected) > 0 {
				div = maxCosineToSelected(remaining[i].rc.Chunk.Embedding, selected)
			}

			score := lambda*remaining[i].relevance - (1-lambda)*div
			if score > bestScore {
				bestScore = score
				bestIdx = i
				continue
			}
			// Tie-breaker: исходный индекс (детерминизм).
			if score == bestScore && remaining[i].ix < remaining[bestIdx].ix {
				bestIdx = i
			}
		}

		selected = append(selected, remaining[bestIdx])
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	out := make([]domain.RetrievedChunk, 0, len(selected))
	for _, c := range selected {
		out = append(out, c.rc)
	}
	return out, nil
}

func maxCosineToSelected(embedding []float64, selected []mmrCandidate) float64 {
	max := math.Inf(-1)
	for _, s := range selected {
		v := cosine(embedding, s.rc.Chunk.Embedding)
		if v > max {
			max = v
		}
	}
	if max == math.Inf(-1) {
		return 0
	}
	return max
}

func cosine(a, b []float64) float64 {
	// Требуем одинаковую размерность (проверяется на входе), но держим guard на всякий случай.
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	dot := 0.0
	na := 0.0
	nb := 0.0
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
