package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test api-consistency-pass#T2.1: validation errors reachable через публичный API
// (errors.Is на ErrEmptyQuery/ErrInvalidTopK/ErrEmptyDocument). (AC-003, RQ-003)
func TestPipelineErrorMapping_ValidationReachesPublicSentinels(t *testing.T) {
	ctx := context.Background()
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		call func() error
		want error
	}{
		{
			name: "Pipeline.Answer with empty question wraps ErrEmptyQuery",
			call: func() error {
				_, err := p.Answer(ctx, "   ")
				return err
			},
			want: ErrEmptyQuery,
		},
		{
			name: "Pipeline.Query with empty question wraps ErrEmptyQuery",
			call: func() error {
				_, err := p.Query(ctx, "   ")
				return err
			},
			want: ErrEmptyQuery,
		},
		{
			name: "Pipeline.Retrieve with empty question wraps ErrEmptyQuery",
			call: func() error {
				_, err := p.Retrieve(ctx, "", 5)
				return err
			},
			want: ErrEmptyQuery,
		},
		{
			name: "Pipeline.Retrieve with topK=0 wraps ErrInvalidTopK",
			call: func() error {
				_, err := p.Retrieve(ctx, "q", 0)
				return err
			},
			want: ErrInvalidTopK,
		},
		{
			name: "Search().Answer with empty question wraps ErrEmptyQuery",
			call: func() error {
				_, err := p.Search("   ").Answer(ctx)
				return err
			},
			want: ErrEmptyQuery,
		},
		{
			name: "Search().Retrieve with empty question wraps ErrEmptyQuery",
			call: func() error {
				_, err := p.Search("   ").Retrieve(ctx)
				return err
			},
			want: ErrEmptyQuery,
		},
		{
			name: "Search().TopK(0).Answer wraps ErrInvalidTopK",
			call: func() error {
				_, err := p.Search("q").TopK(0).Answer(ctx)
				return err
			},
			want: ErrInvalidTopK,
		},
		{
			name: "Search().TopK(0).Retrieve wraps ErrInvalidTopK",
			call: func() error {
				_, err := p.Search("q").TopK(0).Retrieve(ctx)
				return err
			},
			want: ErrInvalidTopK,
		},
		{
			name: "IndexBatch with empty doc.Content wraps ErrEmptyDocument",
			call: func() error {
				_, err := p.IndexBatch(ctx, []Document{{ID: "d1", Content: "  "}}, 1)
				return err
			},
			want: ErrEmptyDocument,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected errors.Is(err, %v) == true, got err=%v", tc.want, err)
			}
		})
	}
}
