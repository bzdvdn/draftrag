package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
const hydeSystemPrompt = "You are a helpful assistant. Write a short factual passage that would directly answer the question. Write only the passage."

// QueryHyDE выполняет поиск с использованием Hypothetical Document Embeddings.
// Сначала LLM генерирует гипотетический ответ на вопрос, затем ищем по его embedding.
func (p *Pipeline) QueryHyDE(ctx context.Context, question string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	genStart := time.Now()
	p.hookStart(ctx, "QueryHyDE:generate", domain.HookStageGenerate)
	hypothetical, err := p.llm.Generate(ctx, hydeSystemPrompt, question)
	p.hookEnd(ctx, "QueryHyDE:generate", domain.HookStageGenerate, genStart, err)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("hyde generate: %w", err)
	}

	embedStart := time.Now()
	p.hookStart(ctx, "QueryHyDE:embed", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, hypothetical)
	p.hookEnd(ctx, "QueryHyDE:embed", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("hyde embed: %w", err)
	}

	searchStart := time.Now()
	p.hookStart(ctx, "QueryHyDE:search", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, topK)
	p.hookEnd(ctx, "QueryHyDE:search", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question
	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	return result, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
const multiQuerySystemPrompt = "You are a helpful assistant. Generate alternative phrasings of the given question to improve document retrieval. Output only the questions, one per line, no numbering, no extra text."

// QueryMulti выполняет multi-query retrieval: генерирует n перефразировок вопроса,
// выполняет поиск по каждой, объединяет результаты через Reciprocal Rank Fusion.
func (p *Pipeline) QueryMulti(ctx context.Context, question string, n, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if n <= 0 {
		n = 3
	}

	userMsg := fmt.Sprintf("Generate %d alternative phrasings of this question:\n\n%s", n, question)
	genStart := time.Now()
	p.hookStart(ctx, "QueryMulti:generate", domain.HookStageGenerate)
	raw, err := p.llm.Generate(ctx, multiQuerySystemPrompt, userMsg)
	p.hookEnd(ctx, "QueryMulti:generate", domain.HookStageGenerate, genStart, err)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("multi-query generate: %w", err)
	}

	queries := parseMultiQueryLines(raw)
	queries = append([]string{question}, queries...)

	allResults := make([]domain.RetrievalResult, 0, len(queries))
	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return domain.RetrievalResult{}, err
		}
		embedStart := time.Now()
		p.hookStart(ctx, "QueryMulti:embed", domain.HookStageEmbed)
		emb, err := p.embedder.Embed(ctx, q)
		p.hookEnd(ctx, "QueryMulti:embed", domain.HookStageEmbed, embedStart, err)
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("multi-query embed: %w", err)
		}
		searchStart := time.Now()
		p.hookStart(ctx, "QueryMulti:search", domain.HookStageSearch)
		res, err := p.store.Search(ctx, emb, topK)
		p.hookEnd(ctx, "QueryMulti:search", domain.HookStageSearch, searchStart, err)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		allResults = append(allResults, res)
	}

	merged := rrfMergeMultiple(allResults, topK)
	merged = p.maybeDedup(merged)
	merged.QueryText = question
	merged, err = p.maybeRerank(ctx, question, merged)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	return merged, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// Query выполняет поиск по вопросу и возвращает RetrievalResult.
func (p *Pipeline) Query(ctx context.Context, question string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, topK)
	p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// QueryWithParentIDs выполняет поиск по вопросу с фильтром по ParentIDs.
//
// Если parentIDs пустой — эквивалентно Query.
func (p *Pipeline) QueryWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (domain.RetrievalResult, error) {
	if len(parentIDs) == 0 {
		return p.Query(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageSearch)
	result, err := vs.SearchWithFilter(ctx, embedding, topK, domain.ParentIDFilter{ParentIDs: parentIDs})
	p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// QueryWithMetadataFilter выполняет поиск по вопросу с фильтром по метаданным документа.
//
// Если filter.Fields пустой — эквивалентно Query.
// Если store не реализует VectorStoreWithFilters — возвращает ErrFiltersNotSupported.
//
// @ds-task T3.1: Добавить QueryWithMetadataFilter в application.Pipeline (RQ-005, AC-003, DEC-003)
func (p *Pipeline) QueryWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return p.Query(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Query", domain.HookStageSearch)
	result, err := vs.SearchWithMetadataFilter(ctx, embedding, topK, filter)
	p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// ErrHybridNotSupported возвращается, если pipeline-метод гибридного поиска вызван,
// но underlying VectorStore не поддерживает HybridSearcher capability.
var ErrHybridNotSupported = errors.New("vector store does not support hybrid search")

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// QueryHybrid выполняет гибридный поиск (BM25 + semantic) по вопросу.
//
// Если store не реализует HybridSearcher — возвращает ErrHybridNotSupported.
func (p *Pipeline) QueryHybrid(ctx context.Context, question string, topK int, config domain.HybridConfig) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return domain.RetrievalResult{}, errors.New("question is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}

	hs, ok := p.store.(domain.HybridSearcher)
	if !ok {
		return domain.RetrievalResult{}, ErrHybridNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "QueryHybrid", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "QueryHybrid", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	searchStart := time.Now()
	p.hookStart(ctx, "QueryHybrid", domain.HookStageSearch)
	result, err := hs.SearchHybrid(ctx, question, embedding, topK, config)
	p.hookEnd(ctx, "QueryHybrid", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	result = p.maybeDedup(result)
	result.QueryText = question

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}
