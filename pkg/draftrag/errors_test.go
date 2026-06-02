package draftrag

import (
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test hardening-2026q2#T3.3: Error chain (AC-009)
func TestErrorChain_ErrEmptyDocument(t *testing.T) {
	if !errors.Is(ErrEmptyDocument, domain.ErrEmptyDocumentContent) {
		t.Error("ErrEmptyDocument should wrap domain.ErrEmptyDocumentContent")
	}
	if !errors.Is(domain.ErrEmptyDocumentContent, ErrEmptyDocument) {
		t.Error("domain.ErrEmptyDocumentContent should be ErrEmptyDocument")
	}
}

func TestErrorChain_ErrEmptyQuery(t *testing.T) {
	if !errors.Is(ErrEmptyQuery, domain.ErrEmptyQueryText) {
		t.Error("ErrEmptyQuery should wrap domain.ErrEmptyQueryText")
	}
	if !errors.Is(domain.ErrEmptyQueryText, ErrEmptyQuery) {
		t.Error("domain.ErrEmptyQueryText should be ErrEmptyQuery")
	}
}

func TestErrorChain_ErrInvalidTopK(t *testing.T) {
	if !errors.Is(ErrInvalidTopK, domain.ErrInvalidQueryTopK) {
		t.Error("ErrInvalidTopK should wrap domain.ErrInvalidQueryTopK")
	}
	if !errors.Is(domain.ErrInvalidQueryTopK, ErrInvalidTopK) {
		t.Error("domain.ErrInvalidQueryTopK should be ErrInvalidTopK")
	}
}

func TestErrorChain_ErrEmbeddingDimensionMismatch(t *testing.T) {
	if !errors.Is(ErrEmbeddingDimensionMismatch, domain.ErrEmbeddingDimensionMismatch) {
		t.Error("ErrEmbeddingDimensionMismatch should wrap domain.ErrEmbeddingDimensionMismatch")
	}
}
