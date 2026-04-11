package draftrag

import (
	"context"
	"errors"
	"strings"

	"github.com/bzdvdn/draftrag/internal/application"
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
func (b *SearchBuilder) Retrieve(ctx context.Context) (RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return RetrievalResult{}, err
	}
	q, err := b.validate()
	if err != nil {
		return RetrievalResult{}, err
	}

	if b.hyDE {
		return b.pipeline.core.QueryHyDE(ctx, q, b.topK)
	}
	if b.multiQuery > 0 {
		return b.pipeline.core.QueryMulti(ctx, q, b.multiQuery, b.topK)
	}
	if b.hybrid != nil {
		res, err := b.pipeline.core.QueryHybrid(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return RetrievalResult{}, ErrHybridNotSupported
		}
		return res, err
	}
	if len(b.parentIDs) > 0 {
		res, err := b.pipeline.core.QueryWithParentIDs(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return RetrievalResult{}, ErrFiltersNotSupported
		}
		return res, err
	}
	if len(b.filter.Fields) > 0 {
		res, err := b.pipeline.core.QueryWithMetadataFilter(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return RetrievalResult{}, ErrFiltersNotSupported
		}
		return res, err
	}
	return b.pipeline.core.Query(ctx, q, b.topK)
}

// Answer выполняет RAG-ответ и возвращает строку.
func (b *SearchBuilder) Answer(ctx context.Context) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	q, err := b.validate()
	if err != nil {
		return "", err
	}

	if b.hyDE {
		return b.pipeline.core.AnswerHyDE(ctx, q, b.topK)
	}
	if b.multiQuery > 0 {
		return b.pipeline.core.AnswerMulti(ctx, q, b.multiQuery, b.topK)
	}
	if b.hybrid != nil {
		answer, err := b.pipeline.core.AnswerHybrid(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return "", ErrHybridNotSupported
		}
		return answer, err
	}
	if len(b.parentIDs) > 0 {
		answer, err := b.pipeline.core.AnswerWithParentIDs(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return "", ErrFiltersNotSupported
		}
		return answer, err
	}
	if len(b.filter.Fields) > 0 {
		answer, err := b.pipeline.core.AnswerWithMetadataFilter(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return "", ErrFiltersNotSupported
		}
		return answer, err
	}
	return b.pipeline.core.Answer(ctx, q, b.topK)
}

// Cite выполняет RAG-ответ и возвращает ответ + источники (чанки со score).
// Поддерживает полный routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
func (b *SearchBuilder) Cite(ctx context.Context) (string, RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", RetrievalResult{}, err
	}
	q, err := b.validate()
	if err != nil {
		return "", RetrievalResult{}, err
	}

	if b.hyDE {
		return b.pipeline.core.AnswerHyDEWithCitations(ctx, q, b.topK)
	}
	if b.multiQuery > 0 {
		return b.pipeline.core.AnswerMultiWithCitations(ctx, q, b.multiQuery, b.topK)
	}
	if b.hybrid != nil {
		answer, sources, err := b.pipeline.core.AnswerHybridWithCitations(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return "", RetrievalResult{}, ErrHybridNotSupported
		}
		return answer, sources, err
	}
	if len(b.parentIDs) > 0 {
		answer, sources, err := b.pipeline.core.AnswerWithCitationsWithParentIDs(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return "", RetrievalResult{}, ErrFiltersNotSupported
		}
		return answer, sources, err
	}
	if len(b.filter.Fields) > 0 {
		answer, sources, err := b.pipeline.core.AnswerWithCitationsWithMetadataFilter(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return "", RetrievalResult{}, ErrFiltersNotSupported
		}
		return answer, sources, err
	}
	return b.pipeline.core.AnswerWithCitations(ctx, q, b.topK)
}

// InlineCite выполняет RAG-ответ с inline-цитатами `[n]`.
// LLM расставляет ссылки в тексте; citations содержит только использованные источники.
// Поддерживает полный routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
func (b *SearchBuilder) InlineCite(ctx context.Context) (string, RetrievalResult, []InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", RetrievalResult{}, nil, err
	}
	q, err := b.validate()
	if err != nil {
		return "", RetrievalResult{}, nil, err
	}

	if b.hyDE {
		return b.pipeline.core.AnswerHyDEWithInlineCitations(ctx, q, b.topK)
	}
	if b.multiQuery > 0 {
		return b.pipeline.core.AnswerMultiWithInlineCitations(ctx, q, b.multiQuery, b.topK)
	}
	if b.hybrid != nil {
		answer, sources, citations, err := b.pipeline.core.AnswerHybridWithInlineCitations(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return "", RetrievalResult{}, nil, ErrHybridNotSupported
		}
		return answer, sources, citations, err
	}
	if len(b.parentIDs) > 0 {
		answer, sources, citations, err := b.pipeline.core.AnswerWithInlineCitationsWithParentIDs(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return "", RetrievalResult{}, nil, ErrFiltersNotSupported
		}
		return answer, sources, citations, err
	}
	if len(b.filter.Fields) > 0 {
		// @ds-task T1.1: маппинг application.ErrFiltersNotSupported в публичный ErrFiltersNotSupported (AC-001, AC-003)
		answer, sources, citations, err := b.pipeline.core.AnswerWithInlineCitationsWithMetadataFilter(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return "", RetrievalResult{}, nil, ErrFiltersNotSupported
		}
		return answer, sources, citations, err
	}
	return b.pipeline.core.AnswerWithInlineCitations(ctx, q, b.topK)
}

// Stream выполняет RAG-ответ через streaming (токен за токеном).
// Если LLM не поддерживает streaming — возвращает ErrStreamingNotSupported.
// Поддерживает полный routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
func (b *SearchBuilder) Stream(ctx context.Context) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q, err := b.validate()
	if err != nil {
		return nil, err
	}

	if b.hyDE {
		tokenChan, err := b.pipeline.core.AnswerHyDEStream(ctx, q, b.topK)
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, ErrStreamingNotSupported
		}
		return tokenChan, err
	}
	if b.multiQuery > 0 {
		tokenChan, err := b.pipeline.core.AnswerMultiStream(ctx, q, b.multiQuery, b.topK)
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, ErrStreamingNotSupported
		}
		return tokenChan, err
	}
	if b.hybrid != nil {
		tokenChan, err := b.pipeline.core.AnswerHybridStream(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return nil, ErrHybridNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, ErrStreamingNotSupported
		}
		return tokenChan, err
	}
	if len(b.parentIDs) > 0 {
		tokenChan, err := b.pipeline.core.AnswerStreamWithParentIDs(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return nil, ErrFiltersNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, ErrStreamingNotSupported
		}
		return tokenChan, err
	}
	if len(b.filter.Fields) > 0 {
		tokenChan, err := b.pipeline.core.AnswerStreamWithMetadataFilter(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return nil, ErrFiltersNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, ErrStreamingNotSupported
		}
		return tokenChan, err
	}
	tokenChan, err := b.pipeline.core.AnswerStream(ctx, q, b.topK)
	if errors.Is(err, application.ErrStreamingNotSupported) {
		return nil, ErrStreamingNotSupported
	}
	return tokenChan, err
}

// StreamSources выполняет RAG-ответ через streaming с синхронно готовым списком источников.
// sources готов сразу (поиск синхронный); токены — асинхронно через канал.
// Если LLM не поддерживает streaming — возвращает ErrStreamingNotSupported.
// Поддерживает полный routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
//
// @ds-task T2.1: потоковый аналог Cite без inline-разметки (AC-001, AC-002, DEC-002)
func (b *SearchBuilder) StreamSources(ctx context.Context) (<-chan string, RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, RetrievalResult{}, err
	}
	q, err := b.validate()
	if err != nil {
		return nil, RetrievalResult{}, err
	}

	if b.hyDE {
		tokenChan, sources, err := b.pipeline.core.AnswerHyDEStreamWithSources(ctx, q, b.topK)
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, ErrStreamingNotSupported
		}
		return tokenChan, sources, err
	}
	if b.multiQuery > 0 {
		tokenChan, sources, err := b.pipeline.core.AnswerMultiStreamWithSources(ctx, q, b.multiQuery, b.topK)
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, ErrStreamingNotSupported
		}
		return tokenChan, sources, err
	}
	if b.hybrid != nil {
		tokenChan, sources, err := b.pipeline.core.AnswerHybridStreamWithSources(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return nil, RetrievalResult{}, ErrHybridNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, ErrStreamingNotSupported
		}
		return tokenChan, sources, err
	}
	if len(b.parentIDs) > 0 {
		tokenChan, sources, err := b.pipeline.core.AnswerStreamWithParentIDsWithSources(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return nil, RetrievalResult{}, ErrFiltersNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, ErrStreamingNotSupported
		}
		return tokenChan, sources, err
	}
	if len(b.filter.Fields) > 0 {
		tokenChan, sources, err := b.pipeline.core.AnswerStreamWithMetadataFilterWithSources(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return nil, RetrievalResult{}, ErrFiltersNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, ErrStreamingNotSupported
		}
		return tokenChan, sources, err
	}
	tokenChan, sources, err := b.pipeline.core.AnswerStreamWithSources(ctx, q, b.topK)
	if errors.Is(err, application.ErrStreamingNotSupported) {
		return nil, RetrievalResult{}, ErrStreamingNotSupported
	}
	return tokenChan, sources, err
}

// StreamCite выполняет RAG-ответ через streaming с inline-цитатами.
// sources и citations готовы сразу (поиск синхронный); токены — асинхронно.
// Если LLM не поддерживает streaming — возвращает ErrStreamingNotSupported.
// Поддерживает полный routing: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
func (b *SearchBuilder) StreamCite(ctx context.Context) (<-chan string, RetrievalResult, []InlineCitation, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, RetrievalResult{}, nil, err
	}
	q, err := b.validate()
	if err != nil {
		return nil, RetrievalResult{}, nil, err
	}

	if b.hyDE {
		tokenChan, sources, citations, err := b.pipeline.core.AnswerHyDEStreamWithInlineCitations(ctx, q, b.topK)
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, nil, ErrStreamingNotSupported
		}
		return tokenChan, sources, citations, err
	}
	if b.multiQuery > 0 {
		tokenChan, sources, citations, err := b.pipeline.core.AnswerMultiStreamWithInlineCitations(ctx, q, b.multiQuery, b.topK)
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, nil, ErrStreamingNotSupported
		}
		return tokenChan, sources, citations, err
	}
	if b.hybrid != nil {
		tokenChan, sources, citations, err := b.pipeline.core.AnswerHybridStreamWithInlineCitations(ctx, q, b.topK, *b.hybrid)
		if errors.Is(err, application.ErrHybridNotSupported) {
			return nil, RetrievalResult{}, nil, ErrHybridNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, nil, ErrStreamingNotSupported
		}
		return tokenChan, sources, citations, err
	}
	if len(b.parentIDs) > 0 {
		tokenChan, sources, citations, err := b.pipeline.core.AnswerStreamWithParentIDsWithInlineCitations(ctx, q, b.topK, b.parentIDs)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return nil, RetrievalResult{}, nil, ErrFiltersNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, nil, ErrStreamingNotSupported
		}
		return tokenChan, sources, citations, err
	}
	if len(b.filter.Fields) > 0 {
		tokenChan, sources, citations, err := b.pipeline.core.AnswerStreamWithMetadataFilterWithInlineCitations(ctx, q, b.topK, b.filter)
		if errors.Is(err, application.ErrFiltersNotSupported) {
			return nil, RetrievalResult{}, nil, ErrFiltersNotSupported
		}
		if errors.Is(err, application.ErrStreamingNotSupported) {
			return nil, RetrievalResult{}, nil, ErrStreamingNotSupported
		}
		return tokenChan, sources, citations, err
	}
	tokenChan, sources, citations, err := b.pipeline.core.AnswerStreamWithInlineCitations(ctx, q, b.topK)
	if errors.Is(err, application.ErrStreamingNotSupported) {
		return nil, RetrievalResult{}, nil, ErrStreamingNotSupported
	}
	return tokenChan, sources, citations, err
}
