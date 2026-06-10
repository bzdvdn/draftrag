package draftrag

import (
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test hardening-2026q2#T3.3: Error chain (AC-009)
func TestErrorChain_ErrEmptyDocument(t *testing.T) {
	err := mapAppError(domain.ErrEmptyDocumentContent)
	if !errors.Is(err, ErrEmptyDocument) {
		t.Error("mapAppError(domain.ErrEmptyDocumentContent) should return ErrEmptyDocument")
	}
}

func TestErrorChain_ErrEmptyQuery(t *testing.T) {
	err := mapAppError(domain.ErrEmptyQueryText)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Error("mapAppError(domain.ErrEmptyQueryText) should return ErrEmptyQuery")
	}
}

func TestErrorChain_ErrInvalidTopK(t *testing.T) {
	err := mapAppError(domain.ErrInvalidQueryTopK)
	if !errors.Is(err, ErrInvalidTopK) {
		t.Error("mapAppError(domain.ErrInvalidQueryTopK) should return ErrInvalidTopK")
	}
}

func TestErrorChain_ErrEmbeddingDimensionMismatch(t *testing.T) {
	err := mapAppError(domain.ErrEmbeddingDimensionMismatch)
	if !errors.Is(err, ErrEmbeddingDimensionMismatch) {
		t.Error("mapAppError(domain.ErrEmbeddingDimensionMismatch) should return ErrEmbeddingDimensionMismatch")
	}
}
