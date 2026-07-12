package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

const hydeSystemPrompt = "You are a helpful assistant. Write a short factual passage that would directly answer the question. Write only the passage."

// QueryHyDE выполняет поиск с использованием Hypothetical Document Embeddings.
//
// @sk-task hierarchical-indices#T3.3: parent context attach in QueryHyDE (AC-002)
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// Сначала LLM генерирует гипотетический ответ на вопрос, затем ищем по его embedding.
// @sk-task arch-issues#T2.1: PII redaction в QueryHyDE (AC-001)
func (p *Pipeline) QueryHyDE(ctx context.Context, question string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	question = p.redact(question)

	var hypothetical string
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageGenerate, "QueryHyDE", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		genStart := time.Now()
		p.hookStart(ctx, "QueryHyDE:generate", domain.HookStageGenerate)
		var genErr error
		hypothetical, genErr = p.llm.Generate(ctx, hydeSystemPrompt, question)
		p.hookEnd(ctx, "QueryHyDE:generate", domain.HookStageGenerate, genStart, genErr)
		if genErr != nil {
			return d, genErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("hyde generate: %w", err)
	}

	var embedding []float64
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "QueryHyDE", domain.StageData{Query: hypothetical}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "QueryHyDE:embed", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "QueryHyDE:embed", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("hyde embed: %w", err)
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "QueryHyDE", domain.StageData{Query: hypothetical, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "QueryHyDE:search", domain.HookStageSearch)
		var searchErr error
		result, searchErr = p.store.Search(ctx, d.Embedding, topK)
		p.hookEnd(ctx, "QueryHyDE:search", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result = p.maybeAttachParentContent(ctx, result)
	result.QueryText = question
	result = p.RedactRetrievalResult(result)
	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	return result, nil
}

const multiQuerySystemPrompt = "You are a helpful assistant. Generate alternative phrasings of the given question to improve document retrieval. Output only the questions, one per line, no numbering, no extra text."

// QueryMulti выполняет multi-query retrieval: генерирует n перефразировок вопроса,
//
// @sk-task hierarchical-indices#T3.3: parent context attach in QueryMulti (AC-002)
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// выполняет поиск по каждой, объединяет результаты через Reciprocal Rank Fusion.
// @sk-task arch-issues#T2.1: PII redaction в QueryMulti (AC-001)
func (p *Pipeline) QueryMulti(ctx context.Context, question string, n, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	question = p.redact(question)
	if n <= 0 {
		n = 3
	}

	userMsg := fmt.Sprintf("Generate %d alternative phrasings of this question:\n\n%s", n, question)
	var raw string
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageGenerate, "QueryMulti", domain.StageData{Query: userMsg}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		genStart := time.Now()
		p.hookStart(ctx, "QueryMulti:generate", domain.HookStageGenerate)
		var genErr error
		raw, genErr = p.llm.Generate(ctx, multiQuerySystemPrompt, userMsg)
		p.hookEnd(ctx, "QueryMulti:generate", domain.HookStageGenerate, genStart, genErr)
		if genErr != nil {
			return d, genErr
		}
		return d, nil
	})
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
		var emb []float64
		_, err = p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "QueryMulti", domain.StageData{Query: q}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
			embedStart := time.Now()
			p.hookStart(ctx, "QueryMulti:embed", domain.HookStageEmbed)
			var embedErr error
			emb, embedErr = p.embedder.Embed(ctx, d.Query)
			p.hookEnd(ctx, "QueryMulti:embed", domain.HookStageEmbed, embedStart, embedErr)
			if embedErr != nil {
				return d, embedErr
			}
			return d, nil
		})
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("multi-query embed: %w", err)
		}
		var res domain.RetrievalResult
		_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "QueryMulti", domain.StageData{Query: q, Embedding: emb}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
			searchStart := time.Now()
			p.hookStart(ctx, "QueryMulti:search", domain.HookStageSearch)
			var searchErr error
			res, searchErr = p.store.Search(ctx, d.Embedding, topK)
			p.hookEnd(ctx, "QueryMulti:search", domain.HookStageSearch, searchStart, searchErr)
			if searchErr != nil {
				return d, searchErr
			}
			return d, nil
		})
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		allResults = append(allResults, res)
	}

	merged := rrfMergeMultiple(allResults, topK)
	merged = p.maybeDedup(merged)
	merged = p.maybeAttachParentContent(ctx, merged)
	merged.QueryText = question
	merged = p.RedactRetrievalResult(merged)

	// @sk-task reranker-cross-encoder#T3.2: QueryMulti integration — BatchReranker type-assert + fallback (AC-009)
	if p.reranker != nil {
		merged, err = p.maybeRerankBatch(ctx, queries, merged)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
	}

	return merged, nil
}

// Query выполняет поиск по вопросу и возвращает RetrievalResult.
//
// @sk-task arch-issues#T2.1: PII redaction в Query (AC-001, AC-002)
// @sk-task hierarchical-indices#T3.3: parent context attach in Query (AC-002)
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
// @sk-task arch-issues#T3.1: closed guard в Query (AC-008)
func (p *Pipeline) Query(ctx context.Context, question string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := p.checkClosed(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	question = p.redact(question)
	if question == "" {
		return domain.RetrievalResult{}, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "Query", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "Query", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "Query", domain.StageData{Query: question, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "Query", domain.HookStageSearch)
		var searchErr error
		result, searchErr = p.store.Search(ctx, d.Embedding, topK)
		p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result = p.maybeAttachParentContent(ctx, result)
	result.QueryText = question
	result = p.RedactRetrievalResult(result)

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// @sk-task arch-issues#T2.1: PII redaction в QueryWithParentIDs (AC-001)
//
// QueryWithParentIDs выполняет поиск по вопросу с фильтром по ParentIDs.
//
// Если parentIDs пустой — эквивалентно Query.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
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
	question = p.redact(question)
	if question == "" {
		return domain.RetrievalResult{}, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "Query", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "Query", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "Query", domain.StageData{Query: question, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "Query", domain.HookStageSearch)
		var searchErr error
		result, searchErr = vs.SearchWithFilter(ctx, d.Embedding, topK, domain.ParentIDFilter{ParentIDs: parentIDs})
		p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question
	result = p.RedactRetrievalResult(result)

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// @sk-task arch-issues#T2.1: PII redaction в QueryWithMetadataFilter (AC-001)
//
// QueryWithMetadataFilter выполняет поиск по вопросу с фильтром по метаданным документа.
//
// Если filter.Fields пустой — эквивалентно Query.
// Если store не реализует VectorStoreWithFilters — возвращает ErrFiltersNotSupported.
//
// @ds-task T3.1: Добавить QueryWithMetadataFilter в application.Pipeline (RQ-005, AC-003, DEC-003)
// @sk-task hierarchical-indices#T3.3: parent context attach in QueryWithMetadataFilter (AC-002)
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
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
	question = p.redact(question)
	if question == "" {
		return domain.RetrievalResult{}, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "Query", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "Query", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "Query", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "Query", domain.StageData{Query: question, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "Query", domain.HookStageSearch)
		var searchErr error
		result, searchErr = vs.SearchWithMetadataFilter(ctx, d.Embedding, topK, filter)
		p.hookEnd(ctx, "Query", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result = p.maybeAttachParentContent(ctx, result)
	result.QueryText = question
	result = p.RedactRetrievalResult(result)

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}

// @sk-task arch-issues#T2.1: PII redaction в QueryWithQueries (AC-001)
//
// @sk-task query-rewriting#T2.2: QueryWithQueries для pre-generated переформулировок (AC-003)
// @sk-task hierarchical-indices#T3.3: parent context attach in QueryWithQueries (AC-002)
// QueryWithQueries выполняет multi-query retrieval из уже готового списка запросов.
// Каждый запрос эмбеддится и ищется, результаты объединяются через RRF.
func (p *Pipeline) QueryWithQueries(ctx context.Context, queries []string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if len(queries) == 0 {
		return domain.RetrievalResult{}, fmt.Errorf("%w: no queries", domain.ErrEmptyQueryText)
	}

	allResults := make([]domain.RetrievalResult, 0, len(queries))
	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return domain.RetrievalResult{}, err
		}
		q = strings.TrimSpace(q)
		q = p.redact(q)
		if q == "" {
			continue
		}
		var emb []float64
		_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "QueryWithQueries", domain.StageData{Query: q}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
			embedStart := time.Now()
			p.hookStart(ctx, "QueryWithQueries:embed", domain.HookStageEmbed)
			var embedErr error
			emb, embedErr = p.embedder.Embed(ctx, d.Query)
			p.hookEnd(ctx, "QueryWithQueries:embed", domain.HookStageEmbed, embedStart, embedErr)
			if embedErr != nil {
				return d, embedErr
			}
			return d, nil
		})
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("rewriter embed: %w", err)
		}
		var res domain.RetrievalResult
		_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "QueryWithQueries", domain.StageData{Query: q, Embedding: emb}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
			searchStart := time.Now()
			p.hookStart(ctx, "QueryWithQueries:search", domain.HookStageSearch)
			var searchErr error
			res, searchErr = p.store.Search(ctx, d.Embedding, topK)
			p.hookEnd(ctx, "QueryWithQueries:search", domain.HookStageSearch, searchStart, searchErr)
			if searchErr != nil {
				return d, searchErr
			}
			return d, nil
		})
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		allResults = append(allResults, res)
	}

	if len(allResults) == 0 {
		return domain.RetrievalResult{}, fmt.Errorf("%w: all queries were empty", domain.ErrEmptyQueryText)
	}

	merged := rrfMergeMultiple(allResults, topK)
	merged = p.maybeDedup(merged)
	merged = p.maybeAttachParentContent(ctx, merged)
	merged.QueryText = queries[0]
	merged = p.RedactRetrievalResult(merged)

	if p.reranker != nil {
		var err error
		merged, err = p.maybeRerankBatch(ctx, queries, merged)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
	}

	return merged, nil
}

// ErrHybridNotSupported is returned when a hybrid search method is called
// but the underlying VectorStore does not implement HybridSearcher.
var ErrHybridNotSupported = errors.New("vector store does not support hybrid search")

// QueryHybrid выполняет гибридный поиск (BM25 + semantic) по вопросу.
//
// Если store не реализует HybridSearcher — возвращает ErrHybridNotSupported.
//
// @sk-task hierarchical-indices#T3.3: parent context attach in QueryHybrid (AC-002)
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
// @sk-task arch-issues#T2.1: PII redaction в QueryHybrid (AC-001)
func (p *Pipeline) QueryHybrid(ctx context.Context, question string, topK int, config domain.HybridConfig) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	question = p.redact(question)
	if question == "" {
		return domain.RetrievalResult{}, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}

	hs, ok := p.store.(domain.HybridSearcher)
	if !ok {
		return domain.RetrievalResult{}, ErrHybridNotSupported
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "QueryHybrid", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "QueryHybrid", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "QueryHybrid", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "QueryHybrid", domain.StageData{Query: question, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "QueryHybrid", domain.HookStageSearch)
		var searchErr error
		result, searchErr = hs.SearchHybrid(ctx, d.Query, d.Embedding, topK, config)
		p.hookEnd(ctx, "QueryHybrid", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	result = p.maybeDedup(result)
	result = p.maybeAttachParentContent(ctx, result)
	result.QueryText = question
	result = p.RedactRetrievalResult(result)

	result, err = p.maybeRerank(ctx, question, result)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return result, nil
}
