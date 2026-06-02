package application

import (
	"sort"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func parseMultiQueryLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func rrfMergeMultiple(lists []domain.RetrievalResult, topK int) domain.RetrievalResult {
	const k = 60
	scores := make(map[string]float64)
	byID := make(map[string]domain.RetrievedChunk)
	for _, res := range lists {
		for rank, rc := range res.Chunks {
			id := rc.Chunk.ID
			scores[id] += 1.0 / float64(k+rank+1)
			byID[id] = rc
		}
	}
	merged := make([]domain.RetrievedChunk, 0, len(scores))
	for id, rc := range byID {
		rc.Score = scores[id]
		merged = append(merged, rc)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	if topK > 0 && len(merged) > topK {
		merged = merged[:topK]
	}
	return domain.RetrievalResult{Chunks: merged}
}
