package eval

import (
	"math"
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

// @sk-task T2.1: computeNDCG вычисляет NDCG@K для одного кейса с бинарной релевантностью (AC-001)
func computeNDCG(expected []string, retrievedParentIDs []string) float64 {
	expectedSet := buildExpectedSet(expected)
	if len(expectedSet) == 0 {
		return 0
	}

	// Вычисляем DCG (Discounted Cumulative Gain) для retrieved
	dcg := 0.0
	for i, id := range retrievedParentIDs {
		id = normalizeID(id)
		if id == "" {
			continue
		}
		if _, ok := expectedSet[id]; ok {
			// Бинарная релевантность: 1 если документ релевантен, иначе 0
			relevance := 1.0
			dcg += relevance / math.Log2(float64(i+2))
		}
	}

	// Вычисляем IDCG (Ideal DCG) - идеальное ранжирование (все релевантные документы в начале)
	idcg := 0.0
	i := 0
	for range expectedSet {
		idcg += 1.0 / math.Log2(float64(i+2))
		i++
	}

	if idcg == 0 {
		return 0
	}

	return dcg / idcg
}

// @sk-task T2.2: computePrecision вычисляет Precision@K для одного кейса (AC-002)
func computePrecision(expected []string, retrievedParentIDs []string, k int) float64 {
	expectedSet := buildExpectedSet(expected)
	if len(expectedSet) == 0 || k <= 0 {
		return 0
	}

	relevantCount := 0
	for i := 0; i < k && i < len(retrievedParentIDs); i++ {
		id := normalizeID(retrievedParentIDs[i])
		if id == "" {
			continue
		}
		if _, ok := expectedSet[id]; ok {
			relevantCount++
		}
	}

	return float64(relevantCount) / float64(k)
}

// @sk-task T2.2: computeRecall вычисляет Recall@K для одного кейса (AC-002)
func computeRecall(expected []string, retrievedParentIDs []string, k int) float64 {
	expectedSet := buildExpectedSet(expected)
	if len(expectedSet) == 0 {
		return 0
	}

	relevantCount := 0
	for i := 0; i < k && i < len(retrievedParentIDs); i++ {
		id := normalizeID(retrievedParentIDs[i])
		if id == "" {
			continue
		}
		if _, ok := expectedSet[id]; ok {
			relevantCount++
		}
	}

	return float64(relevantCount) / float64(len(expectedSet))
}

// @sk-task T2.5: computeMetrics обновлена для условного вычисления новых метрик (AC-003, AC-004)
func computeMetrics(cases []Case, caseResults []CaseResult, opts Options) Metrics {
	total := len(caseResults)
	if total == 0 {
		return Metrics{}
	}

	hits := 0
	mrrSum := 0.0
	ndcgSum := 0.0
	precisionSum := 0.0
	recallSum := 0.0

	for i, cr := range caseResults {
		if cr.Found {
			hits++
		}
		if cr.Rank > 0 {
			mrrSum += 1.0 / float64(cr.Rank)
		}

		// Вычисляем per-case метрики если включены соответствующие флаги
		if i < len(cases) {
			c := cases[i]
			if opts.EnableNDCG {
				cr.NDCG = computeNDCG(c.ExpectedParentIDs, cr.RetrievedParentIDs)
				ndcgSum += cr.NDCG
			}
			if opts.EnablePrecision {
				k := len(cr.RetrievedParentIDs)
				if k == 0 {
					k = 5 // дефолт если retrieved пустой
				}
				cr.Precision = computePrecision(c.ExpectedParentIDs, cr.RetrievedParentIDs, k)
				precisionSum += cr.Precision
			}
			if opts.EnableRecall {
				k := len(cr.RetrievedParentIDs)
				if k == 0 {
					k = 5 // дефолт если retrieved пустой
				}
				cr.Recall = computeRecall(c.ExpectedParentIDs, cr.RetrievedParentIDs, k)
				recallSum += cr.Recall
			}
			// Обновляем caseResults с заполненными per-case метриками
			caseResults[i] = cr
		}
	}

	metrics := Metrics{
		TotalCases: total,
		HitAtK:     float64(hits) / float64(total),
		MRR:        mrrSum / float64(total),
	}

	if opts.EnableNDCG {
		metrics.NDCG = ndcgSum / float64(total)
	}
	if opts.EnablePrecision {
		metrics.Precision = precisionSum / float64(total)
	}
	if opts.EnableRecall {
		metrics.Recall = recallSum / float64(total)
	}

	return metrics
}
