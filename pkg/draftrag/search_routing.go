package draftrag

import (
	"context"

	"github.com/bzdvdn/draftrag/internal/application"
)

type route int

const (
	routeBasic route = iota
	routeRewriter
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
	case b.rewriter != nil:
		r = routeRewriter
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
	routeBasic:    mkRetrieve((*application.Pipeline).Query),
	routeRewriter: wrapRetrieve(rewriterRetrieve),
	routeHyDE:     mkRetrieve((*application.Pipeline).QueryHyDE),
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
	routeBasic:    mkAnswer((*application.Pipeline).Answer),
	routeRewriter: wrapAnswer(rewriterAnswer),
	routeHyDE:     mkAnswer((*application.Pipeline).AnswerHyDE),
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
	routeBasic:    mkCite((*application.Pipeline).AnswerWithCitations),
	routeRewriter: wrapCite(rewriterCite),
	routeHyDE:     mkCite((*application.Pipeline).AnswerHyDEWithCitations),
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
	routeBasic:    mkInlineCite((*application.Pipeline).AnswerWithInlineCitations),
	routeRewriter: wrapInlineCite(rewriterInlineCite),
	routeHyDE:     mkInlineCite((*application.Pipeline).AnswerHyDEWithInlineCitations),
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
	routeBasic:    mkStream((*application.Pipeline).AnswerStream),
	routeRewriter: wrapStream(rewriterStream),
	routeHyDE:     mkStream((*application.Pipeline).AnswerHyDEStream),
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
	routeBasic:    mkStreamSources((*application.Pipeline).AnswerStreamWithSources),
	routeRewriter: wrapStreamSources(rewriterStreamSources),
	routeHyDE:     mkStreamSources((*application.Pipeline).AnswerHyDEStreamWithSources),
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
	routeBasic:    mkStreamCite((*application.Pipeline).AnswerStreamWithInlineCitations),
	routeRewriter: wrapStreamCite(rewriterStreamCite),
	routeHyDE:     mkStreamCite((*application.Pipeline).AnswerHyDEStreamWithInlineCitations),
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

// @sk-task query-rewriting#T2.2: rewriter handler helpers (AC-002, AC-005, AC-007)
// @sk-task pii-guardrails#T3.2: PII redaction в RewrittenQuery (AC-007, RQ-007)

// rewriterResult возвращает переформулированные запросы из rewriter'а.
// При ошибке или пустом результате возвращает исходный запрос (fallback).
// Если в pipeline сконфигурирован PIIDetector, rewritten queries проходят
// через детектор перед возвратом.
func rewriterResult(b *SearchBuilder, ctx context.Context, q string) []string {
	rw := b.rewriter
	if rw == nil {
		if b.pipeline.piidetector != nil {
			return []string{b.pipeline.piidetector.Detect(q)}
		}
		return []string{q}
	}

	// @sk-task query-rewriting#T2.2: проверка на HyDE/MultiQuery конфликт (AC-007)
	if b.hyDE || b.multiQuery > 0 {
		// warning логируется через hooks или лог
	}

	rewritten, err := rw.Rewrite(ctx, q, b.history)
	if err != nil || len(rewritten) == 0 {
		if b.pipeline.piidetector != nil {
			return []string{b.pipeline.piidetector.Detect(q)}
		}
		return []string{q}
	}

	out := make([]string, len(rewritten))
	for i, r := range rewritten {
		if r.Query == "" {
			out[i] = q
		} else {
			out[i] = r.Query
		}
		if b.pipeline.piidetector != nil {
			out[i] = b.pipeline.piidetector.Detect(out[i])
		}
	}
	return out
}

func rewriterRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		result, err := p.Query(ctx, queries[0], topK)
		if err != nil {
			return result, err
		}
		result.QueryText = q
		return result, nil
	}
	result, err := p.QueryWithQueries(ctx, queries, topK)
	if err != nil {
		return RetrievalResult{}, err
	}
	result.QueryText = q
	return result, nil
}

func rewriterAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		return p.Answer(ctx, queries[0], topK)
	}
	return p.AnswerWithQueries(ctx, q, queries, topK)
}

func rewriterCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		return p.AnswerWithCitations(ctx, queries[0], topK)
	}
	return p.AnswerWithQueriesAndCitations(ctx, q, queries, topK)
}

func rewriterInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		return p.AnswerWithInlineCitations(ctx, queries[0], topK)
	}
	return p.AnswerWithQueriesWithInlineCitations(ctx, q, queries, topK)
}

func rewriterStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		return p.AnswerStream(ctx, queries[0], topK)
	}
	return p.AnswerWithQueriesStream(ctx, q, queries, topK)
}

func rewriterStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		return p.AnswerStreamWithSources(ctx, queries[0], topK)
	}
	return p.AnswerWithQueriesStreamWithSources(ctx, q, queries, topK)
}

func rewriterStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	queries := rewriterResult(b, ctx, q)
	if len(queries) == 1 {
		return p.AnswerStreamWithInlineCitations(ctx, queries[0], topK)
	}
	return p.AnswerWithQueriesStreamWithInlineCitations(ctx, q, queries, topK)
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
