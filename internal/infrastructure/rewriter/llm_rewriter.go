// Package rewriter implements LLM-based query rewriting strategy.
package rewriter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task query-rewriting#T3.3: LLMRewriter реализация (AC-006)

const defaultLLMRewriterPrompt = `Rewrite the following question to improve document retrieval for a RAG system. Output only the rewritten question, no additional text.

Original: %s
Rewritten:`

// LLMRewriter — встроенная LLM-стратегия переписывания запросов.
//
// Использует LLMProvider для генерации переформулировки.
// По умолчанию работает в режиме 1:1; для multi-query используйте кастомный prompt,
// который возвращает несколько строк (по одной переформулировке на строку).
type LLMRewriter struct {
	llm            domain.LLMProvider
	promptTemplate string
}

// NewLLMRewriter создаёт LLMRewriter.
// llm — обязательный LLMProvider.
// promptTemplate — опциональный system prompt (пустая строка = дефолтный).
func NewLLMRewriter(llm domain.LLMProvider, promptTemplate string) (*LLMRewriter, error) {
	if llm == nil {
		return nil, errors.New("rewriter: nil llm")
	}
	return &LLMRewriter{
		llm:            llm,
		promptTemplate: promptTemplate,
	}, nil
}

// @sk-task query-rewriting#T3.3: LLMRewriter.Rewrite (AC-006)
// Rewrite переписывает запрос через LLM с учётом истории диалога.
func (r *LLMRewriter) Rewrite(ctx context.Context, query string, history domain.QueryHistory) ([]domain.RewrittenQuery, error) {
	prompt := r.promptTemplate
	if prompt == "" {
		prompt = defaultLLMRewriterPrompt
	}

	var userMsg string
	if len(history.Entries) > 0 {
		var b strings.Builder
		for _, msg := range history.Entries {
			b.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
		userMsg = fmt.Sprintf("Conversation history:\n%sQuery: %s", b.String(), query)
	} else {
		userMsg = query
	}

	response, err := r.llm.Generate(ctx, prompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("rewriter llm generate: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(response), "\n")
	var rewritten []domain.RewrittenQuery
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			rewritten = append(rewritten, domain.RewrittenQuery{Query: line, Weight: 1.0})
		}
	}

	if len(rewritten) == 0 {
		rewritten = append(rewritten, domain.RewrittenQuery{Query: query, Weight: 1.0})
	}

	return rewritten, nil
}
