package eval

import (
	"context"
	"math"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

type mockLLMProvider struct {
	reply string
	err   error
}

func (m mockLLMProvider) Generate(_ context.Context, _, _ string) (string, error) {
	return m.reply, m.err
}

func (mockLLMProvider) Health(_ context.Context) error { return nil }

type mockEmbedder struct {
	vecs map[string][]float64
	err  error
}

func (m mockEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	if v, ok := m.vecs[text]; ok {
		return v, nil
	}
	return nil, nil
}

func (mockEmbedder) Health(_ context.Context) error { return nil }

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_FullySupported (AC-001)
func TestComputeFaithfulness_FullySupported(t *testing.T) {
	llm := mockLLMProvider{
		reply: `{"faithfulness_score": 1.0, "claims": ["claim1"], "supported_claims": ["claim1"], "unsupported_claims": []}`,
	}
	score, err := ComputeFaithfulness(context.Background(), "The sky is blue.", "The sky appears blue due to Rayleigh scattering.", llm)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 1.0 {
		t.Fatalf("expected score=1.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_PartiallySupported (AC-001)
func TestComputeFaithfulness_PartiallySupported(t *testing.T) {
	llm := mockLLMProvider{
		reply: `{"faithfulness_score": 0.5, "claims": ["claim1", "claim2"], "supported_claims": ["claim1"], "unsupported_claims": ["claim2"]}`,
	}
	score, err := ComputeFaithfulness(context.Background(), "The sky is blue and grass is pink.", "The sky appears blue.", llm)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.5 {
		t.Fatalf("expected score=0.5, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_EmptyAnswer (AC-006)
func TestComputeFaithfulness_EmptyAnswer(t *testing.T) {
	llm := mockLLMProvider{reply: `{"faithfulness_score": 1.0}`}
	score, err := ComputeFaithfulness(context.Background(), "", "some context", llm)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_NilProvider (AC-005)
func TestComputeFaithfulness_NilProvider(t *testing.T) {
	score, err := ComputeFaithfulness(context.Background(), "answer", "context", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeContextRelevance_AllRelevant (AC-003)
func TestComputeContextRelevance_AllRelevant(t *testing.T) {
	emb := mockEmbedder{
		vecs: map[string][]float64{
			"question": {1, 0, 0},
			"chunk1":   {1, 0, 0},
			"chunk2":   {1, 0, 0},
		},
	}
	score, err := ComputeContextRelevance(context.Background(), "question", []string{"chunk1", "chunk2"}, emb)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 1.0 {
		t.Fatalf("expected score=1.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeContextRelevance_Partial (AC-003)
func TestComputeContextRelevance_Partial(t *testing.T) {
	emb := mockEmbedder{
		vecs: map[string][]float64{
			"question": {1, 0},
			"chunk1":   {1, 0},
			"chunk2":   {0, 1},
		},
	}
	score, err := ComputeContextRelevance(context.Background(), "question", []string{"chunk1", "chunk2"}, emb)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// cosine([1,0], [1,0]) = 1; cosine([1,0], [0,1]) = 0; avg = 0.5
	if score != 0.5 {
		t.Fatalf("expected score=0.5, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeContextRelevance_EmptyChunks (AC-003)
func TestComputeContextRelevance_EmptyChunks(t *testing.T) {
	emb := mockEmbedder{vecs: map[string][]float64{"question": {1, 0}}}
	score, err := ComputeContextRelevance(context.Background(), "question", []string{}, emb)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeContextRelevance_NilEmbedder (AC-005)
func TestComputeContextRelevance_NilEmbedder(t *testing.T) {
	score, err := ComputeContextRelevance(context.Background(), "question", []string{"chunk1"}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeAnswerRelevance_DirectAnswer (AC-002)
func TestComputeAnswerRelevance_DirectAnswer(t *testing.T) {
	emb := mockEmbedder{
		vecs: map[string][]float64{
			"what is the sky color?": {1, 0},
			"the sky is blue":        {1, 0},
			"i like to eat pizza":    {0, 1},
		},
	}
	directScore, err := ComputeAnswerRelevance(context.Background(), "what is the sky color?", "the sky is blue", emb)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	irrelevantScore, _ := ComputeAnswerRelevance(context.Background(), "what is the sky color?", "i like to eat pizza", emb)
	if directScore <= irrelevantScore {
		t.Fatalf("expected direct score (%v) > irrelevant score (%v)", directScore, irrelevantScore)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeAnswerRelevance_EmptyAnswer (AC-002)
func TestComputeAnswerRelevance_EmptyAnswer(t *testing.T) {
	emb := mockEmbedder{vecs: map[string][]float64{"question": {1, 0}}}
	score, err := ComputeAnswerRelevance(context.Background(), "question", "", emb)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeAnswerRelevance_NilEmbedder (AC-005)
func TestComputeAnswerRelevance_NilEmbedder(t *testing.T) {
	score, err := ComputeAnswerRelevance(context.Background(), "question", "answer", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestCosineSimilarity (AC-003)
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float64
		b    []float64
		want float64
	}{
		{name: "identical", a: []float64{1, 0}, b: []float64{1, 0}, want: 1.0},
		{name: "orthogonal", a: []float64{1, 0}, b: []float64{0, 1}, want: 0.0},
		{name: "opposite", a: []float64{1, 0}, b: []float64{-1, 0}, want: -1.0},
		{name: "partial", a: []float64{1, 0, 0}, b: []float64{1, 1, 0}, want: 1.0 / math.Sqrt2},
		{name: "zero vector", a: []float64{0, 0}, b: []float64{1, 0}, want: 0.0},
		{name: "mismatched length", a: []float64{1}, b: []float64{1, 0}, want: 0.0},
		{name: "empty", a: []float64{}, b: []float64{}, want: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_LLMError (AC-001)
func TestComputeFaithfulness_LLMError(t *testing.T) {
	llm := mockLLMProvider{reply: "", err: assertError("llm failure")}
	_, err := ComputeFaithfulness(context.Background(), "answer", "context", llm)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }

// @sk-test eval-ragas-metrics#T2.4: TestComputeContextRelevance_EmbedderError (AC-003)
func TestComputeContextRelevance_EmbedderError(t *testing.T) {
	emb := mockEmbedder{err: assertError("embedder failure")}
	_, err := ComputeContextRelevance(context.Background(), "question", []string{"chunk1"}, emb)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_InvalidJSON (AC-001)
func TestComputeFaithfulness_InvalidJSON(t *testing.T) {
	llm := mockLLMProvider{reply: "not json"}
	score, err := ComputeFaithfulness(context.Background(), "answer", "context", llm)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0 for invalid JSON, got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T2.4: TestComputeFaithfulness_OutOfRangeScore (AC-001)
func TestComputeFaithfulness_OutOfRangeScore(t *testing.T) {
	llm := mockLLMProvider{reply: `{"faithfulness_score": 1.5}`}
	score, err := ComputeFaithfulness(context.Background(), "answer", "context", llm)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 1.0 {
		t.Fatalf("expected score=1.0 (clamped), got %v", score)
	}

	llm.reply = `{"faithfulness_score": -0.5}`
	score, err = ComputeFaithfulness(context.Background(), "answer", "context", llm)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if score != 0.0 {
		t.Fatalf("expected score=0.0 (clamped), got %v", score)
	}
}

// @sk-test eval-ragas-metrics#T3.2: TestRunWithAnswer_RAGASMetrics (AC-004)
func TestRunWithAnswer_RAGASMetrics(t *testing.T) {
	runner := fixedRunner{
		results: map[string]draftrag.RetrievalResult{
			"what color is the sky?": {
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ParentID: "p1", Content: "The sky appears blue."}, Score: 0.9},
				},
			},
		},
	}

	llm := mockLLMProvider{
		reply: `{"faithfulness_score": 0.9, "claims": ["the sky is blue"], "supported_claims": ["the sky is blue"], "unsupported_claims": []}`,
	}

	emb := mockEmbedder{
		vecs: map[string][]float64{
			"what color is the sky?": {1, 0},
			"The sky appears blue.":  {1, 0},
		},
	}

	cases := []Case{
		{
			ID:                "c1",
			Question:          "what color is the sky?",
			ExpectedParentIDs: []string{"p1"},
			ExpectedAnswer:    "The sky appears blue.",
		},
	}

	report, err := RunWithAnswer(context.Background(), runner, llm, emb, cases, Options{
		DefaultTopK:            5,
		EnableFaithfulness:     true,
		EnableAnswerRelevance:  true,
		EnableContextRelevance: true,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if report.Metrics.Faithfulness == 0 {
		t.Fatalf("expected Faithfulness != 0, got %v", report.Metrics.Faithfulness)
	}
	if report.Metrics.AnswerRelevance == 0 {
		t.Fatalf("expected AnswerRelevance != 0, got %v", report.Metrics.AnswerRelevance)
	}
	if report.Metrics.ContextRelevance == 0 {
		t.Fatalf("expected ContextRelevance != 0, got %v", report.Metrics.ContextRelevance)
	}
}
