package draftrag

import (
	"errors"

	"github.com/bzdvdn/draftrag/internal/infrastructure/reranker"
)

// LLMReranker — публичный тип, обёртка над internal reranker.
type LLMReranker = reranker.LLMReranker

// LLMRerankerOption — функциональная опция для NewLLMReranker.
type LLMRerankerOption interface {
	apply(promptTemplate *string, batchSize *int, maxRetries *int)
}

type llmRerankerOptionFunc func(promptTemplate *string, batchSize *int, maxRetries *int)

func (f llmRerankerOptionFunc) apply(promptTemplate *string, batchSize *int, maxRetries *int) {
	f(promptTemplate, batchSize, maxRetries)
}

// WithBatchSize задаёт количество чанков в одном LLM-вызове (default: 10).
func WithBatchSize(n int) LLMRerankerOption {
	return llmRerankerOptionFunc(func(_ *string, batchSize *int, _ *int) {
		*batchSize = n
	})
}

// WithPromptTemplate задаёт кастомный system prompt для judge.
func WithPromptTemplate(tmpl string) LLMRerankerOption {
	return llmRerankerOptionFunc(func(promptTemplate *string, _ *int, _ *int) {
		*promptTemplate = tmpl
	})
}

// WithMaxRetries задаёт количество повторных попыток при ошибке LLM (default: 1).
func WithMaxRetries(n int) LLMRerankerOption {
	return llmRerankerOptionFunc(func(_ *string, _ *int, maxRetries *int) {
		*maxRetries = n
	})
}

// NewLLMReranker создаёт LLM-as-judge reranker.
// @sk-task reranker-llm-based#T1.2: публичный конструктор с опциями (AC-001, AC-002)
// llm — обязательный LLMProvider для скоринга чанков.
// Опции: WithBatchSize, WithPromptTemplate, WithMaxRetries.
//
// Пример:
//
//	rr, err := draftrag.NewLLMReranker(llm,
//	    draftrag.WithBatchSize(5),
//	    draftrag.WithPromptTemplate("Rate relevance 0-10..."),
//	)
//	pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, PipelineOptions{
//	    Reranker: rr,
//	})
func NewLLMReranker(llm LLMProvider, opts ...LLMRerankerOption) (*LLMReranker, error) {
	if llm == nil {
		return nil, errors.New("reranker: nil llm")
	}

	var promptTemplate string
	batchSize := 10
	maxRetries := 1

	for _, o := range opts {
		o.apply(&promptTemplate, &batchSize, &maxRetries)
	}

	return reranker.NewLLMReranker(llm, promptTemplate, batchSize, maxRetries)
}
