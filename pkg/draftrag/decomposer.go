package draftrag

import (
	"fmt"

	"github.com/bzdvdn/draftrag/internal/infrastructure/decomposer"
)

// NewLLMQueryDecomposer создаёт LLM-based реализацию QueryDecomposer.
// llm — обязательный LLMProvider для генерации под-вопросов.
// promptTemplate — опциональный system prompt (пустая строка = дефолтный промпт).
func NewLLMQueryDecomposer(llm LLMProvider, promptTemplate string) (QueryDecomposer, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLMProvider is required")
	}
	return decomposer.NewLLMQueryDecomposer(llm, promptTemplate)
}

// NewRuleQueryDecomposer создаёт rule-based реализацию QueryDecomposer,
// разбивающую запрос на под-вопросы по союзам "и", "или" и запятым.
func NewRuleQueryDecomposer() QueryDecomposer {
	return decomposer.NewRuleQueryDecomposer()
}
