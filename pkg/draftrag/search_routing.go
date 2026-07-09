package draftrag

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/application"
)

type route int

const (
	routeBasic route = iota
	routeHyDE
	routeMultiQuery
	routeHybrid
	routeParentIDs
	routeFilter
)

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

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
//
//nolint:dupl // structurally similar by design (per-output-type maps)
var retrieveHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rRetrieve, error){
	routeBasic: mkRetrieve((*application.Pipeline).Query),
	routeHyDE:  mkRetrieve((*application.Pipeline).QueryHyDE),
	routeMultiQuery: wrapRetrieve(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
		return p.QueryMulti(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapRetrieve(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
		return p.QueryHybrid(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapRetrieve(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
		return p.QueryWithParentIDs(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapRetrieve(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
		return p.QueryWithMetadataFilter(ctx, q, topK, b.filter)
	}),
}

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
//
//nolint:dupl // structurally similar by design (per-output-type maps)
var answerHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rAnswer, error){
	routeBasic: mkAnswer((*application.Pipeline).Answer),
	routeHyDE:  mkAnswer((*application.Pipeline).AnswerHyDE),
	routeMultiQuery: wrapAnswer(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
		return p.AnswerMulti(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapAnswer(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
		return p.AnswerHybrid(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapAnswer(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
		return p.AnswerWithParentIDs(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapAnswer(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
		return p.AnswerWithMetadataFilter(ctx, q, topK, b.filter)
	}),
}

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
var citeHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rCite, error){
	routeBasic: mkCite((*application.Pipeline).AnswerWithCitations),
	routeHyDE:  mkCite((*application.Pipeline).AnswerHyDEWithCitations),
	routeMultiQuery: wrapCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
		return p.AnswerMultiWithCitations(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
		return p.AnswerHybridWithCitations(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
		return p.AnswerWithCitationsWithParentIDs(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
		return p.AnswerWithCitationsWithMetadataFilter(ctx, q, topK, b.filter)
	}),
}

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
var inlineCiteHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rInlineCite, error){
	routeBasic: mkInlineCite((*application.Pipeline).AnswerWithInlineCitations),
	routeHyDE:  mkInlineCite((*application.Pipeline).AnswerHyDEWithInlineCitations),
	routeMultiQuery: wrapInlineCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerMultiWithInlineCitations(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapInlineCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerHybridWithInlineCitations(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapInlineCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerWithInlineCitationsWithParentIDs(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapInlineCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerWithInlineCitationsWithMetadataFilter(ctx, q, topK, b.filter)
	}),
}

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
var streamHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rStream, error){
	routeBasic: mkStream((*application.Pipeline).AnswerStream),
	routeHyDE:  mkStream((*application.Pipeline).AnswerHyDEStream),
	routeMultiQuery: wrapStream(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
		return p.AnswerMultiStream(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapStream(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
		return p.AnswerHybridStream(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapStream(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
		return p.AnswerStreamWithParentIDs(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapStream(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
		return p.AnswerStreamWithMetadataFilter(ctx, q, topK, b.filter)
	}),
}

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
var streamSourcesHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rStreamSources, error){
	routeBasic: mkStreamSources((*application.Pipeline).AnswerStreamWithSources),
	routeHyDE:  mkStreamSources((*application.Pipeline).AnswerHyDEStreamWithSources),
	routeMultiQuery: wrapStreamSources(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
		return p.AnswerMultiStreamWithSources(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapStreamSources(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
		return p.AnswerHybridStreamWithSources(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapStreamSources(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
		return p.AnswerStreamWithParentIDsWithSources(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapStreamSources(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
		return p.AnswerStreamWithMetadataFilterWithSources(ctx, q, topK, b.filter)
	}),
}

// @sk-task arch-generics#T2.1: handler maps через generic factory (AC-001)
var streamCiteHandlers = map[route]func(context.Context, string, int, *SearchBuilder) (rStreamCite, error){
	routeBasic: mkStreamCite((*application.Pipeline).AnswerStreamWithInlineCitations),
	routeHyDE:  mkStreamCite((*application.Pipeline).AnswerHyDEStreamWithInlineCitations),
	routeMultiQuery: wrapStreamCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerMultiStreamWithInlineCitations(ctx, q, b.multiQuery, topK)
	}),
	routeHybrid: wrapStreamCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerHybridStreamWithInlineCitations(ctx, q, topK, *b.hybrid)
	}),
	routeParentIDs: wrapStreamCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerStreamWithParentIDsWithInlineCitations(ctx, q, topK, b.parentIDs)
	}),
	routeFilter: wrapStreamCite(func(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
		return p.AnswerStreamWithMetadataFilterWithInlineCitations(ctx, q, topK, b.filter)
	}),
}

var (
	retrieveRouter      = router[rRetrieve]{handlers: retrieveHandlers}
	answerRouter        = router[rAnswer]{handlers: answerHandlers}
	citeRouter          = router[rCite]{handlers: citeHandlers}
	inlineCiteRouter    = router[rInlineCite]{handlers: inlineCiteHandlers}
	streamRouter        = router[rStream]{handlers: streamHandlers}
	streamSourcesRouter = router[rStreamSources]{handlers: streamSourcesHandlers}
	streamCiteRouter    = router[rStreamCite]{handlers: streamCiteHandlers}
)
