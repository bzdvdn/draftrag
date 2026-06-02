package draftrag

import (
	"context"
)

// route описывает выбранный маршрут retrieval в SearchBuilder.
//
// Приоритет: HyDE > MultiQuery > Hybrid > ParentIDs > Filter > basic.
type route int

const (
	routeBasic route = iota
	routeHyDE
	routeMultiQuery
	routeHybrid
	routeParentIDs
	routeFilter
)

// pickRoute валидирует SearchBuilder и возвращает выбранный маршрут.
//
// @sk-task api-consistency-pass#T2.3: routing decision shared across all public methods (AC-001, AC-002, RQ-001)
func (b *SearchBuilder) pickRoute() (q string, r route, err error) {
	q, err = b.validate()
	if err != nil {
		return
	}
	switch {
	case b.hyDE:
		r = routeHyDE
	case b.multiQuery > 0:
		r = routeMultiQuery
	case b.hybrid != nil:
		r = routeHybrid
	case len(b.parentIDs) > 0:
		r = routeParentIDs
	case len(b.filter.Fields) > 0:
		r = routeFilter
	default:
		r = routeBasic
	}
	return
}

// runRetrieve выполняет выбранный маршрут и возвращает RetrievalResult.
//
// @sk-task api-consistency-pass#T2.3: единая точка retrieval-routing с mapAppError (AC-001, RQ-001)
func (b *SearchBuilder) runRetrieve(ctx context.Context, q string, topK int, r route) (RetrievalResult, error) {
	var (
		res RetrievalResult
		err error
	)
	switch r {
	case routeHyDE:
		res, err = b.pipeline.core.QueryHyDE(ctx, q, topK)
	case routeMultiQuery:
		res, err = b.pipeline.core.QueryMulti(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		res, err = b.pipeline.core.QueryHybrid(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		res, err = b.pipeline.core.QueryWithParentIDs(ctx, q, topK, b.parentIDs)
	case routeFilter:
		res, err = b.pipeline.core.QueryWithMetadataFilter(ctx, q, topK, b.filter)
	default:
		res, err = b.pipeline.core.Query(ctx, q, topK)
	}
	return res, mapAppError(err)
}

// runAnswer выполняет выбранный маршрут и возвращает string-ответ.
//
// @sk-task api-consistency-pass#T2.3: единая точка answer-routing (AC-001, RQ-001)
func (b *SearchBuilder) runAnswer(ctx context.Context, q string, topK int, r route) (string, error) {
	var (
		answer string
		err    error
	)
	switch r {
	case routeHyDE:
		answer, err = b.pipeline.core.AnswerHyDE(ctx, q, topK)
	case routeMultiQuery:
		answer, err = b.pipeline.core.AnswerMulti(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		answer, err = b.pipeline.core.AnswerHybrid(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		answer, err = b.pipeline.core.AnswerWithParentIDs(ctx, q, topK, b.parentIDs)
	case routeFilter:
		answer, err = b.pipeline.core.AnswerWithMetadataFilter(ctx, q, topK, b.filter)
	default:
		answer, err = b.pipeline.core.Answer(ctx, q, topK)
	}
	return answer, mapAppError(err)
}

// runCite выполняет выбранный маршрут и возвращает (answer, RetrievalResult).
//
// @sk-task api-consistency-pass#T2.3: единая точка cite-routing (AC-001, RQ-001)
func (b *SearchBuilder) runCite(ctx context.Context, q string, topK int, r route) (string, RetrievalResult, error) {
	var (
		answer  string
		sources RetrievalResult
		err     error
	)
	switch r {
	case routeHyDE:
		answer, sources, err = b.pipeline.core.AnswerHyDEWithCitations(ctx, q, topK)
	case routeMultiQuery:
		answer, sources, err = b.pipeline.core.AnswerMultiWithCitations(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		answer, sources, err = b.pipeline.core.AnswerHybridWithCitations(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		answer, sources, err = b.pipeline.core.AnswerWithCitationsWithParentIDs(ctx, q, topK, b.parentIDs)
	case routeFilter:
		answer, sources, err = b.pipeline.core.AnswerWithCitationsWithMetadataFilter(ctx, q, topK, b.filter)
	default:
		answer, sources, err = b.pipeline.core.AnswerWithCitations(ctx, q, topK)
	}
	return answer, sources, mapAppError(err)
}

// runInlineCite выполняет выбранный маршрут и возвращает (answer, RetrievalResult, []InlineCitation).
//
// @sk-task api-consistency-pass#T2.3: единая точка inline-cite-routing (AC-001, AC-002, RQ-001)
func (b *SearchBuilder) runInlineCite(ctx context.Context, q string, topK int, r route) (string, RetrievalResult, []InlineCitation, error) {
	var (
		answer    string
		sources   RetrievalResult
		citations []InlineCitation
		err       error
	)
	switch r {
	case routeHyDE:
		answer, sources, citations, err = b.pipeline.core.AnswerHyDEWithInlineCitations(ctx, q, topK)
	case routeMultiQuery:
		answer, sources, citations, err = b.pipeline.core.AnswerMultiWithInlineCitations(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		answer, sources, citations, err = b.pipeline.core.AnswerHybridWithInlineCitations(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		answer, sources, citations, err = b.pipeline.core.AnswerWithInlineCitationsWithParentIDs(ctx, q, topK, b.parentIDs)
	case routeFilter:
		answer, sources, citations, err = b.pipeline.core.AnswerWithInlineCitationsWithMetadataFilter(ctx, q, topK, b.filter)
	default:
		answer, sources, citations, err = b.pipeline.core.AnswerWithInlineCitations(ctx, q, topK)
	}
	return answer, sources, citations, mapAppError(err)
}

// runStream выполняет выбранный маршрут и возвращает канал токенов.
//
// @sk-task api-consistency-pass#T2.3: единая точка stream-routing с ErrStreamingNotSupported mapping (AC-001, RQ-001)
func (b *SearchBuilder) runStream(ctx context.Context, q string, topK int, r route) (<-chan string, error) {
	var (
		ch  <-chan string
		err error
	)
	switch r {
	case routeHyDE:
		ch, err = b.pipeline.core.AnswerHyDEStream(ctx, q, topK)
	case routeMultiQuery:
		ch, err = b.pipeline.core.AnswerMultiStream(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		ch, err = b.pipeline.core.AnswerHybridStream(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		ch, err = b.pipeline.core.AnswerStreamWithParentIDs(ctx, q, topK, b.parentIDs)
	case routeFilter:
		ch, err = b.pipeline.core.AnswerStreamWithMetadataFilter(ctx, q, topK, b.filter)
	default:
		ch, err = b.pipeline.core.AnswerStream(ctx, q, topK)
	}
	return ch, mapAppError(err)
}

// runStreamSources выполняет выбранный маршрут и возвращает (chan, RetrievalResult).
//
// @sk-task api-consistency-pass#T2.3: единая точка StreamSources-routing (AC-001, RQ-001)
func (b *SearchBuilder) runStreamSources(ctx context.Context, q string, topK int, r route) (<-chan string, RetrievalResult, error) {
	var (
		ch      <-chan string
		sources RetrievalResult
		err     error
	)
	switch r {
	case routeHyDE:
		ch, sources, err = b.pipeline.core.AnswerHyDEStreamWithSources(ctx, q, topK)
	case routeMultiQuery:
		ch, sources, err = b.pipeline.core.AnswerMultiStreamWithSources(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		ch, sources, err = b.pipeline.core.AnswerHybridStreamWithSources(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		ch, sources, err = b.pipeline.core.AnswerStreamWithParentIDsWithSources(ctx, q, topK, b.parentIDs)
	case routeFilter:
		ch, sources, err = b.pipeline.core.AnswerStreamWithMetadataFilterWithSources(ctx, q, topK, b.filter)
	default:
		ch, sources, err = b.pipeline.core.AnswerStreamWithSources(ctx, q, topK)
	}
	return ch, sources, mapAppError(err)
}

// runStreamInline выполняет выбранный маршрут и возвращает (chan, RetrievalResult, []InlineCitation).
//
// @sk-task api-consistency-pass#T2.3: единая точка StreamCite-routing (AC-001, AC-002, RQ-001)
func (b *SearchBuilder) runStreamInline(ctx context.Context, q string, topK int, r route) (<-chan string, RetrievalResult, []InlineCitation, error) {
	var (
		ch        <-chan string
		sources   RetrievalResult
		citations []InlineCitation
		err       error
	)
	switch r {
	case routeHyDE:
		ch, sources, citations, err = b.pipeline.core.AnswerHyDEStreamWithInlineCitations(ctx, q, topK)
	case routeMultiQuery:
		ch, sources, citations, err = b.pipeline.core.AnswerMultiStreamWithInlineCitations(ctx, q, b.multiQuery, topK)
	case routeHybrid:
		ch, sources, citations, err = b.pipeline.core.AnswerHybridStreamWithInlineCitations(ctx, q, topK, *b.hybrid)
	case routeParentIDs:
		ch, sources, citations, err = b.pipeline.core.AnswerStreamWithParentIDsWithInlineCitations(ctx, q, topK, b.parentIDs)
	case routeFilter:
		ch, sources, citations, err = b.pipeline.core.AnswerStreamWithMetadataFilterWithInlineCitations(ctx, q, topK, b.filter)
	default:
		ch, sources, citations, err = b.pipeline.core.AnswerStreamWithInlineCitations(ctx, q, topK)
	}
	return ch, sources, citations, mapAppError(err)
}
