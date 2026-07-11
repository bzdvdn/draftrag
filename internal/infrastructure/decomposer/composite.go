package decomposer

import (
	"context"
	"errors"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var (
	errCompositeNilPrimary   = errors.New("composite decomposer: primary must not be nil")
	errCompositeNilSecondary = errors.New("composite decomposer: secondary must not be nil")
)

// @sk-task sub-query-decomposition#T3.1: CompositeDecomposer (AC-005)
// CompositeDecomposer объединяет несколько QueryDecomposer'ов в цепочку fallback.
//
// Порядок:
//  1. Primary decomposer (LLM).
//  2. Если primary вернул ошибку или nil/пустой — Secondary decomposer (Rule).
//  3. Если secondary вернул ошибку или nil/пустой — возвращается nil (single-query fallback).
type CompositeDecomposer struct {
	primary   domain.QueryDecomposer
	secondary domain.QueryDecomposer
}

// NewCompositeDecomposer создаёт CompositeDecomposer с primary (LLM) и secondary (Rule) decomposer'ами.
// Оба decomposer'а должны быть не nil.
func NewCompositeDecomposer(primary, secondary domain.QueryDecomposer) (*CompositeDecomposer, error) {
	if primary == nil {
		return nil, errCompositeNilPrimary
	}
	if secondary == nil {
		return nil, errCompositeNilSecondary
	}
	return &CompositeDecomposer{
		primary:   primary,
		secondary: secondary,
	}, nil
}

// @sk-task sub-query-decomposition#T3.1: Decompose — composite chain (AC-005)
// Decompose выполняет primary → secondary → nil fallback chain.
func (d *CompositeDecomposer) Decompose(ctx context.Context, query string) ([]string, error) {
	subs, err := d.primary.Decompose(ctx, query)
	if err == nil && len(subs) > 0 {
		return subs, nil
	}

	subs, err = d.secondary.Decompose(ctx, query)
	if err == nil && len(subs) > 0 {
		return subs, nil
	}

	return nil, nil
}
