package eval

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

type fixedRunner struct {
	results map[string]draftrag.RetrievalResult
}

func (r fixedRunner) Retrieve(ctx context.Context, question string, topK int) (draftrag.RetrievalResult, error) {
	if rr, ok := r.results[question]; ok {
		return rr, nil
	}
	return draftrag.RetrievalResult{}, nil
}

func TestRun_MetricsOnSyntheticDataset(t *testing.T) {
	runner := fixedRunner{
		results: map[string]draftrag.RetrievalResult{
			"q1": {
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ParentID: "A"}, Score: 0.9},
					{Chunk: domain.Chunk{ParentID: "B"}, Score: 0.8},
				},
			},
			"q2": {
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ParentID: "X"}, Score: 0.9},
					{Chunk: domain.Chunk{ParentID: "Y"}, Score: 0.8},
					{Chunk: domain.Chunk{ParentID: "Z"}, Score: 0.7},
				},
			},
			"q3": {
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ParentID: "N1"}, Score: 0.9},
					{Chunk: domain.Chunk{ParentID: "C"}, Score: 0.8},
				},
			},
		},
	}

	cases := []Case{
		{ID: "c1", Question: "q1", ExpectedParentIDs: []string{"A"}},       // rank=1
		{ID: "c2", Question: "q2", ExpectedParentIDs: []string{"Z"}},       // rank=3
		{ID: "c3", Question: "q3", ExpectedParentIDs: []string{"missing"}}, // rank=0
	}

	report, err := Run(context.Background(), runner, cases, Options{DefaultTopK: 5})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if report.Metrics.TotalCases != 3 {
		t.Fatalf("expected TotalCases=3, got %d", report.Metrics.TotalCases)
	}
	// hits: 2/3
	if report.Metrics.HitAtK != float64(2)/float64(3) {
		t.Fatalf("expected HitAtK=2/3, got %v", report.Metrics.HitAtK)
	}
	// MRR = (1 + 1/3 + 0) / 3 = 4/9
	if report.Metrics.MRR != float64(4)/float64(9) {
		t.Fatalf("expected MRR=4/9, got %v", report.Metrics.MRR)
	}

	if got := report.Cases[0].Rank; got != 1 {
		t.Fatalf("expected c1 rank=1, got %d", got)
	}
	if got := report.Cases[1].Rank; got != 3 {
		t.Fatalf("expected c2 rank=3, got %d", got)
	}
	if got := report.Cases[2].Rank; got != 0 {
		t.Fatalf("expected c3 rank=0, got %d", got)
	}
}
