package draftrag

import (
	"context"
	"errors"
)

var errUnknownRoute = errors.New("unknown route")

// @sk-task searchbuilder-generics#T1.1: generic-маршрутизатор для output-методов SearchBuilder (AC-001)
type router[T any] struct {
	handlers map[route]func(context.Context, string, int, *SearchBuilder) (T, error)
}

// @sk-task searchbuilder-generics#T1.1: execute — dispatch + mapAppError в одном месте (AC-001)
func (r *router[T]) execute(ctx context.Context, q string, topK int, rc route, b *SearchBuilder) (T, error) {
	h, ok := r.handlers[rc]
	if !ok {
		var zero T
		return zero, errUnknownRoute
	}
	res, err := h(ctx, q, topK, b)
	return res, mapAppError(err)
}

// @sk-task searchbuilder-generics#T1.1: result-struct для Retrieve (AC-001)
type rRetrieve struct{ Result RetrievalResult }

// @sk-task searchbuilder-generics#T1.1: result-struct для Answer (AC-001)
type rAnswer struct{ Text string }

// @sk-task searchbuilder-generics#T1.1: result-struct для Cite (AC-001)
type rCite struct{ Text string; Sources RetrievalResult }

// @sk-task searchbuilder-generics#T1.1: result-struct для InlineCite (AC-001)
type rInlineCite struct{ Text string; Sources RetrievalResult; Citations []InlineCitation }

// @sk-task searchbuilder-generics#T1.1: result-struct для Stream (AC-001)
type rStream struct{ Ch <-chan string }

// @sk-task searchbuilder-generics#T1.1: result-struct для StreamSources (AC-001)
type rStreamSources struct{ Ch <-chan string; Sources RetrievalResult }

// @sk-task searchbuilder-generics#T1.1: result-struct для StreamCite (AC-001)
type rStreamCite struct{ Ch <-chan string; Sources RetrievalResult; Citations []InlineCitation }
