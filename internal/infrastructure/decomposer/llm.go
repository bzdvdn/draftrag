package decomposer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

const defaultDecomposePrompt = `Разбей следующий запрос на независимые под-вопросы, каждый из которых можно найти по отдельности в документации. Ответь ТОЛЬКО JSON-массивом строк, без пояснений. Пример: ["подвопрос 1", "подвопрос 2"]

Запрос: %s`

var jsonArrayRegex = regexp.MustCompile(`\[(?:"[^"]*"(?:,\s*)?)*\]`)

// @sk-task sub-query-decomposition#T2.1: LLMQueryDecomposer (AC-002)
// LLMQueryDecomposer разбивает запрос на под-вопросы через LLM.
type LLMQueryDecomposer struct {
	llm            domain.LLMProvider
	promptTemplate string
}

// NewLLMQueryDecomposer создаёт LLMQueryDecomposer.
// llm — обязательный LLMProvider.
// promptTemplate — опциональный system prompt (пустая строка = дефолтный).
func NewLLMQueryDecomposer(llm domain.LLMProvider, promptTemplate string) (*LLMQueryDecomposer, error) {
	if llm == nil {
		return nil, errors.New("llm decomposer: nil llm")
	}
	return &LLMQueryDecomposer{
		llm:            llm,
		promptTemplate: promptTemplate,
	}, nil
}

// @sk-task sub-query-decomposition#T2.1: Decompose — LLM-based (AC-002)
// Decompose разбивает запрос на под-вопросы через LLM.
// Парсит JSON-массив строк из ответа; при ошибке парсинга — fallback на regex extraction.
func (d *LLMQueryDecomposer) Decompose(ctx context.Context, query string) ([]string, error) {
	prompt := d.promptTemplate
	if prompt == "" {
		prompt = defaultDecomposePrompt
	}

	userMsg := fmt.Sprintf(prompt, query)
	response, err := d.llm.Generate(ctx, "", userMsg)
	if err != nil {
		return nil, fmt.Errorf("llm decomposer generate: %w", err)
	}

	response = strings.TrimSpace(response)
	subs := parseJSONStringArray(response)
	if len(subs) > 0 {
		return subs, nil
	}

	return []string{query}, nil
}

// parseJSONStringArray пытается распарсить JSON-массив строк.
// Сначала пробует json.Unmarshal, при неудаче — regex extraction.
func parseJSONStringArray(input string) []string {
	input = strings.TrimSpace(input)

	var result []string
	if err := json.Unmarshal([]byte(input), &result); err == nil {
		cleaned := make([]string, 0, len(result))
		for _, s := range result {
			s = strings.TrimSpace(s)
			if s != "" {
				cleaned = append(cleaned, s)
			}
		}
		return cleaned
	}

	matches := jsonArrayRegex.FindString(input)
	if matches == "" {
		return nil
	}

	var fallback []string
	if err := json.Unmarshal([]byte(matches), &fallback); err == nil {
		cleaned := make([]string, 0, len(fallback))
		for _, s := range fallback {
			s = strings.TrimSpace(s)
			if s != "" {
				cleaned = append(cleaned, s)
			}
		}
		return cleaned
	}

	return nil
}
