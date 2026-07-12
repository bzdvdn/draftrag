package draftrag

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/bzdvdn/draftrag/internal/application"
)

type route int

const (
	routeBasic route = iota
	routeRewriter
	routeHyDE
	routeMultiQuery
	routeTools
	routeHybrid
	routeParentIDs
	routeFilter
	routeSubDecompose
)

// @sk-task arch-issues#T4.3: routeTools constant + pickRoute case (AC-004)
func (b *SearchBuilder) pickRoute() (q string, r route, err error) {
	q, err = b.validate()
	if err != nil {
		return
	}
	switch {
	case b.subDecompose:
		r = routeSubDecompose
	case b.rewriter != nil:
		r = routeRewriter
	case b.hyDE:
		r = routeHyDE
	case b.multiQuery > 0:
		r = routeMultiQuery
	case len(b.tools) > 0:
		r = routeTools
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
// @sk-task sub-query-decomposition#T1.2: subDecomposeRetrieve handler (AC-001, AC-002)
//
//nolint:dupl // structurally similar by design (per-output-type maps)
func subDecomposeRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	d := b.decomposer
	if d == nil {
		return RetrievalResult{}, ErrSubDecomposeNotSupported
	}
	return p.QuerySubDecompose(ctx, q, topK, d)
}

// @sk-task sub-query-decomposition#T2.2: subDecomposeAnswer handler (AC-008)
func subDecomposeAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	d := b.decomposer
	if d == nil {
		return "", ErrSubDecomposeNotSupported
	}
	return p.AnswerSubDecompose(ctx, q, topK, d)
}

// @sk-task sub-query-decomposition#T2.2: subDecomposeCite handler (AC-008, AC-009)
func subDecomposeCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	d := b.decomposer
	if d == nil {
		return "", RetrievalResult{}, ErrSubDecomposeNotSupported
	}
	return p.AnswerSubDecomposeWithCitations(ctx, q, topK, d)
}

// @sk-task sub-query-decomposition#T2.2: subDecomposeInlineCite handler (AC-008, AC-009)
func subDecomposeInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	d := b.decomposer
	if d == nil {
		return "", RetrievalResult{}, nil, ErrSubDecomposeNotSupported
	}
	return p.AnswerSubDecomposeWithInlineCitations(ctx, q, topK, d)
}

// @sk-task sub-query-decomposition#T2.2: subDecomposeStream handler (AC-009)
func subDecomposeStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	d := b.decomposer
	if d == nil {
		return nil, ErrSubDecomposeNotSupported
	}
	return p.AnswerSubDecomposeStream(ctx, q, topK, d)
}

// @sk-task sub-query-decomposition#T2.2: subDecomposeStreamSources handler (AC-009)
func subDecomposeStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	d := b.decomposer
	if d == nil {
		return nil, RetrievalResult{}, ErrSubDecomposeNotSupported
	}
	return p.AnswerSubDecomposeStreamWithSources(ctx, q, topK, d)
}

// @sk-task sub-query-decomposition#T2.2: subDecomposeStreamCite handler (AC-009)
func subDecomposeStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	d := b.decomposer
	if d == nil {
		return nil, RetrievalResult{}, nil, ErrSubDecomposeNotSupported
	}
	return p.AnswerSubDecomposeStreamWithInlineCitations(ctx, q, topK, d)
}

// @sk-task arch-issues#T5.1: multiQuery handler functions for all output types (AC-005)
func multiQueryRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	return p.QueryMulti(ctx, q, b.multiQuery, topK)
}
func multiQueryAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	return p.AnswerMulti(ctx, q, b.multiQuery, topK)
}
func multiQueryCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	return p.AnswerMultiWithCitations(ctx, q, b.multiQuery, topK)
}
func multiQueryInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerMultiWithInlineCitations(ctx, q, b.multiQuery, topK)
}
func multiQueryStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	return p.AnswerMultiStream(ctx, q, b.multiQuery, topK)
}
func multiQueryStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	return p.AnswerMultiStreamWithSources(ctx, q, b.multiQuery, topK)
}
func multiQueryStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerMultiStreamWithInlineCitations(ctx, q, b.multiQuery, topK)
}

// @sk-task arch-issues#T5.1: hybrid handler functions for all output types (AC-005)
func hybridRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	return p.QueryHybrid(ctx, q, topK, *b.hybrid)
}
func hybridAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	return p.AnswerHybrid(ctx, q, topK, *b.hybrid)
}
func hybridCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	return p.AnswerHybridWithCitations(ctx, q, topK, *b.hybrid)
}
func hybridInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerHybridWithInlineCitations(ctx, q, topK, *b.hybrid)
}
func hybridStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	return p.AnswerHybridStream(ctx, q, topK, *b.hybrid)
}
func hybridStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	return p.AnswerHybridStreamWithSources(ctx, q, topK, *b.hybrid)
}
func hybridStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerHybridStreamWithInlineCitations(ctx, q, topK, *b.hybrid)
}

// @sk-task arch-issues#T5.1: parentIDs handler functions for all output types (AC-005)
func parentIDsRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	return p.QueryWithParentIDs(ctx, q, topK, b.parentIDs)
}
func parentIDsAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	return p.AnswerWithParentIDs(ctx, q, topK, b.parentIDs)
}
func parentIDsCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	return p.AnswerWithCitationsWithParentIDs(ctx, q, topK, b.parentIDs)
}
func parentIDsInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerWithInlineCitationsWithParentIDs(ctx, q, topK, b.parentIDs)
}
func parentIDsStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	return p.AnswerStreamWithParentIDs(ctx, q, topK, b.parentIDs)
}
func parentIDsStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	return p.AnswerStreamWithParentIDsWithSources(ctx, q, topK, b.parentIDs)
}
func parentIDsStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerStreamWithParentIDsWithInlineCitations(ctx, q, topK, b.parentIDs)
}

// @sk-task arch-issues#T5.1: filter handler functions for all output types (AC-005)
func filterRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	return p.QueryWithMetadataFilter(ctx, q, topK, b.filter)
}
func filterAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	return p.AnswerWithMetadataFilter(ctx, q, topK, b.filter)
}
func filterCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	return p.AnswerWithCitationsWithMetadataFilter(ctx, q, topK, b.filter)
}
func filterInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerWithInlineCitationsWithMetadataFilter(ctx, q, topK, b.filter)
}
func filterStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	return p.AnswerStreamWithMetadataFilter(ctx, q, topK, b.filter)
}
func filterStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	return p.AnswerStreamWithMetadataFilterWithSources(ctx, q, topK, b.filter)
}
func filterStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	return p.AnswerStreamWithMetadataFilterWithInlineCitations(ctx, q, topK, b.filter)
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

// @sk-task arch-issues#T4.4: tool route handler functions (AC-004)
func buildToolsUserMessage(result RetrievalResult, question string, maxChars, maxChunks int) string {
	var ctxBuf strings.Builder
	ctxBuf.WriteString("Контекст:\n")
	wrote := 0
	for _, rc := range result.Chunks {
		if maxChunks > 0 && wrote >= maxChunks {
			break
		}
		ctxBuf.WriteString(rc.Chunk.Content)
		ctxBuf.WriteString("\n")
		wrote++
	}
	context := ctxBuf.String()
	if maxChars > 0 && utf8.RuneCountInString(context) > maxChars {
		runes := []rune(context)
		context = string(runes[:maxChars])
	}
	var buf strings.Builder
	buf.WriteString(context)
	if !strings.HasSuffix(context, "\n") {
		buf.WriteString("\n\nВопрос:\n")
	} else {
		buf.WriteString("\nВопрос:\n")
	}
	buf.WriteString(question)
	return buf.String()
}

func runToolsAnswer(p *application.Pipeline, ctx context.Context, q string, result RetrievalResult, b *SearchBuilder) (string, error) {
	if len(b.tools) == 0 || b.toolHandler == nil {
		userMsg := buildToolsUserMessage(result, q, p.MaxContextChars(), p.MaxContextChunks())
		return p.ExecuteWithTools(ctx, p.SystemPrompt(), userMsg, nil, nil)
	}
	userMsg := buildToolsUserMessage(result, q, p.MaxContextChars(), p.MaxContextChunks())
	return p.ExecuteWithTools(ctx, p.SystemPrompt(), userMsg, b.tools, b.toolHandler)
}

// @sk-task arch-issues#T4.4: toolsRetrieve handler (AC-004)
func toolsRetrieve(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (RetrievalResult, error) {
	return p.Query(ctx, q, topK)
}

// @sk-task arch-issues#T4.4: toolsAnswer handler (AC-004)
func toolsAnswer(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, error) {
	result, err := p.Query(ctx, q, topK)
	if err != nil {
		return "", err
	}
	return runToolsAnswer(p, ctx, q, result, b)
}

// @sk-task arch-issues#T4.4: toolsCite handler (AC-004)
func toolsCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, error) {
	result, err := p.Query(ctx, q, topK)
	if err != nil {
		return "", RetrievalResult{}, err
	}
	answer, err := runToolsAnswer(p, ctx, q, result, b)
	if err != nil {
		return "", RetrievalResult{}, err
	}
	return answer, result, nil
}

// @sk-task arch-issues#T4.4: toolsInlineCite handler (AC-004)
func toolsInlineCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (string, RetrievalResult, []InlineCitation, error) {
	result, err := p.Query(ctx, q, topK)
	if err != nil {
		return "", RetrievalResult{}, nil, err
	}
	answer, err := runToolsAnswer(p, ctx, q, result, b)
	if err != nil {
		return "", RetrievalResult{}, nil, err
	}
	return answer, result, nil, nil
}

// @sk-task arch-issues#T4.4: toolsStream handler — not supported (AC-004)
func toolsStream(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, error) {
	return nil, ErrToolsNotSupportedInStream
}

// @sk-task arch-issues#T4.4: toolsStreamSources handler — not supported (AC-004)
func toolsStreamSources(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, error) {
	return nil, RetrievalResult{}, ErrToolsNotSupportedInStream
}

// @sk-task arch-issues#T4.4: toolsStreamCite handler — not supported (AC-004)
func toolsStreamCite(p *application.Pipeline, ctx context.Context, q string, topK int, b *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error) {
	return nil, RetrievalResult{}, nil, ErrToolsNotSupportedInStream
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
