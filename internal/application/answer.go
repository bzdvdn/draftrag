package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) generateAnswer(ctx context.Context, question string, result domain.RetrievalResult) (string, error) {
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)
	genStart := time.Now()
	p.hookStart(ctx, "Answer:generate", domain.HookStageGenerate)
	answer, err := p.llm.Generate(ctx, p.systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer:generate", domain.HookStageGenerate, genStart, err)
	return answer, err
}

// Answer выполняет полный RAG-цикл: retrieval (Embed+Search) → prompt → LLM.Generate.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
func (p *Pipeline) Answer(ctx context.Context, question string, topK int) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return "", fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", err
	}
	result = p.maybeDedup(result)

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, err := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, err)
	return answer, err
}

// AnswerWithParentIDs выполняет retrieval с фильтром по ParentIDs и генерирует ответ.
//
// Если parentIDs пустой — эквивалентно Answer.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (string, error) {
	answer, _, err := p.AnswerWithCitationsWithParentIDs(ctx, question, topK, parentIDs)
	return answer, err
}

// AnswerWithMetadataFilter выполняет retrieval с фильтром по метаданным и генерирует ответ.
//
// Если filter.Fields пустой — эквивалентно Answer.
// Если store не реализует VectorStoreWithFilters — возвращает ErrFiltersNotSupported.
//
// @ds-task T3.1: Добавить AnswerWithMetadataFilter в application.Pipeline (RQ-006, AC-003, DEC-003)
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
func (p *Pipeline) AnswerWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (string, error) {
	if len(filter.Fields) == 0 {
		return p.Answer(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return "", fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return "", ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := vs.SearchWithMetadataFilter(ctx, embedding, candidateTopK, filter)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", err
	}
	result = p.maybeDedup(result)

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	return answer, genErr
}

// AnswerHybrid выполняет гибридный поиск (BM25 + semantic) и генерирует ответ.
//
// Если store не реализует HybridSearcher — возвращает ErrHybridNotSupported.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
func (p *Pipeline) AnswerHybrid(ctx context.Context, question string, topK int, config domain.HybridConfig) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return "", fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}
	if err := config.Validate(); err != nil {
		return "", err
	}

	result, err := p.QueryHybrid(ctx, question, topK, config)
	if err != nil {
		return "", err
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerHybrid", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "AnswerHybrid", domain.HookStageGenerate, genStart, genErr)
	return answer, genErr
}

// AnswerWithInlineCitations выполняет полный RAG-цикл и возвращает ответ с inline-цитатами `[n]`,
// а также retrieval evidence и детерминированный маппинг `n -> chunk`.
//
// Если retrieval уже выполнен, а Generate вернул ошибку, метод возвращает retrieval результат (partial),
// массив citations и ошибку.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
func (p *Pipeline) AnswerWithInlineCitations(
	ctx context.Context,
	question string,
	topK int,
) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", domain.RetrievalResult{}, nil, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return "", domain.RetrievalResult{}, nil, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", domain.RetrievalResult{}, nil, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, citations, genErr
	}
	return answer, result, citations, nil
}

// AnswerWithCitations выполняет полный RAG-цикл и возвращает retrieval evidence вместе с ответом.
//
// Если retrieval уже выполнен, а Generate вернул ошибку, метод возвращает retrieval результат (partial)
// и ошибку, чтобы упростить диагностику и отображение источников.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
func (p *Pipeline) AnswerWithCitations(ctx context.Context, question string, topK int) (string, domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", domain.RetrievalResult{}, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return "", domain.RetrievalResult{}, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", domain.RetrievalResult{}, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// AnswerWithCitationsWithParentIDs выполняет RAG-цикл с фильтром по ParentIDs и возвращает retrieval evidence.
//
// Если parentIDs пустой — эквивалентно AnswerWithCitations.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
func (p *Pipeline) AnswerWithCitationsWithParentIDs(
	ctx context.Context,
	question string,
	topK int,
	parentIDs []string,
) (string, domain.RetrievalResult, error) {
	if len(parentIDs) == 0 {
		return p.AnswerWithCitations(ctx, question, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.RetrievalResult{}, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", domain.RetrievalResult{}, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return "", domain.RetrievalResult{}, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	vs, ok := p.store.(domain.VectorStoreWithFilters)
	if !ok {
		return "", domain.RetrievalResult{}, ErrFiltersNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "Answer", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageSearch)
	result, err := vs.SearchWithFilter(ctx, embedding, candidateTopK, domain.ParentIDFilter{ParentIDs: parentIDs})
	p.hookEnd(ctx, "Answer", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return "", domain.RetrievalResult{}, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// generateCitedFromResult генерирует ответ с цитатами из готового RetrievalResult.
// Helper для унификации логики генерации в Answer*WithCitations методах.
func (p *Pipeline) generateCitedFromResult(
	ctx context.Context,
	question string,
	result domain.RetrievalResult,
) (string, domain.RetrievalResult, error) {
	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// generateInlineCitedFromResult генерирует ответ с inline-цитатами из готового RetrievalResult.
func (p *Pipeline) generateInlineCitedFromResult(
	ctx context.Context,
	question string,
	result domain.RetrievalResult,
) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "Answer", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "Answer", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, citations, genErr
	}
	return answer, result, citations, nil
}

// AnswerHyDEWithCitations выполняет HyDE retrieval и генерирует ответ с цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEWithCitations(ctx context.Context, question string, topK int) (string, domain.RetrievalResult, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerMultiWithCitations выполняет MultiQuery retrieval и генерирует ответ с цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiWithCitations(ctx context.Context, question string, n, topK int) (string, domain.RetrievalResult, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerHybridWithCitations выполняет Hybrid retrieval и генерирует ответ с цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridWithCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (string, domain.RetrievalResult, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerWithCitationsWithMetadataFilter выполняет retrieval с фильтром по метаданным и генерирует ответ с цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerWithCitationsWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (string, domain.RetrievalResult, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}
	return p.generateCitedFromResult(ctx, question, result)
}

// AnswerHyDEWithInlineCitations выполняет HyDE retrieval и генерирует ответ с inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEWithInlineCitations(ctx context.Context, question string, topK int) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerMultiWithInlineCitations выполняет MultiQuery retrieval и генерирует ответ с inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiWithInlineCitations(ctx context.Context, question string, n, topK int) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerHybridWithInlineCitations выполняет Hybrid retrieval и генерирует ответ с inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridWithInlineCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerWithInlineCitationsWithMetadataFilter выполняет retrieval с фильтром по метаданным и генерирует ответ с inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerWithInlineCitationsWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerWithInlineCitationsWithParentIDs выполняет retrieval с фильтром по ParentIDs и генерирует ответ с inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerWithInlineCitationsWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}
	return p.generateInlineCitedFromResult(ctx, question, result)
}

// AnswerHyDE генерирует ответ, используя HyDE для retrieval.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDE(ctx context.Context, question string, topK int) (string, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return "", err
	}
	return p.generateAnswer(ctx, question, result)
}

// AnswerMulti генерирует ответ используя multi-query retrieval.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMulti(ctx context.Context, question string, n, topK int) (string, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return "", err
	}
	return p.generateAnswer(ctx, question, result)
}
