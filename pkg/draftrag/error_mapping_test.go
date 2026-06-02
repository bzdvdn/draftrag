package draftrag

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bzdvdn/draftrag/internal/application"
	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test api-consistency-pass#T2.2: mapAppError маршрутизирует все sentinel'ы в публичные ошибки (AC-005, AC-006, RQ-003)
func TestMapAppError_AllSentinels(t *testing.T) {
	cases := []struct {
		name  string
		input error
		want  error
	}{
		// Application-level sentinels → public sentinels
		{
			name:  "application.ErrStreamingNotSupported → ErrStreamingNotSupported",
			input: application.ErrStreamingNotSupported,
			want:  ErrStreamingNotSupported,
		},
		{
			name:  "application.ErrHybridNotSupported → ErrHybridNotSupported",
			input: application.ErrHybridNotSupported,
			want:  ErrHybridNotSupported,
		},
		{
			name:  "application.ErrFiltersNotSupported → ErrFiltersNotSupported",
			input: application.ErrFiltersNotSupported,
			want:  ErrFiltersNotSupported,
		},
		{
			name:  "application.ErrDeleteNotSupported → ErrDeleteNotSupported",
			input: application.ErrDeleteNotSupported,
			want:  ErrDeleteNotSupported,
		},
		// Domain-level sentinels → public sentinels
		{
			name:  "domain.ErrEmptyQueryText → ErrEmptyQuery",
			input: domain.ErrEmptyQueryText,
			want:  ErrEmptyQuery,
		},
		{
			name:  "domain.ErrInvalidQueryTopK → ErrInvalidTopK",
			input: domain.ErrInvalidQueryTopK,
			want:  ErrInvalidTopK,
		},
		{
			name:  "domain.ErrEmptyDocumentContent → ErrEmptyDocument",
			input: domain.ErrEmptyDocumentContent,
			want:  ErrEmptyDocument,
		},
		{
			name:  "domain.ErrEmbeddingDimensionMismatch → ErrEmbeddingDimensionMismatch",
			input: domain.ErrEmbeddingDimensionMismatch,
			want:  ErrEmbeddingDimensionMismatch,
		},
		{
			name:  "domain.ErrUpdateNotAtomic → ErrUpdateNotAtomic",
			input: domain.ErrUpdateNotAtomic,
			want:  ErrUpdateNotAtomic,
		},
		// Wrapped errors (errors wrapped with %w) must still resolve.
		{
			name:  "wrapped domain.ErrEmptyQueryText → ErrEmptyQuery",
			input: fmt.Errorf("validation: %w", domain.ErrEmptyQueryText),
			want:  ErrEmptyQuery,
		},
		{
			name:  "wrapped application.ErrStreamingNotSupported → ErrStreamingNotSupported",
			input: fmt.Errorf("pipeline: %w", application.ErrStreamingNotSupported),
			want:  ErrStreamingNotSupported,
		},
		// Passthrough: non-sentinel error returns unchanged.
		{
			name:  "non-sentinel error → passthrough",
			input: errors.New("some other error"),
			want:  nil, // special: verified via errors.Is below
		},
		// Nil error → nil.
		{
			name:  "nil error → nil",
			input: nil,
			want:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapAppError(tc.input)
			if tc.input == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if tc.want == nil {
				// passthrough: errors.Is should match the original input
				if !errors.Is(got, tc.input) {
					t.Fatalf("expected passthrough of %v, got %v", tc.input, got)
				}
				return
			}
			if !errors.Is(got, tc.want) {
				t.Fatalf("expected errors.Is(got, %v) == true, got %v", tc.want, got)
			}
		})
	}
}
