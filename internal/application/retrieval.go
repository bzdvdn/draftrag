package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) maybeRerank(ctx context.Context, query string, result domain.RetrievalResult) (domain.RetrievalResult, error) {
	if p.reranker == nil {
		return result, nil
	}
	reranked, err := p.reranker.Rerank(ctx, query, result.Chunks)
	if err != nil {
		return result, fmt.Errorf("reranker: %w", err)
	}
	result.Chunks = reranked
	return result, nil
}

func (p *Pipeline) maybeDedup(result domain.RetrievalResult) domain.RetrievalResult {
	if !p.dedupByParentID {
		return result
	}
	result.Chunks = dedupRetrievedChunksByParentID(result.Chunks)
	return result
}

func dedupRetrievedChunksByParentID(chunks []domain.RetrievedChunk) []domain.RetrievedChunk {
	if len(chunks) == 0 {
		return chunks
	}

	type best struct {
		chunk domain.RetrievedChunk
		ix    int
	}

	bestByParent := make(map[string]best, len(chunks))
	for i, rc := range chunks {
		parentID := rc.Chunk.ParentID
		prev, ok := bestByParent[parentID]
		if !ok {
			bestByParent[parentID] = best{chunk: rc, ix: i}
			continue
		}

		// Выбираем лучший по score; при равенстве оставляем более ранний (детерминизм).
		if rc.Score > prev.chunk.Score {
			bestByParent[parentID] = best{chunk: rc, ix: i}
		}
	}

	out := make([]best, 0, len(bestByParent))
	for _, v := range bestByParent {
		out = append(out, v)
	}

	// Порядок по релевантности: score desc, tie-breaker — исходный индекс (stable).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].chunk.Score == out[j].chunk.Score {
			return out[i].ix < out[j].ix
		}
		return out[i].chunk.Score > out[j].chunk.Score
	})

	deduped := make([]domain.RetrievedChunk, 0, len(out))
	for _, v := range out {
		deduped = append(deduped, v.chunk)
	}
	return deduped
}
