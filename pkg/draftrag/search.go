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
type SearchBuilder struct {
	pipeline   *Pipeline
	question   string
	topK       int
	parentIDs  []string
	filter     MetadataFilter
	hybrid     *HybridConfig
	hyDE       bool
	multiQuery int // 0 = disabled
}

// Search создаёт SearchBuilder для заданного вопроса.
// По умолчанию TopK берётся из PipelineOptions.DefaultTopK (или 5).
func (p *Pipeline) Search(question string) *SearchBuilder {
	return &SearchBuilder{
		pipeline: p,
		question: question,
		topK:     p.defaultTop,
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
// @sk-task api-consistency-pass#T2.3: делегирование routing в runRetrieve (AC-001, RQ-001)
func (b *SearchBuilder) Retrieve(ctx context.Context) (RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return RetrievalResult{}, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return RetrievalResult{}, err
	}
	return b.runRetrieve(ctx, q, b.topK, r)
}

// Answer выполняет RAG-ответ и возвращает строку.
// Routing делегирован в runAnswer.
//
// @sk-task api-consistency-pass#T2.3: делегирование routing в runAnswer (AC-001, RQ-001)
func (b *SearchBuilder) Answer(ctx context.Context) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return "", err
	}
	return b.runAnswer(ctx, q, b.topK, r)
}

// Cite выполняет RAG-ответ и возвращает ответ + источники (чанки со score).
// Routing делегирован в runCite.
//
// @sk-task api-consistency-pass#T2.3: делегирование routing в runCite (AC-001, RQ-001)
func (b *SearchBuilder) Cite(ctx context.Context) (string, RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", RetrievalResult{}, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return "", RetrievalResult{}, err
	}
	return b.runCite(ctx, q, b.topK, r)
}

// InlineCite выполняет RAG-ответ с inline-цитатами `[n]`.
// LLM расставляет ссылки в тексте; citations содержит только использованные источники.
// Routing делегирован в runInlineCite.
//
// @sk-task api-consistency-pass#T2.3: делегирование routing в runInlineCite (AC-001, AC-002, RQ-001)
func (b *SearchBuilder) InlineCite(ctx context.Context) (string, RetrievalResult, []InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", RetrievalResult{}, nil, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return "", RetrievalResult{}, nil, err
	}
	return b.runInlineCite(ctx, q, b.topK, r)
}

// Stream выполняет RAG-ответ через streaming (токен за токеном).
// Если LLM не поддерживает streaming — возвращает ErrStreamingNotSupported.
// Routing делегирован в runStream.
//
// @sk-task api-consistency-pass#T2.3: делегирование routing в runStream (AC-001, RQ-001)
func (b *SearchBuilder) Stream(ctx context.Context) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return nil, err
	}
	return b.runStream(ctx, q, b.topK, r)
}

// StreamSources выполняет RAG-ответ через streaming с синхронно готовым списком источников.
// sources готов сразу (поиск синхронный); токены — асинхронно через канал.
// Routing делегирован в runStreamSources.
//
// @sk-task api-consistency-pass#T2.3: делегирование routing в runStreamSources (AC-001, RQ-001)
func (b *SearchBuilder) StreamSources(ctx context.Context) (<-chan string, RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, RetrievalResult{}, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return nil, RetrievalResult{}, err
	}
	return b.runStreamSources(ctx, q, b.topK, r)
}

// StreamCite выполняет RAG-ответ через streaming с inline-цитатами.
// sources и citations готовы сразу (поиск синхронный); токены — асинхронно.
// Routing делегирован в runStreamInline.
//
// @sk-task api-consistency-pass#T2.3: делегирование routing в runStreamInline (AC-001, AC-002, RQ-001)
func (b *SearchBuilder) StreamCite(ctx context.Context) (<-chan string, RetrievalResult, []InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, RetrievalResult{}, nil, err
	}
	q, r, err := b.pickRoute()
	if err != nil {
		return nil, RetrievalResult{}, nil, err
	}
	return b.runStreamInline(ctx, q, b.topK, r)
}
