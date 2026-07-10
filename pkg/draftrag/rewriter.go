package draftrag

import (
	"github.com/bzdvdn/draftrag/internal/infrastructure/rewriter"
)

// @sk-task query-rewriting#T3.3: NewLLMRewriter публичный конструктор (AC-006)

// NewLLMRewriter создаёт LLM-based реализацию QueryRewriter.
// llm — обязательный LLMProvider.
// promptTemplate — опциональный system prompt (пустая строка = дефолтный промпт
// для однократного переписывания запроса).
//
// Пример:
//
//	rw, err := draftrag.NewLLMRewriter(llm, "")
//	result, err := pipeline.Search("question").Rewriter(rw).Retrieve(ctx)
func NewLLMRewriter(llm LLMProvider, promptTemplate string) (QueryRewriter, error) {
	return rewriter.NewLLMRewriter(llm, promptTemplate)
}
