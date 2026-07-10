package draftrag

import (
	"context"
	"strings"
)

// SearchBuilder накапливает параметры поиска и позволяет выполнить поисковый или генеративный запрос
// через цепочку вызовов. Создаётся через Pipeline.Search.
//
// Пример:
//
//	// Поиск
//	result, err := pipeline.Search("вопрос").TopK(5).Retrieve(ctx)
//
//	// Ответ с фильтром
//	answer, err := pipeline.Search("вопрос").TopK(5).Filter(f).Answer(ctx)
//
//	// Inline цитаты
//	answer, sources, cits, err := pipeline.Search("вопрос").TopK(5).InlineCite(ctx)
//
//	// Streaming
//	tokens, err := pipeline.Search("вопрос").TopK(5).Stream(ctx)
//
//	// С кастомным rewriter
//	result, err := pipeline.Search("вопрос").Rewriter(myRewriter).Retrieve(ctx)
//
//	// С multi-turn историей
//	history := QueryHistory{Entries: []Message{{Role: "user", Content: "..."}}}
//	result, err := pipeline.Search("вопрос").Rewriter(rw).History(history).Retrieve(ctx)
//
// @sk-task query-rewriting#T2.1: добавлены поля rewriter и history (AC-002)
type SearchBuilder struct {
	pipeline   *Pipeline
	question   string
	topK       int
	parentIDs  []string
	filter     MetadataFilter
	hybrid     *HybridConfig
	hyDE       bool
	multiQuery int // 0 = disabled
	rewriter   QueryRewriter
	history    QueryHistory
}

// Search создаёт SearchBuilder для заданного вопроса.
// По умолчанию TopK берётся из PipelineOptions.DefaultTopK (или 5).
// Pipeline-level QueryRewriter передаётся в SearchBuilder, но может быть
// переопределён через Rewriter().
func (p *Pipeline) Search(question string) *SearchBuilder {
	return &SearchBuilder{
		pipeline: p,
		question: question,
		topK:     p.defaultTop,
		rewriter: p.queryRewriter,
	}
}

// TopK задаёт количество возвращаемых чанков.
func (b *SearchBuilder) TopK(n int) *SearchBuilder {
	b.topK = n
	return b
}

// Filter задаёт AND-фильтр по полям метаданных документа.
// Совместим только с хранилищами, реализующими VectorStoreWithFilters.
func (b *SearchBuilder) Filter(f MetadataFilter) *SearchBuilder {
	b.filter = f
	return b
}

// ParentIDs ограничивает поиск чанками из указанных документов.
func (b *SearchBuilder) ParentIDs(ids ...string) *SearchBuilder {
	b.parentIDs = ids
	return b
}

// Hybrid включает гибридный поиск (BM25 + semantic) с заданной конфигурацией.
// Совместим только с хранилищами, реализующими HybridSearcher.
func (b *SearchBuilder) Hybrid(cfg HybridConfig) *SearchBuilder {
	b.hybrid = &cfg
	return b
}

// HyDE включает Hypothetical Document Embeddings.
// LLM генерирует гипотетический ответ на вопрос, который используется для embedding-поиска.
// Улучшает recall для сложных вопросов. Не совместим с Hybrid.
func (b *SearchBuilder) HyDE() *SearchBuilder {
	b.hyDE = true
	return b
}

// MultiQuery включает multi-query retrieval: LLM генерирует n перефразировок вопроса,
// результаты каждого поиска объединяются через Reciprocal Rank Fusion.
// n=0 → использует дефолт (3).
func (b *SearchBuilder) MultiQuery(n int) *SearchBuilder {
	b.multiQuery = n
	if b.multiQuery == 0 {
		b.multiQuery = 3
	}
	return b
}

// @sk-task query-rewriting#T2.1: Rewriter — per-request override (AC-002)
// Rewriter задаёт per-request rewriter, который имеет приоритет над pipeline-level.
func (b *SearchBuilder) Rewriter(r QueryRewriter) *SearchBuilder {
	b.rewriter = r
	return b
}

// @sk-task query-rewriting#T2.1: History — multi-turn контекст (AC-004)
// History задаёт историю диалога для multi-turn переписывания.
func (b *SearchBuilder) History(h QueryHistory) *SearchBuilder {
	b.history = h
	return b
}

func (b *SearchBuilder) validate() (string, error) {
	q := strings.TrimSpace(b.question)
	if q == "" {
		return "", ErrEmptyQuery
	}
	if b.topK <= 0 {
		return "", ErrInvalidTopK
	}
	return q, nil
}

// Retrieve выполняет поиск и возвращает RetrievalResult.
// Routing (HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic) делегирован в runRetrieve.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
// @sk-task pii-guardrails#T2.3: PII redaction в SearchBuilder.Retrieve (AC-002, RQ-002)
func (b *SearchBuilder) Retrieve(ctx context.Context) (RetrievalResult, error) {
	if err := checkCtx(ctx); err != nil {
		return RetrievalResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return RetrievalResult{}, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return RetrievalResult{}, err
	}
	res, err := retrieveRouter.execute(ctx, q, b.topK, r, b)
	if err != nil {
		return RetrievalResult{}, err
	}
	b.pipeline.redactRetrievalResult(&res.Result)
	return res.Result, nil
}

// Answer выполняет RAG-ответ и возвращает строку.
// Routing делегирован в runAnswer.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
func (b *SearchBuilder) Answer(ctx context.Context) (string, error) {
	if err := checkCtx(ctx); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return "", err
	}
	res, err := answerRouter.execute(ctx, q, b.topK, r, b)
	return res.Text, err
}

// Cite выполняет RAG-ответ и возвращает ответ + источники (чанки со score).
// Routing делегирован в runCite.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
// @sk-task pii-guardrails#T2.3: PII redaction в Cite (RQ-002)
func (b *SearchBuilder) Cite(ctx context.Context) (string, RetrievalResult, error) {
	if err := checkCtx(ctx); err != nil {
		return "", RetrievalResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return "", RetrievalResult{}, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return "", RetrievalResult{}, err
	}
	res, err := citeRouter.execute(ctx, q, b.topK, r, b)
	if err != nil {
		return "", RetrievalResult{}, err
	}
	b.pipeline.redactRetrievalResult(&res.Sources)
	return res.Text, res.Sources, nil
}

// InlineCite выполняет RAG-ответ с inline-цитатами `[n]`.
// LLM расставляет ссылки в тексте; citations содержит только использованные источники.
// Routing делегирован в runInlineCite.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
// @sk-task pii-guardrails#T2.3: PII redaction в InlineCite (RQ-002)
func (b *SearchBuilder) InlineCite(ctx context.Context) (string, RetrievalResult, []InlineCitation, error) {
	if err := checkCtx(ctx); err != nil {
		return "", RetrievalResult{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return "", RetrievalResult{}, nil, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return "", RetrievalResult{}, nil, err
	}
	res, err := inlineCiteRouter.execute(ctx, q, b.topK, r, b)
	if err != nil {
		return "", RetrievalResult{}, nil, err
	}
	b.pipeline.redactRetrievalResult(&res.Sources)
	return res.Text, res.Sources, res.Citations, nil
}

// Stream выполняет RAG-ответ через streaming (токен за токеном).
// Если LLM не поддерживает streaming — возвращает ErrStreamingNotSupported.
// Routing делегирован в runStream.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
func (b *SearchBuilder) Stream(ctx context.Context) (<-chan string, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return nil, err
	}
	res, err := streamRouter.execute(ctx, q, b.topK, r, b)
	return res.Ch, err
}

// StreamSources выполняет RAG-ответ через streaming с синхронно готовым списком источников.
// sources готов сразу (поиск синхронный); токены — асинхронно через канал.
// Routing делегирован в runStreamSources.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
func (b *SearchBuilder) StreamSources(ctx context.Context) (<-chan string, RetrievalResult, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, RetrievalResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return nil, RetrievalResult{}, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return nil, RetrievalResult{}, err
	}
	res, err := streamSourcesRouter.execute(ctx, q, b.topK, r, b)
	return res.Ch, res.Sources, err
}

// StreamCite выполняет RAG-ответ через streaming с inline-цитатами.
// sources и citations готовы сразу (поиск синхронный); токены — асинхронно.
// Routing делегирован в runStreamInline.
//
// @sk-task arch-generics#T2.2: nil context guard + routing delegate через router.execute (AC-001, AC-002)
func (b *SearchBuilder) StreamCite(ctx context.Context) (<-chan string, RetrievalResult, []InlineCitation, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, RetrievalResult{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, RetrievalResult{}, nil, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return nil, RetrievalResult{}, nil, err
	}
	res, err := streamCiteRouter.execute(ctx, q, b.topK, r, b)
	return res.Ch, res.Sources, res.Citations, err
}
