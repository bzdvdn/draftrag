package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// AnswerStream выполняет RAG-цикл с streaming генерацией ответа.
// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
// Retrieval выполняется синхронно перед началом streaming'а.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
// @ds-task T2.3: Реализовать AnswerStream в application Pipeline (AC-001, DEC-003)
// @sk-task middleware-chain#T3.2: userMessage from d.Query (AC-004)
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
		return nil, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return nil, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	// Type assertion для проверки поддержки streaming
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "AnswerStream", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "AnswerStream", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "AnswerStream", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "AnswerStream", domain.StageData{Query: question, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "AnswerStream", domain.HookStageSearch)
		var searchErr error
		result, searchErr = p.store.Search(ctx, d.Embedding, candidateTopK)
		p.hookEnd(ctx, "AnswerStream", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
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

	var tokenChan <-chan string
	genStart := time.Now()
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageGenerate, "AnswerStream", domain.StageData{Query: question, Chunks: result.Chunks}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
		userMessage := buildUserMessageV1(result, d.Query, p.maxContextChars, p.maxContextChunks)
		var genErr error
		tokenChan, genErr = streamingLLM.GenerateStream(ctx, p.systemPrompt, userMessage)
		if genErr != nil {
			p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
			return d, genErr
		}
		return d, nil
	})
	if err != nil {
		return nil, err
	}

	// Обёртка для отслеживания завершения генерации + middleware channel wrapper
	return p.wrapStreamWithHook(ctx, p.wrapStreamWithMiddleware(ctx, tokenChan), genStart), nil
}

// wrapStreamWithHook оборачивает канал токенов для вызова hook по завершении.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T3.3: bounded backpressure — output chan с cap=p.streamBufferSize (DEC-006, RQ-006, AC-010)
// При p.streamBufferSize > 0 выходной канал буферизуется с указанной ёмкостью —
// producer (LLM-стрим) может обгонять consumer (вызывающий код) на cap токенов,
// не блокируясь. При 0 канал unbuffered (backward-compat, OQ-2) — синхронная
// передача токенов.
func (p *Pipeline) wrapStreamWithHook(ctx context.Context, source <-chan string, genStart time.Time) <-chan string {
	output := make(chan string, p.streamBufferSize)

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

// AnswerStreamWithInlineCitations выполняет RAG-цикл с streaming генерацией и inline-цитатами.
// Возвращает канал для чтения текстовых чанков и слайс цитат (заполняется синхронно перед streaming'ом).
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task api-consistency-pass#T2.1: wrapped domain.ErrEmptyQueryText/ErrInvalidQueryTopK в validation (RQ-003, AC-003)
// @ds-task T2.4: Реализовать AnswerStreamWithInlineCitations в application Pipeline (AC-002)
// @sk-task middleware-chain#T-concern: add middleware on embed/search/generate (AC-003)
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
		return nil, domain.RetrievalResult{}, nil, fmt.Errorf("%w: question is empty", domain.ErrEmptyQueryText)
	}
	if topK <= 0 {
		return nil, domain.RetrievalResult{}, nil, fmt.Errorf("%w: topK must be > 0", domain.ErrInvalidQueryTopK)
	}

	// Type assertion для проверки поддержки streaming
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, domain.RetrievalResult{}, nil, ErrStreamingNotSupported
	}

	var embedding []float64
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageEmbed, "AnswerStream", domain.StageData{Query: question}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		embedStart := time.Now()
		p.hookStart(ctx, "AnswerStream", domain.HookStageEmbed)
		var embedErr error
		embedding, embedErr = p.embedder.Embed(ctx, d.Query)
		p.hookEnd(ctx, "AnswerStream", domain.HookStageEmbed, embedStart, embedErr)
		if embedErr != nil {
			return d, embedErr
		}
		return d, nil
	})
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}

	candidateTopK := topK
	if p.mmrEnabled && p.mmrCandidatePool > candidateTopK {
		candidateTopK = p.mmrCandidatePool
	}

	var result domain.RetrievalResult
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageSearch, "AnswerStream", domain.StageData{Query: question, Embedding: embedding}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		searchStart := time.Now()
		p.hookStart(ctx, "AnswerStream", domain.HookStageSearch)
		var searchErr error
		result, searchErr = p.store.Search(ctx, d.Embedding, candidateTopK)
		p.hookEnd(ctx, "AnswerStream", domain.HookStageSearch, searchStart, searchErr)
		if searchErr != nil {
			return d, searchErr
		}
		return d, nil
	})
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
	var tokenChan <-chan string
	var citations []domain.InlineCitation
	genStart := time.Now()
	_, err = p.execWithStageMiddleware(ctx, domain.HookStageGenerate, "AnswerStream", domain.StageData{Query: question, Chunks: result.Chunks}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
		userMessage, ic := buildUserMessageV1InlineCitations(result, d.Query, p.maxContextChars, p.maxContextChunks)
		citations = ic
		var genErr error
		tokenChan, genErr = streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
		if genErr != nil {
			p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
			return d, genErr
		}
		return d, nil
	})
	if err != nil {
		return nil, domain.RetrievalResult{}, citations, err
	}

	return p.wrapStreamWithHook(ctx, p.wrapStreamWithMiddleware(ctx, tokenChan), genStart), result, citations, nil
}

// @sk-task middleware-chain#T2.4: wrapStreamWithMiddleware (AC-003, AC-004)
//
// wrapStreamWithMiddleware обёртывает канал токенов, вызывая post-generate middleware
// после закрытия source-канала. Middleware может прочитать полный ответ из канала,
// модифицировать его и отправить в выходной канал.
func (p *Pipeline) wrapStreamWithMiddleware(ctx context.Context, source <-chan string) <-chan string {
	if len(p.middleware) == 0 {
		return source
	}
	output := make(chan string, p.streamBufferSize)
	go func() {
		defer close(output)
		var sb strings.Builder
		for token := range source {
			sb.WriteString(token)
			output <- token
		}
		data := domain.StageData{
			Stage:  domain.HookStageGenerate,
			Answer: sb.String(),
		}
		_, _ = runMiddleware(ctx, p.middleware, data, func(_ context.Context, d domain.StageData) (domain.StageData, error) {
			return d, nil
		})
	}()
	return output
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task middleware-chain#T-concern: add middleware on generate (AC-003)
// streamFromResult выполняет streaming генерацию из готового RetrievalResult.
func (p *Pipeline) streamFromResult(ctx context.Context, question string, result domain.RetrievalResult) (<-chan string, error) {
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	systemPrompt := p.systemPrompt

	var tokenChan <-chan string
	genStart := time.Now()
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageGenerate, "AnswerStream", domain.StageData{Query: question, Chunks: result.Chunks}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
		userMessage := buildUserMessageV1(result, d.Query, p.maxContextChars, p.maxContextChunks)
		var genErr error
		tokenChan, genErr = streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
		if genErr != nil {
			p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
			return d, genErr
		}
		return d, nil
	})
	if err != nil {
		return nil, err
	}

	return p.wrapStreamWithHook(ctx, p.wrapStreamWithMiddleware(ctx, tokenChan), genStart), nil
}

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
// @sk-task middleware-chain#T-concern: add middleware on generate (AC-003)
// streamInlineFromResult выполняет streaming генерацию с inline citations из готового RetrievalResult.
func (p *Pipeline) streamInlineFromResult(ctx context.Context, question string, result domain.RetrievalResult) (<-chan string, []domain.InlineCitation, error) {
	streamingLLM, ok := p.llm.(domain.StreamingLLMProvider)
	if !ok {
		return nil, nil, ErrStreamingNotSupported
	}

	systemPrompt := p.systemPrompt
	var tokenChan <-chan string
	var citations []domain.InlineCitation
	genStart := time.Now()
	_, err := p.execWithStageMiddleware(ctx, domain.HookStageGenerate, "AnswerStream", domain.StageData{Query: question, Chunks: result.Chunks}, func(ctx context.Context, d domain.StageData) (domain.StageData, error) {
		p.hookStart(ctx, "AnswerStream", domain.HookStageGenerate)
		userMessage, ic := buildUserMessageV1InlineCitations(result, d.Query, p.maxContextChars, p.maxContextChunks)
		citations = ic
		var genErr error
		tokenChan, genErr = streamingLLM.GenerateStream(ctx, systemPrompt, userMessage)
		if genErr != nil {
			p.hookEnd(ctx, "AnswerStream", domain.HookStageGenerate, genStart, genErr)
			return d, genErr
		}
		return d, nil
	})
	if err != nil {
		return nil, citations, err
	}

	return p.wrapStreamWithHook(ctx, p.wrapStreamWithMiddleware(ctx, tokenChan), genStart), citations, nil
}

// @sk-task query-rewriting#T2.2: AnswerWithQueriesStream (AC-003)
// AnswerWithQueriesStream выполняет multi-query retrieval из готового списка запросов и streaming генерацию.
func (p *Pipeline) AnswerWithQueriesStream(ctx context.Context, originalQuery string, queries []string, topK int) (<-chan string, error) {
	result, err := p.QueryWithQueries(ctx, queries, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, originalQuery, result)
}

// @sk-task query-rewriting#T2.2: AnswerWithQueriesStreamWithSources (AC-003)
// AnswerWithQueriesStreamWithSources выполняет multi-query retrieval из готового списка запросов,
// streaming генерацию и возвращает источники.
func (p *Pipeline) AnswerWithQueriesStreamWithSources(ctx context.Context, originalQuery string, queries []string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithQueries(ctx, queries, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, originalQuery, result)
	return tokenChan, result, err
}

// @sk-task query-rewriting#T2.2: AnswerWithQueriesStreamWithInlineCitations (AC-003)
// AnswerWithQueriesStreamWithInlineCitations выполняет multi-query retrieval из готового списка запросов,
// streaming генерацию с inline-цитатами.
func (p *Pipeline) AnswerWithQueriesStreamWithInlineCitations(ctx context.Context, originalQuery string, queries []string, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithQueries(ctx, queries, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, originalQuery, result)
	return tokenChan, result, citations, err
}

// AnswerHyDEStream выполняет HyDE retrieval и streaming генерацию.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEStream(ctx context.Context, question string, topK int) (<-chan string, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerMultiStream выполняет MultiQuery retrieval и streaming генерацию.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiStream(ctx context.Context, question string, n, topK int) (<-chan string, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerHybridStream выполняет Hybrid retrieval и streaming генерацию.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridStream(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerStreamWithParentIDs выполняет retrieval с фильтром по ParentIDs и streaming генерацию.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithParentIDs(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerStreamWithMetadataFilter выполняет retrieval с фильтром по метаданным и streaming генерацию.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithMetadataFilter(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// AnswerStreamWithSources выполняет RAG-цикл с streaming и возвращает источники.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithSources(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.Query(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerHyDEStreamWithSources выполняет HyDE retrieval с streaming и возвращает источники.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEStreamWithSources(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerMultiStreamWithSources выполняет MultiQuery retrieval с streaming и возвращает источники.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiStreamWithSources(ctx context.Context, question string, n, topK int) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerHybridStreamWithSources выполняет Hybrid retrieval с streaming и возвращает источники.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridStreamWithSources(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerStreamWithParentIDsWithSources выполняет retrieval с фильтром по ParentIDs с streaming и возвращает источники.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithParentIDsWithSources(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerStreamWithMetadataFilterWithSources выполняет retrieval с фильтром по метаданным с streaming и возвращает источники.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithMetadataFilterWithSources(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// AnswerHyDEStreamWithInlineCitations выполняет HyDE retrieval с streaming и inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHyDEStreamWithInlineCitations(ctx context.Context, question string, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHyDE(ctx, question, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerMultiStreamWithInlineCitations выполняет MultiQuery retrieval с streaming и inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerMultiStreamWithInlineCitations(ctx context.Context, question string, n, topK int) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryMulti(ctx, question, n, topK)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerHybridStreamWithInlineCitations выполняет Hybrid retrieval с streaming и inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerHybridStreamWithInlineCitations(ctx context.Context, question string, topK int, cfg domain.HybridConfig) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryHybrid(ctx, question, topK, cfg)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerStreamWithParentIDsWithInlineCitations выполняет retrieval с фильтром по ParentIDs с streaming и inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithParentIDsWithInlineCitations(ctx context.Context, question string, topK int, parentIDs []string) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithParentIDs(ctx, question, topK, parentIDs)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// AnswerStreamWithMetadataFilterWithInlineCitations выполняет retrieval с фильтром по метаданным с streaming и inline-цитатами.
//
// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func (p *Pipeline) AnswerStreamWithMetadataFilterWithInlineCitations(ctx context.Context, question string, topK int, filter domain.MetadataFilter) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QueryWithMetadataFilter(ctx, question, topK, filter)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}
