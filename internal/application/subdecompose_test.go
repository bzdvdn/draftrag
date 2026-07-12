package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test sub-query-decomposition#T2.3: recordingSubDecomposer — mock QueryDecomposer для тестов (AC-002)
type recordingSubDecomposer struct {
	subQueries []string
	err        error
	callCount  int
	mu         sync.Mutex
}

func (m *recordingSubDecomposer) Decompose(_ context.Context, query string) ([]string, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return m.subQueries, m.err
}

func makeTestResult(ids ...string) domain.RetrievalResult {
	var chunks []domain.RetrievedChunk
	for i, id := range ids {
		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: domain.Chunk{ID: id, Content: "content " + id},
			Score: float64(len(ids)-i) * 0.1,
		})
	}
	return domain.RetrievalResult{Chunks: chunks}
}

// @sk-test sub-query-decomposition#T2.3: TestPipeline_QuerySubDecompose_MultipleSubQueries (AC-001, AC-002, AC-007)
func TestPipeline_QuerySubDecompose_MultipleSubQueries(t *testing.T) {
	var calls []string
	store := &recordingStore{
		calls:  &calls,
		result: makeTestResult("c1", "c2"),
	}
	p, err := NewPipeline(store, &mockLLMProvider{}, &mockEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: []string{"cat", "dog"},
	}

	start := time.Now()
	result, err := p.QuerySubDecompose(context.Background(), "test query", 5, decomposer)
	if err != nil {
		t.Fatal(err)
	}
	_ = time.Since(start)

	if result.QueryText != "test query" {
		t.Fatalf("expected QueryText 'test query', got %q", result.QueryText)
	}

	if len(calls) < 2 {
		t.Fatalf("expected at least 2 search calls for 2 sub-queries, got %d", len(calls))
	}
}

// @sk-test sub-query-decomposition#T2.3: TestPipeline_QuerySubDecompose_SingleSubQuery (AC-002)
func TestPipeline_QuerySubDecompose_SingleSubQuery(t *testing.T) {
	var calls []string
	store := &recordingStore{
		calls:  &calls,
		result: makeTestResult("c1"),
	}
	p, err := NewPipeline(store, &mockLLMProvider{}, &mockEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: []string{"cat"},
	}

	_, err = p.QuerySubDecompose(context.Background(), "test query", 5, decomposer)
	if err != nil {
		t.Fatal(err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 search call for single sub-query, got %d", len(calls))
	}
}

// @sk-test sub-query-decomposition#T2.3: TestPipeline_QuerySubDecompose_DecomposerError_Fallback (AC-005)
func TestPipeline_QuerySubDecompose_DecomposerError_Fallback(t *testing.T) {
	var calls []string
	store := &recordingStore{
		calls:  &calls,
		result: makeTestResult("c1"),
	}
	p, err := NewPipeline(store, &mockLLMProvider{}, &mockEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: nil,
		err:        errors.New("decomposer error"),
	}

	result, err := p.QuerySubDecompose(context.Background(), "test query", 5, decomposer)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results from fallback query")
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 search call (fallback), got %d", len(calls))
	}
}

// @sk-test sub-query-decomposition#T2.3: TestPipeline_QuerySubDecompose_EmptySubQueries_Fallback (AC-005)
func TestPipeline_QuerySubDecompose_EmptySubQueries_Fallback(t *testing.T) {
	var calls []string
	store := &recordingStore{
		calls:  &calls,
		result: makeTestResult("c1"),
	}
	p, err := NewPipeline(store, &mockLLMProvider{}, &mockEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: []string{},
	}

	result, err := p.QuerySubDecompose(context.Background(), "test query", 5, decomposer)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results from fallback query")
	}
}

// @sk-test sub-query-decomposition#T2.3: TestPipeline_QuerySubDecompose_ContextCancellation (AC-007)
func TestPipeline_QuerySubDecompose_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls []string
	store := &recordingStore{
		calls:  &calls,
		result: makeTestResult("c1"),
	}
	p, err := NewPipeline(store, &mockLLMProvider{}, &mockEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: []string{"cat", "dog"},
	}

	_, err = p.QuerySubDecompose(ctx, "test query", 5, decomposer)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// @sk-test sub-query-decomposition#T4.1: TestPipeline_QuerySubDecompose_MergeDedup (AC-004)
func TestPipeline_QuerySubDecompose_MergeDedup(t *testing.T) {
	result1 := domain.RetrievalResult{Chunks: []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "c1", Content: "common"}, Score: 0.9},
		{Chunk: domain.Chunk{ID: "c2", Content: "unique1"}, Score: 0.8},
	}}
	result2 := domain.RetrievalResult{Chunks: []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "c1", Content: "common"}, Score: 0.7},
		{Chunk: domain.Chunk{ID: "c3", Content: "unique2"}, Score: 0.6},
	}}

	merged := mergeSubResults([]domain.RetrievalResult{result1, result2}, 10)
	if len(merged.Chunks) != 3 {
		t.Fatalf("expected 3 unique chunks after merge, got %d", len(merged.Chunks))
	}

	// c1 должен быть с max score (0.9)
	for _, rc := range merged.Chunks {
		if rc.Chunk.ID == "c1" && rc.Score != 0.9 {
			t.Fatalf("expected c1 score 0.9, got %f", rc.Score)
		}
	}
}

// @sk-test sub-query-decomposition#T4.1: TestPipeline_AnswerSubDecompose_Integration (AC-008)
func TestPipeline_AnswerSubDecompose_Integration(t *testing.T) {
	var calls []string
	llm := &recordingLLM{calls: &calls}

	p, err := NewPipeline(
		&recordingStore{
			calls: &calls,
			result: domain.RetrievalResult{
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ID: "c1", Content: "test content"}},
				},
			},
		},
		llm,
		recordingEmbedder{calls: &calls},
	)
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: []string{"sub-q1"},
	}

	answer, err := p.AnswerSubDecompose(context.Background(), "test question", 5, decomposer)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "answer" {
		t.Fatalf("expected 'answer', got %q", answer)
	}
	if len(calls) == 0 {
		t.Fatal("expected at least one call")
	}
}

// @sk-test sub-query-decomposition#T4.1: TestPipeline_AnswerSubDecomposeWithCitations (AC-008, AC-009)
func TestPipeline_AnswerSubDecomposeWithCitations(t *testing.T) {
	var calls []string
	llm := &recordingLLM{calls: &calls}

	p, err := NewPipeline(
		&recordingStore{
			calls: &calls,
			result: domain.RetrievalResult{
				Chunks: []domain.RetrievedChunk{
					{Chunk: domain.Chunk{ID: "c1", Content: "test"}},
				},
			},
		},
		llm,
		recordingEmbedder{calls: &calls},
	)
	if err != nil {
		t.Fatal(err)
	}

	decomposer := &recordingSubDecomposer{
		subQueries: []string{"sub-q1"},
	}

	answer, sources, err := p.AnswerSubDecomposeWithCitations(context.Background(), "test question", 5, decomposer)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "answer" {
		t.Fatalf("expected 'answer', got %q", answer)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected non-empty sources")
	}
}
