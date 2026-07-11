package application

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// errResult объединяет результат одного под-запроса.
type errResult struct {
	query  string
	result domain.RetrievalResult
	err    error
}

// @sk-task sub-query-decomposition#T1.3: QuerySubDecompose (AC-002, AC-004, AC-007)
// QuerySubDecompose разбивает запрос на под-вопросы через decomposer,
// выполняет параллельный retrieval по каждому и объединяет результаты.
//
// Семантика:
// - Декомпозиция: decomposer.Decompose → список под-вопросов.
// - Если под-вопросов <= 1, выполняется обычный Query с исходным запросом.
// - Параллельный embed+search для каждого под-вопроса (errgroup-style).
// - Merge: дедупликация по Chunk.ID, max score per chunk, сортировка по score desc.
// - Graceful degradation: ошибка декомпозиции → single-query по исходному запросу.
func (p *Pipeline) QuerySubDecompose(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if decomposer == nil {
		return domain.RetrievalResult{}, ErrSubDecomposeNotSupported
	}

	subQueryStart := time.Now()
	p.hookStart(ctx, "QuerySubDecompose:decompose", domain.HookStageGenerate)
	subQueries, err := decomposer.Decompose(ctx, question)
	p.hookEnd(ctx, "QuerySubDecompose:decompose", domain.HookStageGenerate, subQueryStart, err)
	if err != nil || len(subQueries) == 0 {
		p.hookStart(ctx, "QuerySubDecompose:single-fallback", domain.HookStageSearch)
		result, err := p.Query(ctx, question, topK)
		p.hookEnd(ctx, "QuerySubDecompose:single-fallback", domain.HookStageSearch, subQueryStart, err)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		return result, nil
	}

	if len(subQueries) == 1 {
		result, err := p.Query(ctx, subQueries[0], topK)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		result.QueryText = question
		return result, nil
	}

	// Параллельный embed+search по каждому под-вопросу.
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type subTask struct {
		query string
	}
	tasks := make([]subTask, len(subQueries))
	for i, q := range subQueries {
		tasks[i] = subTask{query: q}
	}

	resultCh := make(chan errResult, len(tasks))
	var wg sync.WaitGroup

	for _, t := range tasks {
		wg.Add(1)
		go func(q string) {
			defer wg.Done()
			select {
			case <-cancelCtx.Done():
				resultCh <- errResult{query: q, err: cancelCtx.Err()}
				return
			default:
			}

			embedStart := time.Now()
			p.hookStart(cancelCtx, "QuerySubDecompose:embed", domain.HookStageEmbed)
			emb, err := p.embedder.Embed(cancelCtx, q)
			p.hookEnd(cancelCtx, "QuerySubDecompose:embed", domain.HookStageEmbed, embedStart, err)
			if err != nil {
				cancel()
				resultCh <- errResult{query: q, err: fmt.Errorf("sub-decompose embed: %w", err)}
				return
			}

			searchStart := time.Now()
			p.hookStart(cancelCtx, "QuerySubDecompose:search", domain.HookStageSearch)
			res, err := p.store.Search(cancelCtx, emb, topK)
			p.hookEnd(cancelCtx, "QuerySubDecompose:search", domain.HookStageSearch, searchStart, err)
			if err != nil {
				cancel()
				resultCh <- errResult{query: q, err: err}
				return
			}

			resultCh <- errResult{query: q, result: res}
		}(t.query)
	}

	wg.Wait()
	close(resultCh)

	allResults := make([]domain.RetrievalResult, 0, len(tasks))
	for r := range resultCh {
		if r.err != nil {
			return domain.RetrievalResult{}, r.err
		}
		allResults = append(allResults, r.result)
	}

	merged := mergeSubResults(allResults, topK)
	merged = p.maybeDedup(merged)
	merged.QueryText = question

	if p.reranker != nil {
		var err error
		merged, err = p.maybeRerankBatch(ctx, subQueries, merged)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
	}

	return merged, nil
}

// @sk-task sub-query-decomposition#T2.2: AnswerSubDecompose (AC-008)
// AnswerSubDecompose выполняет sub-query decomposition + retrieval + answer.
func (p *Pipeline) AnswerSubDecompose(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (string, error) {
	result, err := p.QuerySubDecompose(ctx, question, topK, decomposer)
	if err != nil {
		return "", err
	}
	return p.generateAnswer(ctx, question, result)
}

// @sk-task sub-query-decomposition#T2.2: AnswerSubDecomposeWithCitations (AC-008, AC-009)
// AnswerSubDecomposeWithCitations выполняет sub-query decomposition + retrieval + answer с источниками.
func (p *Pipeline) AnswerSubDecomposeWithCitations(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (string, domain.RetrievalResult, error) {
	result, err := p.QuerySubDecompose(ctx, question, topK, decomposer)
	if err != nil {
		return "", domain.RetrievalResult{}, err
	}

	systemPrompt := p.systemPrompt
	userMessage := buildUserMessageV1(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerSubDecompose", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "AnswerSubDecompose", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, genErr
	}
	return answer, result, nil
}

// @sk-task sub-query-decomposition#T2.2: AnswerSubDecomposeWithInlineCitations (AC-008, AC-009)
// AnswerSubDecomposeWithInlineCitations выполняет sub-query decomposition + retrieval + answer с inline-цитатами.
func (p *Pipeline) AnswerSubDecomposeWithInlineCitations(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QuerySubDecompose(ctx, question, topK, decomposer)
	if err != nil {
		return "", domain.RetrievalResult{}, nil, err
	}

	systemPrompt := p.systemPrompt
	userMessage, citations := buildUserMessageV1InlineCitations(result, question, p.maxContextChars, p.maxContextChunks)

	genStart := time.Now()
	p.hookStart(ctx, "AnswerSubDecompose", domain.HookStageGenerate)
	answer, genErr := p.llm.Generate(ctx, systemPrompt, userMessage)
	p.hookEnd(ctx, "AnswerSubDecompose", domain.HookStageGenerate, genStart, genErr)
	if genErr != nil {
		return "", result, citations, genErr
	}
	return answer, result, citations, nil
}

// @sk-task sub-query-decomposition#T2.2: AnswerSubDecomposeStream (AC-009)
// AnswerSubDecomposeStream выполняет sub-query decomposition + retrieval + streaming answer.
func (p *Pipeline) AnswerSubDecomposeStream(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (<-chan string, error) {
	result, err := p.QuerySubDecompose(ctx, question, topK, decomposer)
	if err != nil {
		return nil, err
	}
	return p.streamFromResult(ctx, question, result)
}

// @sk-task sub-query-decomposition#T2.2: AnswerSubDecomposeStreamWithSources (AC-009)
// AnswerSubDecomposeStreamWithSources выполняет sub-query decomposition + retrieval + streaming answer с источниками.
func (p *Pipeline) AnswerSubDecomposeStreamWithSources(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (<-chan string, domain.RetrievalResult, error) {
	result, err := p.QuerySubDecompose(ctx, question, topK, decomposer)
	if err != nil {
		return nil, domain.RetrievalResult{}, err
	}
	tokenChan, err := p.streamFromResult(ctx, question, result)
	return tokenChan, result, err
}

// @sk-task sub-query-decomposition#T2.2: AnswerSubDecomposeStreamWithInlineCitations (AC-009)
// AnswerSubDecomposeStreamWithInlineCitations выполняет sub-query decomposition + retrieval + streaming answer с inline-цитатами.
func (p *Pipeline) AnswerSubDecomposeStreamWithInlineCitations(ctx context.Context, question string, topK int, decomposer domain.QueryDecomposer) (<-chan string, domain.RetrievalResult, []domain.InlineCitation, error) {
	result, err := p.QuerySubDecompose(ctx, question, topK, decomposer)
	if err != nil {
		return nil, domain.RetrievalResult{}, nil, err
	}
	tokenChan, citations, err := p.streamInlineFromResult(ctx, question, result)
	return tokenChan, result, citations, err
}

// mergeSubResults объединяет результаты retrieval из нескольких под-вопросов.
// Дедупликация по Chunk.ID, каждый чанк получает максимальный score среди всех источников.
// Результат сортируется по score desc.
func mergeSubResults(results []domain.RetrievalResult, topK int) domain.RetrievalResult {
	type entry struct {
		chunk domain.RetrievedChunk
		score float64
	}

	best := make(map[string]*entry)
	for _, res := range results {
		for _, rc := range res.Chunks {
			id := rc.Chunk.ID
			if existing, ok := best[id]; ok {
				if rc.Score > existing.score {
					existing.score = rc.Score
					existing.chunk = rc
				}
			} else {
				best[id] = &entry{chunk: rc, score: rc.Score}
			}
		}
	}

	out := make([]domain.RetrievedChunk, 0, len(best))
	for _, e := range best {
		out = append(out, e.chunk)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})

	if len(out) > topK {
		out = out[:topK]
	}

	return domain.RetrievalResult{Chunks: out}
}
