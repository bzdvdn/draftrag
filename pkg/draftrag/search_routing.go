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

// @sk-task searchbuilder-generics#T2.1: handler maps для всех output-методов (AC-001)
var retrieveHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rRetrieve, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) { res, err := b.pipeline.core.Query(ctx, q, topK); return rRetrieve{Result: res}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) { res, err := b.pipeline.core.QueryHyDE(ctx, q, topK); return rRetrieve{Result: res}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) { res, err := b.pipeline.core.QueryMulti(ctx, q, b.multiQuery, topK); return rRetrieve{Result: res}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) { res, err := b.pipeline.core.QueryHybrid(ctx, q, topK, *b.hybrid); return rRetrieve{Result: res}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) { res, err := b.pipeline.core.QueryWithParentIDs(ctx, q, topK, b.parentIDs); return rRetrieve{Result: res}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) { res, err := b.pipeline.core.QueryWithMetadataFilter(ctx, q, topK, b.filter); return rRetrieve{Result: res}, err },
}

// @sk-task searchbuilder-generics#T2.1: handler maps для Answer (AC-001)
var answerHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rAnswer, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) { t, err := b.pipeline.core.Answer(ctx, q, topK); return rAnswer{Text: t}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) { t, err := b.pipeline.core.AnswerHyDE(ctx, q, topK); return rAnswer{Text: t}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) { t, err := b.pipeline.core.AnswerMulti(ctx, q, b.multiQuery, topK); return rAnswer{Text: t}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) { t, err := b.pipeline.core.AnswerHybrid(ctx, q, topK, *b.hybrid); return rAnswer{Text: t}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) { t, err := b.pipeline.core.AnswerWithParentIDs(ctx, q, topK, b.parentIDs); return rAnswer{Text: t}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) { t, err := b.pipeline.core.AnswerWithMetadataFilter(ctx, q, topK, b.filter); return rAnswer{Text: t}, err },
}

// @sk-task searchbuilder-generics#T2.1: handler maps для Cite (AC-001)
var citeHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rCite, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) { t, s, err := b.pipeline.core.AnswerWithCitations(ctx, q, topK); return rCite{Text: t, Sources: s}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) { t, s, err := b.pipeline.core.AnswerHyDEWithCitations(ctx, q, topK); return rCite{Text: t, Sources: s}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) { t, s, err := b.pipeline.core.AnswerMultiWithCitations(ctx, q, b.multiQuery, topK); return rCite{Text: t, Sources: s}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) { t, s, err := b.pipeline.core.AnswerHybridWithCitations(ctx, q, topK, *b.hybrid); return rCite{Text: t, Sources: s}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) { t, s, err := b.pipeline.core.AnswerWithCitationsWithParentIDs(ctx, q, topK, b.parentIDs); return rCite{Text: t, Sources: s}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) { t, s, err := b.pipeline.core.AnswerWithCitationsWithMetadataFilter(ctx, q, topK, b.filter); return rCite{Text: t, Sources: s}, err },
}

// @sk-task searchbuilder-generics#T2.1: handler maps для InlineCite (AC-001)
var inlineCiteHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rInlineCite, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) { t, s, c, err := b.pipeline.core.AnswerWithInlineCitations(ctx, q, topK); return rInlineCite{Text: t, Sources: s, Citations: c}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) { t, s, c, err := b.pipeline.core.AnswerHyDEWithInlineCitations(ctx, q, topK); return rInlineCite{Text: t, Sources: s, Citations: c}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) { t, s, c, err := b.pipeline.core.AnswerMultiWithInlineCitations(ctx, q, b.multiQuery, topK); return rInlineCite{Text: t, Sources: s, Citations: c}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) { t, s, c, err := b.pipeline.core.AnswerHybridWithInlineCitations(ctx, q, topK, *b.hybrid); return rInlineCite{Text: t, Sources: s, Citations: c}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) { t, s, c, err := b.pipeline.core.AnswerWithInlineCitationsWithParentIDs(ctx, q, topK, b.parentIDs); return rInlineCite{Text: t, Sources: s, Citations: c}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) { t, s, c, err := b.pipeline.core.AnswerWithInlineCitationsWithMetadataFilter(ctx, q, topK, b.filter); return rInlineCite{Text: t, Sources: s, Citations: c}, err },
}

// @sk-task searchbuilder-generics#T2.1: handler maps для Stream (AC-001)
var streamHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rStream, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) { ch, err := b.pipeline.core.AnswerStream(ctx, q, topK); return rStream{Ch: ch}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) { ch, err := b.pipeline.core.AnswerHyDEStream(ctx, q, topK); return rStream{Ch: ch}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) { ch, err := b.pipeline.core.AnswerMultiStream(ctx, q, b.multiQuery, topK); return rStream{Ch: ch}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) { ch, err := b.pipeline.core.AnswerHybridStream(ctx, q, topK, *b.hybrid); return rStream{Ch: ch}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) { ch, err := b.pipeline.core.AnswerStreamWithParentIDs(ctx, q, topK, b.parentIDs); return rStream{Ch: ch}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) { ch, err := b.pipeline.core.AnswerStreamWithMetadataFilter(ctx, q, topK, b.filter); return rStream{Ch: ch}, err },
}

// @sk-task searchbuilder-generics#T2.1: handler maps для StreamSources (AC-001)
var streamSourcesHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rStreamSources, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) { ch, s, err := b.pipeline.core.AnswerStreamWithSources(ctx, q, topK); return rStreamSources{Ch: ch, Sources: s}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) { ch, s, err := b.pipeline.core.AnswerHyDEStreamWithSources(ctx, q, topK); return rStreamSources{Ch: ch, Sources: s}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) { ch, s, err := b.pipeline.core.AnswerMultiStreamWithSources(ctx, q, b.multiQuery, topK); return rStreamSources{Ch: ch, Sources: s}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) { ch, s, err := b.pipeline.core.AnswerHybridStreamWithSources(ctx, q, topK, *b.hybrid); return rStreamSources{Ch: ch, Sources: s}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) { ch, s, err := b.pipeline.core.AnswerStreamWithParentIDsWithSources(ctx, q, topK, b.parentIDs); return rStreamSources{Ch: ch, Sources: s}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) { ch, s, err := b.pipeline.core.AnswerStreamWithMetadataFilterWithSources(ctx, q, topK, b.filter); return rStreamSources{Ch: ch, Sources: s}, err },
}

// @sk-task searchbuilder-generics#T2.1: handler maps для StreamCite (AC-001)
var streamCiteHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rStreamCite, error){
	routeBasic:      func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) { ch, s, c, err := b.pipeline.core.AnswerStreamWithInlineCitations(ctx, q, topK); return rStreamCite{Ch: ch, Sources: s, Citations: c}, err },
	routeHyDE:       func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) { ch, s, c, err := b.pipeline.core.AnswerHyDEStreamWithInlineCitations(ctx, q, topK); return rStreamCite{Ch: ch, Sources: s, Citations: c}, err },
	routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) { ch, s, c, err := b.pipeline.core.AnswerMultiStreamWithInlineCitations(ctx, q, b.multiQuery, topK); return rStreamCite{Ch: ch, Sources: s, Citations: c}, err },
	routeHybrid:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) { ch, s, c, err := b.pipeline.core.AnswerHybridStreamWithInlineCitations(ctx, q, topK, *b.hybrid); return rStreamCite{Ch: ch, Sources: s, Citations: c}, err },
	routeParentIDs:  func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) { ch, s, c, err := b.pipeline.core.AnswerStreamWithParentIDsWithInlineCitations(ctx, q, topK, b.parentIDs); return rStreamCite{Ch: ch, Sources: s, Citations: c}, err },
	routeFilter:     func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) { ch, s, c, err := b.pipeline.core.AnswerStreamWithMetadataFilterWithInlineCitations(ctx, q, topK, b.filter); return rStreamCite{Ch: ch, Sources: s, Citations: c}, err },
}

var retrieveRouter = router[rRetrieve]{handlers: retrieveHandlers}
var answerRouter = router[rAnswer]{handlers: answerHandlers}
var citeRouter = router[rCite]{handlers: citeHandlers}
var inlineCiteRouter = router[rInlineCite]{handlers: inlineCiteHandlers}
var streamRouter = router[rStream]{handlers: streamHandlers}
var streamSourcesRouter = router[rStreamSources]{handlers: streamSourcesHandlers}
var streamCiteRouter = router[rStreamCite]{handlers: streamCiteHandlers}
