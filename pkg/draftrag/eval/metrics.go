package eval

import (
	"strings"
)

func normalizeID(s string) string {
	return strings.TrimSpace(s)
}

func buildExpectedSet(expected []string) map[string]struct{} {
	set := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		id = normalizeID(id)
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	return set
}

// rankByParentID возвращает rank (1..len(retrieved)) первого совпадения expected с retrieved.
// Если совпадений нет — возвращает 0.
func rankByParentID(expected []string, retrievedParentIDs []string) int {
	expectedSet := buildExpectedSet(expected)
	if len(expectedSet) == 0 {
		return 0
	}

	for i, id := range retrievedParentIDs {
		id = normalizeID(id)
		if id == "" {
			continue
		}
		if _, ok := expectedSet[id]; ok {
			return i + 1
		}
	}
	return 0
}

func computeMetrics(caseResults []CaseResult) Metrics {
	total := len(caseResults)
	if total == 0 {
		return Metrics{}
	}

	hits := 0
	mrrSum := 0.0
	for _, cr := range caseResults {
		if cr.Found {
			hits++
		}
		if cr.Rank > 0 {
			mrrSum += 1.0 / float64(cr.Rank)
		}
	}

	return Metrics{
		TotalCases: total,
		HitAtK:     float64(hits) / float64(total),
		MRR:        mrrSum / float64(total),
	}
}
