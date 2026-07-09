package draftrag

import (
	"context"
	"errors"

	"github.com/bzdvdn/draftrag/internal/application"
)

var errUnknownRoute = errors.New("unknown route")

// @sk-task arch-generics#T1.1: generic-маршрутизатор для output-методов SearchBuilder (AC-001)
type router[T any] struct {
	handlers map[route]func(context.Context, string, int, *SearchBuilder) (T, error)
}

// @sk-task arch-generics#T1.1: execute — dispatch + mapAppError в одном месте (AC-001)
func (r *router[T]) execute(ctx context.Context, q string, topK int, rc route, b *SearchBuilder) (T, error) {
	h, ok := r.handlers[rc]
	if !ok {
		var zero T
		return zero, errUnknownRoute
	}
	res, err := h(ctx, q, topK, b)
	return res, mapAppError(err)
}

// @sk-task arch-generics#T1.1: result-struct для Retrieve (AC-001)
type rRetrieve struct{ Result RetrievalResult }

// @sk-task arch-generics#T1.1: result-struct для Answer (AC-001)
type rAnswer struct{ Text string }

// @sk-task arch-generics#T1.1: result-struct для Cite (AC-001)
type rCite struct {
	Text    string
	Sources RetrievalResult
}

// @sk-task arch-generics#T1.1: result-struct для InlineCite (AC-001)
type rInlineCite struct {
	Text      string
	Sources   RetrievalResult
	Citations []InlineCitation
}

// @sk-task arch-generics#T1.1: result-struct для Stream (AC-001)
type rStream struct{ Ch <-chan string }

// @sk-task arch-generics#T1.1: result-struct для StreamSources (AC-001)
type rStreamSources struct {
	Ch      <-chan string
	Sources RetrievalResult
}

// @sk-task arch-generics#T1.1: result-struct для StreamCite (AC-001)
type rStreamCite struct {
	Ch        <-chan string
	Sources   RetrievalResult
	Citations []InlineCitation
}

// @sk-task arch-generics#T1.1: handler factory helpers для output-методов SearchBuilder (AC-001)

type retrieveCore func(*application.Pipeline, context.Context, string, int) (RetrievalResult, error)

func mkRetrieve(fn retrieveCore) func(context.Context, string, int, *SearchBuilder) (rRetrieve, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) {
		res, err := fn(b.pipeline.core, ctx, q, topK)
		return rRetrieve{Result: res}, err
	}
}

type answerCore func(*application.Pipeline, context.Context, string, int) (string, error)

func mkAnswer(fn answerCore) func(context.Context, string, int, *SearchBuilder) (rAnswer, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) {
		text, err := fn(b.pipeline.core, ctx, q, topK)
		return rAnswer{Text: text}, err
	}
}

type citeCore func(*application.Pipeline, context.Context, string, int) (string, RetrievalResult, error)

func mkCite(fn citeCore) func(context.Context, string, int, *SearchBuilder) (rCite, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) {
		t, s, err := fn(b.pipeline.core, ctx, q, topK)
		return rCite{Text: t, Sources: s}, err
	}
}

type inlineCiteCore func(*application.Pipeline, context.Context, string, int) (string, RetrievalResult, []InlineCitation, error)

func mkInlineCite(fn inlineCiteCore) func(context.Context, string, int, *SearchBuilder) (rInlineCite, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) {
		t, s, c, err := fn(b.pipeline.core, ctx, q, topK)
		return rInlineCite{Text: t, Sources: s, Citations: c}, err
	}
}

type streamCore func(*application.Pipeline, context.Context, string, int) (<-chan string, error)

func mkStream(fn streamCore) func(context.Context, string, int, *SearchBuilder) (rStream, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) {
		ch, err := fn(b.pipeline.core, ctx, q, topK)
		return rStream{Ch: ch}, err
	}
}

type streamSourcesCore func(*application.Pipeline, context.Context, string, int) (<-chan string, RetrievalResult, error)

func mkStreamSources(fn streamSourcesCore) func(context.Context, string, int, *SearchBuilder) (rStreamSources, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) {
		ch, s, err := fn(b.pipeline.core, ctx, q, topK)
		return rStreamSources{Ch: ch, Sources: s}, err
	}
}

type streamCiteCore func(*application.Pipeline, context.Context, string, int) (<-chan string, RetrievalResult, []InlineCitation, error)

func mkStreamCite(fn streamCiteCore) func(context.Context, string, int, *SearchBuilder) (rStreamCite, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) {
		ch, s, c, err := fn(b.pipeline.core, ctx, q, topK)
		return rStreamCite{Ch: ch, Sources: s, Citations: c}, err
	}
}

// wrapRetrieve — handler builder для complex routes с доступом к SearchBuilder.
type retrieveComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (RetrievalResult, error)

func wrapRetrieve(fn retrieveComplex) func(context.Context, string, int, *SearchBuilder) (rRetrieve, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rRetrieve, error) {
		res, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rRetrieve{Result: res}, err
	}
}

type answerComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (string, error)

func wrapAnswer(fn answerComplex) func(context.Context, string, int, *SearchBuilder) (rAnswer, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnswer, error) {
		text, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rAnswer{Text: text}, err
	}
}

type citeComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (string, RetrievalResult, error)

func wrapCite(fn citeComplex) func(context.Context, string, int, *SearchBuilder) (rCite, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rCite, error) {
		t, s, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rCite{Text: t, Sources: s}, err
	}
}

type inlineCiteComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (string, RetrievalResult, []InlineCitation, error)

func wrapInlineCite(fn inlineCiteComplex) func(context.Context, string, int, *SearchBuilder) (rInlineCite, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rInlineCite, error) {
		t, s, c, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rInlineCite{Text: t, Sources: s, Citations: c}, err
	}
}

type streamComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (<-chan string, error)

func wrapStream(fn streamComplex) func(context.Context, string, int, *SearchBuilder) (rStream, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStream, error) {
		ch, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rStream{Ch: ch}, err
	}
}

type streamSourcesComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (<-chan string, RetrievalResult, error)

func wrapStreamSources(fn streamSourcesComplex) func(context.Context, string, int, *SearchBuilder) (rStreamSources, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamSources, error) {
		ch, s, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rStreamSources{Ch: ch, Sources: s}, err
	}
}

type streamCiteComplex func(*application.Pipeline, context.Context, string, int, *SearchBuilder) (<-chan string, RetrievalResult, []InlineCitation, error)

func wrapStreamCite(fn streamCiteComplex) func(context.Context, string, int, *SearchBuilder) (rStreamCite, error) {
	return func(ctx context.Context, q string, topK int, b *SearchBuilder) (rStreamCite, error) {
		ch, s, c, err := fn(b.pipeline.core, ctx, q, topK, b)
		return rStreamCite{Ch: ch, Sources: s, Citations: c}, err
	}
}
