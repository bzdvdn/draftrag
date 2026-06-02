package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerStream выполняет RAG-цикл с streaming генерацией ответа.
// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
// Retrieval выполняется синхронно перед началом streaming'а.
//
// @ds-task T2.3: Реализовать AnswerStream в application Pipeline (AC-001, DEC-003)
func (p *Pipeline) AnswerStream(
	ctx context.Context,
	question string,
	topK int,
) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("question is empty")
	}
	if topK <= 0 {
		return nil, errors.New("topK must be > 0")
	}

	// Type assertion для проверки поддержки streaming
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return nil, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return nil, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, genErr
	}

	// Обёртка для отслеживания завершения генерации
	return p.wrapStreamWithHook(ctx, tokenChan, genStart), nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// wrapStreamWithHook оборачивает канал токенов для вызова hook по завершении.
func (p *Pipeline) wrapStreamWithHook(ctx context.Context, source <-chan string, genStart time.Time) <-chan string {
	output := make(chan string)

	go func() {
		defer close(output)
		defer p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, nil)

		for {
			select {
			case <-ctx.Done():
				return
			case token, ok := <-source:
				if !ok {
					return
				}
				select {
				case output <- token:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return output
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerStreamWithInlineCitations выполняет RAG-цикл с streaming генерацией и inline-цитатами.
// Возвращает канал для чтения текстовых чанков и слайс цитат (заполняется синхронно перед streaming'ом).
//
// @ds-task T2.4: Реализовать AnswerStreamWithInlineCitations в application Pipeline (AC-002)
func (p *Pipeline) AnswerStreamWithInlineCitations(
	ctx context.Context,
	question string,
	topK int,
) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return nil, domain.RetrievalResult{}, nil, errors.New("question is empty")
	}
	if topK <= 0 {
		return nil, domain.RetrievalResult{}, nil, errors.New("topK must be > 0")
	}

	// Type assertion для проверки поддержки streaming
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, domain.RetrievalResult{}, nil, ErrStreamingNotSupported
	}

	embedStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageEmbed)
	embedding, err := p.embedder.Embed(ctx, question)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageEmbed, embedStart, err)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	searchStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageSearch)
	result, err := p.store.Search(ctx, embedding, candidateTopK)
	p.hookEnd(ctx, "AnswerStream", domain.HookStageSearch, searchStart, err)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	result = p.maybeDedup(result)
	result.QueryText = question

	if p.mmrEnabled {
		selected, err := selectMMR(ctx, embedding, result.Chunks, topK, p.mmrLambda)
		if err != nil {
			return nil, domain.RetrievalResult{}, nil, err
		}
		result.Chunks = selected
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, result, citations, genErr
	}

	return p.wrapStreamWithHook(ctx, tokenChan, genStart), result, citations, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// streamFromResult выполняет streaming генерацию из готового RetrievalResult.
func (p *Pipeline) streamFromResult(ctx context.Context, question string, result domain.RetrievalResult) (<-chan string, error) {
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, genErr
	}

	return p.wrapStreamWithHook(ctx, tokenChan, genStart), nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// streamInlineFromResult выполняет streaming генерацию с inline citations из готового RetrievalResult.
func (p *Pipeline) streamInlineFromResult(ctx context.Context, question string, result domain.RetrievalResult) (<-chan string, []domain.InlineCitation, error) {
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, nil, ErrStreamingNotSupported
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
	tokenChan, genErr := streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
	if genErr != nil {
		p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
		return nil, citations, genErr
	}

	return p.wrapStreamWithHook(ctx, tokenChan, genStart), citations, nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerHyDEStream выполняет HyDE retrieval и streaming генерацию.
func (p *Pipeline) AnswerHyDEStream(ctx context.Context, question string, topK int) (<-chan string, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerMultiStream выполняет MultiQuery retrieval и streaming генерацию.
func (p *Pipeline) AnswerMultiStream(ctx context.Context, question string, n, topK int) (<-chan string, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerHybridStream выполняет Hybrid retrieval и streaming генерацию.
func (p *Pipeline) AnswerHybridStream(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerStreamWithParentIDs выполняет retrieval с фильтром по ParentIDs и streaming генерацию.
func (p *Pipeline) AnswerStreamWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// AnswerStreamWithMetadataFilter выполняет retrieval с фильтром по метаданным и streaming генерацию.
func (p *Pipeline) AnswerStreamWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithSources(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.Query(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEStreamWithSources(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiStreamWithSources(ctx context.Context, question string, n, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridStreamWithSources(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithParentIDsWithSources(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithMetadataFilterWithSources(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEStreamWithInlineCitations(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiStreamWithInlineCitations(ctx context.Context, question string, n, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridStreamWithInlineCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithParentIDsWithInlineCitations(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithMetadataFilterWithInlineCitations(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}
