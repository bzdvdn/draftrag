package decomposer

import (
	"context"
	"strings"
)

// разделители для rule-based декомпозиции.
var ruleSeparators = []string{" и ", " или ", ", "}

// @sk-task sub-query-decomposition#T3.1: RuleQueryDecomposer (AC-003)
// RuleQueryDecomposer разбивает запрос на под-вопросы по союзам/разделителям.
//
// Правила:
// - Разбивка по " и ", " или ", ", " (регистронезависимо).
// - Каждая часть возвращается как отдельный под-вопрос.
// - Если разделители не найдены — возвращается nil (single-query fallback).
// - Пустые части отбрасываются.
type RuleQueryDecomposer struct{}

// NewRuleQueryDecomposer создаёт RuleQueryDecomposer.
func NewRuleQueryDecomposer() *RuleQueryDecomposer {
	return &RuleQueryDecomposer{}
}

// @sk-task sub-query-decomposition#T3.1: Decompose — rule-based (AC-003)
// Decompose разбивает запрос на под-вопросы по разделителям.
// Выбирает разделитель, который даёт максимальное количество частей.
func (d *RuleQueryDecomposer) Decompose(_ context.Context, query string) ([]string, error) {
	lower := strings.ToLower(query)
	var best []string

	for _, sep := range ruleSeparators {
		if !strings.Contains(lower, sep) {
			continue
		}
		parts := strings.Split(query, sep)
		var result []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		if len(result) > len(best) {
			best = result
		}
	}

	if len(best) > 1 {
		return best, nil
	}
	return nil, nil
}
