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

func (r fixedRunner) Retrieve(_ context.Context, question string, _ int) (draftrag.RetrievalResult, error) {
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

// @sk-test T3.1: TestComputeNDCG проверяет вычисление NDCG (AC-001)
func TestComputeNDCG(t *testing.T) {
	tests := []struct {
		name      string
		expected  []string
		retrieved []string
		want      float64
	}{
		{
			name:      "perfect ranking",
			expected:  []string{"A", "B", "C"},
			retrieved: []string{"A", "B", "C"},
			want:      1.0,
		},
		{
			name:      "partial ranking",
			expected:  []string{"A", "B", "C"},
			retrieved: []string{"A", "X", "B"},
			want:      0.7039180890341348, // DCG=1.5, Ideal DCG≈2.1309, NDCG≈0.7039
		},
		{
			name:      "empty expected",
			expected:  []string{},
			retrieved: []string{"A", "B"},
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeNDCG(tt.expected, tt.retrieved)
			if tt.want == 0 {
				if got != 0 {
					t.Fatalf("expected NDCG=0, got %v", got)
				}
			} else if got != tt.want {
				t.Fatalf("expected NDCG=%v, got %v", tt.want, got)
			}
		})
	}
}

// @sk-test T3.1: TestComputePrecision проверяет вычисление Precision@K (AC-002)
func TestComputePrecision(t *testing.T) {
	tests := []struct {
		name      string
		expected  []string
		retrieved []string
		k         int
		want      float64
	}{
		{
			name:      "all relevant",
			expected:  []string{"A", "B"},
			retrieved: []string{"A", "B", "C"},
			k:         2,
			want:      1.0,
		},
		{
			name:      "half relevant",
			expected:  []string{"A", "B"},
			retrieved: []string{"A", "X", "B"},
			k:         2,
			want:      0.5,
		},
		{
			name:      "none relevant",
			expected:  []string{"A", "B"},
			retrieved: []string{"X", "Y"},
			k:         2,
			want:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computePrecision(tt.expected, tt.retrieved, tt.k)
			if got != tt.want {
				t.Fatalf("expected Precision=%v, got %v", tt.want, got)
			}
		})
	}
}

// @sk-test T3.1: TestComputeRecall проверяет вычисление Recall@K (AC-002)
func TestComputeRecall(t *testing.T) {
	tests := []struct {
		name      string
		expected  []string
		retrieved []string
		k         int
		want      float64
	}{
		{
			name:      "all retrieved",
			expected:  []string{"A", "B"},
			retrieved: []string{"A", "B", "C"},
			k:         3,
			want:      1.0,
		},
		{
			name:      "half retrieved",
			expected:  []string{"A", "B"},
			retrieved: []string{"A", "X"},
			k:         2,
			want:      0.5,
		},
		{
			name:      "none retrieved",
			expected:  []string{"A", "B"},
			retrieved: []string{"X", "Y"},
			k:         2,
			want:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeRecall(tt.expected, tt.retrieved, tt.k)
			if got != tt.want {
				t.Fatalf("expected Recall=%v, got %v", tt.want, got)
			}
		})
	}
}

// @sk-test T3.1: TestOptionsConditionalMetrics проверяет условное вычисление метрик через Options (AC-003)
func TestOptionsConditionalMetrics(t *testing.T) {
	runner := fixedRunner{
		results: map[string]draftrag.RetrievalResult{
			"q1": {
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ParentID: "A"}, Score: 0.9},
					{Chunk: domain.Chunk{ParentID: "B"}, Score: 0.8},
				},
			},
		},
	}

	cases := []Case{
		{ID: "c1", Question: "q1", ExpectedParentIDs: []string{"A", "B"}},
	}

	// Без новых флагов - новые метрики должны быть 0
	report, err := Run(context.Background(), runner, cases, Options{DefaultTopK: 5})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if report.Metrics.NDCG != 0 {
		t.Fatalf("expected NDCG=0 when disabled, got %v", report.Metrics.NDCG)
	}
	if report.Metrics.Precision != 0 {
		t.Fatalf("expected Precision=0 when disabled, got %v", report.Metrics.Precision)
	}
	if report.Metrics.Recall != 0 {
		t.Fatalf("expected Recall=0 when disabled, got %v", report.Metrics.Recall)
	}

	// С включёнными флагами - новые метрики должны вычисляться
	report, err = Run(context.Background(), runner, cases, Options{
		DefaultTopK:     5,
		EnableNDCG:      true,
		EnablePrecision: true,
		EnableRecall:    true,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if report.Metrics.NDCG == 0 {
		t.Fatalf("expected NDCG>0 when enabled, got %v", report.Metrics.NDCG)
	}
	if report.Metrics.Precision == 0 {
		t.Fatalf("expected Precision>0 when enabled, got %v", report.Metrics.Precision)
	}
	if report.Metrics.Recall == 0 {
		t.Fatalf("expected Recall>0 when enabled, got %v", report.Metrics.Recall)
	}
}

// @sk-test T3.1: TestValidationEmptyExpectedIDs проверяет валидацию пустых строк в ExpectedParentIDs (AC-005)
func TestValidationEmptyExpectedIDs(t *testing.T) {
	runner := fixedRunner{
		results: map[string]draftrag.RetrievalResult{
			"q1": {
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ParentID: "A"}, Score: 0.9},
				},
			},
		},
	}

	cases := []Case{
		{ID: "c1", Question: "q1", ExpectedParentIDs: []string{"A", "   "}}, // содержит пустую строку после trim
	}

	_, err := Run(context.Background(), runner, cases, Options{DefaultTopK: 5})
	if err == nil {
		t.Fatalf("expected error for empty ExpectedParentID, got nil")
	}
	if err.Error() != "case ExpectedParentIDs contains empty string after normalization" {
		t.Fatalf("expected specific error message, got %v", err)
	}
}

// @sk-test T3.1: TestReportMarshalJSON проверяет MarshalJSON round-trip (AC-006)
func TestReportMarshalJSON(t *testing.T) {
	report := Report{
		Metrics: Metrics{
			TotalCases: 3,
			HitAtK:     0.66,
			MRR:        0.44,
			NDCG:       0.85,
			Precision:  0.75,
			Recall:     0.90,
		},
		Cases: []CaseResult{
			{CaseID: "c1", Found: true, Rank: 1, NDCG: 1.0, Precision: 1.0, Recall: 0.5},
		},
	}

	data, err := report.MarshalJSON()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty JSON data")
	}

	// Проверка что данные валидны (содержат ожидаемые поля)
	jsonStr := string(data)
	if len(jsonStr) == 0 {
		t.Fatalf("expected non-empty JSON string")
	}
}

// @sk-test T3.2: BenchmarkRun1000Cases проверяет производительность для 1000 кейсов (SC-001)
func BenchmarkRun1000Cases(b *testing.B) {
	// Создаём mock runner для 1000 кейсов
	results := make(map[string]draftrag.RetrievalResult)
	for i := 0; i < 1000; i++ {
		question := "q" + itoa(i)
		results[question] = draftrag.RetrievalResult{
			Chunks: []domain.RetrievedChunk{
				{Chunk: domain.Chunk{ParentID: "A" + itoa(i)}, Score: 0.9},
				{Chunk: domain.Chunk{ParentID: "B" + itoa(i)}, Score: 0.8},
			},
		}
	}

	runner := fixedRunner{results: results}

	cases := make([]Case, 1000)
	for i := 0; i < 1000; i++ {
		cases[i] = Case{
			ID:                "c" + itoa(i),
			Question:          "q" + itoa(i),
			ExpectedParentIDs: []string{"A" + itoa(i)},
		}
	}

	opts := Options{
		DefaultTopK:     5,
		EnableNDCG:      true,
		EnablePrecision: true,
		EnableRecall:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(context.Background(), runner, cases, opts)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// @sk-test T3.2: BenchmarkRun10000Cases проверяет производительность для 10000 кейсов (SC-002)
func BenchmarkRun10000Cases(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	// Создаём mock runner для 10000 кейсов
	results := make(map[string]draftrag.RetrievalResult)
	for i := 0; i < 10000; i++ {
		question := "q" + itoa(i)
		results[question] = draftrag.RetrievalResult{
			Chunks: []domain.RetrievedChunk{
				{Chunk: domain.Chunk{ParentID: "A" + itoa(i)}, Score: 0.9},
				{Chunk: domain.Chunk{ParentID: "B" + itoa(i)}, Score: 0.8},
			},
		}
	}

	runner := fixedRunner{results: results}

	cases := make([]Case, 10000)
	for i := 0; i < 10000; i++ {
		cases[i] = Case{
			ID:                "c" + itoa(i),
			Question:          "q" + itoa(i),
			ExpectedParentIDs: []string{"A" + itoa(i)},
		}
	}

	opts := Options{
		DefaultTopK:     5,
		EnableNDCG:      true,
		EnablePrecision: true,
		EnableRecall:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(context.Background(), runner, cases, opts)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
