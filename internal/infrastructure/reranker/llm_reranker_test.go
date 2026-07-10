package reranker

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type mockLLM struct {
	reply string
	err   error
}

func (m *mockLLM) Health(_ context.Context) error { return nil }
func (m *mockLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return m.reply, m.err
}

type mockLLMWithCapture struct {
	fn      func(system, user string) (string, error)
	counter atomic.Int64
}

func (m *mockLLMWithCapture) Health(_ context.Context) error { return nil }
func (m *mockLLMWithCapture) Generate(_ context.Context, system, user string) (string, error) {
	m.counter.Add(1)
	return m.fn(system, user)
}

func makeChunks(contents ...string) []domain.RetrievedChunk {
	chunks := make([]domain.RetrievedChunk, len(contents))
	for i, c := range contents {
		chunks[i] = domain.RetrievedChunk{
			Chunk: domain.Chunk{
				ID:       string(rune('a' + i)),
				Content:  c,
				ParentID: "doc1",
			},
			Score: float64(len(contents) - i),
		}
	}
	return chunks
}

// @sk-test reranker-llm-based#T1.3: TestLLMReranker_Rerank_ScoresSet (AC-001)

// TestLLMReranker_Rerank_ScoresSet проверяет, что после rerank у каждого чанка установлен Score.
func TestLLMReranker_Rerank_ScoresSet(t *testing.T) {
	llm := &mockLLM{reply: `[8, 3, 9]`}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("relevant document", "unrelated info", "highly relevant result")
	result, err := r.Rerank(context.Background(), "test query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	for i, ch := range result {
		if ch.Score < 0 || ch.Score > 1 {
			t.Fatalf("chunk %d: score %f out of [0,1]", i, ch.Score)
		}
	}
}

// @sk-test reranker-llm-based#T1.3: TestLLMReranker_Rerank_Order (AC-002)

// TestLLMReranker_Rerank_Order проверяет сортировку по убыванию LLM-score.
func TestLLMReranker_Rerank_Order(t *testing.T) {
	llm := &mockLLM{reply: `[3, 9, 5]`}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("low relevance", "high relevance", "medium relevance")
	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if result[0].Score < result[1].Score || result[1].Score < result[2].Score {
		t.Fatalf("expected descending order, got scores: %f, %f, %f",
			result[0].Score, result[1].Score, result[2].Score)
	}
}

// @sk-test reranker-llm-based#T1.3: TestLLMReranker_Rerank_GracefulDegradation (AC-004)

// TestLLMReranker_Rerank_GracefulDegradation проверяет возврат исходных чанков при ошибке LLM.
func TestLLMReranker_Rerank_GracefulDegradation(t *testing.T) {
	llm := &mockLLM{reply: "", err: errors.New("llm unavailable")}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("doc a", "doc b", "doc c")
	originalScores := make([]float64, len(chunks))
	for i, ch := range chunks {
		originalScores[i] = ch.Score
	}

	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal("expected no error on graceful degradation, got:", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	for i, ch := range result {
		if ch.Score != originalScores[i] {
			t.Fatalf("chunk %d: expected original score %f, got %f", i, originalScores[i], ch.Score)
		}
	}
}

// @sk-test reranker-llm-based#T1.3: TestLLMReranker_Rerank_BatchScoring (AC-005)

// TestLLMReranker_Rerank_BatchScoring проверяет, что при batchSize >= N выполняется 1 LLM-вызов.
func TestLLMReranker_Rerank_BatchScoring(t *testing.T) {
	mock := &mockLLMWithCapture{
		fn: func(_, _ string) (string, error) {
			return `[9, 7, 5, 3, 1]`, nil
		},
	}
	r, err := NewLLMReranker(mock, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("a", "b", "c", "d", "e")
	_, err = r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if calls := mock.counter.Load(); calls != 1 {
		t.Fatalf("expected 1 LLM call, got %d", calls)
	}
}

// @sk-test reranker-llm-based#T1.3: TestLLMReranker_Rerank_EmptyChunks (AC-004)

// TestLLMReranker_Rerank_EmptyChunks проверяет пустой список чанков.
func TestLLMReranker_Rerank_EmptyChunks(t *testing.T) {
	llm := &mockLLM{reply: `[]`}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	result, err := r.Rerank(context.Background(), "query", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 chunks, got %d", len(result))
	}
}

// @sk-test reranker-llm-based#T1.3: TestLLMReranker_Rerank_SingleChunk (AC-001)

// TestLLMReranker_Rerank_SingleChunk проверяет один чанк.
func TestLLMReranker_Rerank_SingleChunk(t *testing.T) {
	llm := &mockLLM{reply: `[7]`}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("single doc")
	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0].Score != 0.7 {
		t.Fatalf("expected score 0.7, got %f", result[0].Score)
	}
}

// @sk-test reranker-llm-based#T1.3,T2.1: TestLLMReranker_Rerank_CustomPrompt (AC-003)

// TestLLMReranker_Rerank_CustomPrompt проверяет передачу кастомного system prompt.
func TestLLMReranker_Rerank_CustomPrompt(t *testing.T) {
	var capturedSystem string
	mock := &mockLLMWithCapture{
		fn: func(system, _ string) (string, error) {
			capturedSystem = system
			return `[5, 5]`, nil
		},
	}
	r, err := NewLLMReranker(mock, "Custom judge prompt: rate by domain expertise", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("doc a", "doc b")
	_, err = r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedSystem, "Custom judge prompt") {
		t.Fatalf("expected custom prompt, got: %s", capturedSystem)
	}
}

// @sk-test reranker-llm-based#T1.3: TestParseScores (AC-001)

// @sk-test reranker-llm-based#T2.2: TestLLMReranker_RerankBatch (AC-006)

// TestLLMReranker_RerankBatch проверяет RerankBatch с несколькими query.
func TestLLMReranker_RerankBatch(t *testing.T) {
	mock := &mockLLMWithCapture{
		fn: func(_, _ string) (string, error) {
			return `[9, 3, 6]`, nil
		},
	}
	r, err := NewLLMReranker(mock, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("doc a", "doc b", "doc c")
	queries := []string{"query one", "query two"}

	results, err := r.RerankBatch(context.Background(), queries, chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, res := range results {
		if len(res) != 3 {
			t.Fatalf("result %d: expected 3 chunks, got %d", i, len(res))
		}
		if res[0].Score < res[1].Score || res[1].Score < res[2].Score {
			t.Fatalf("result %d: not sorted descending: %f, %f, %f", i, res[0].Score, res[1].Score, res[2].Score)
		}
	}
}

// @sk-test reranker-llm-based#T2.3: TestLLMReranker_Rerank_RetryThenSuccess (AC-007)

// TestLLMReranker_Rerank_RetryThenSuccess проверяет retry: ошибка 2 раза, успех на 3-й.
func TestLLMReranker_Rerank_RetryThenSuccess(t *testing.T) {
	var attempt atomic.Int64
	mock := &mockLLMWithCapture{
		fn: func(_, _ string) (string, error) {
			cur := attempt.Add(1)
			if cur <= 2 {
				return "", errors.New("temporary error")
			}
			return `[9, 5, 7]`, nil
		},
	}

	r, err := NewLLMReranker(mock, "", 10, 2)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("a", "b", "c")
	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal("expected success after retry, got:", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	if calls := mock.counter.Load(); calls != 3 {
		t.Fatalf("expected 3 LLM calls (1 initial + 2 retries), got %d", calls)
	}
}

// @sk-test reranker-llm-based#T2.3: TestLLMReranker_Rerank_RetryExhausted (AC-007)

// TestLLMReranker_Rerank_RetryExhausted проверяет graceful degradation после исчерпания retry.
func TestLLMReranker_Rerank_RetryExhausted(t *testing.T) {
	mock := &mockLLMWithCapture{
		fn: func(_, _ string) (string, error) {
			return "", errors.New("persistent error")
		},
	}

	r, err := NewLLMReranker(mock, "", 10, 2)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("a", "b", "c")
	originalScores := make([]float64, len(chunks))
	for i, ch := range chunks {
		originalScores[i] = ch.Score
	}

	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal("expected graceful degradation, not error, got:", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	for i, ch := range result {
		if ch.Score != originalScores[i] {
			t.Fatalf("chunk %d: expected original score %f, got %f", i, originalScores[i], ch.Score)
		}
	}
	if calls := mock.counter.Load(); calls != 3 {
		t.Fatalf("expected 3 LLM calls (1 initial + 2 retries), got %d", calls)
	}
}

func TestParseScores(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"valid", "[8, 3, 9]", 3, false},
		{"empty array", "[]", 0, false},
		{"out of bounds", "[15, -2, 5]", 3, false},
		{"wrong length", "[1, 2]", 3, true},
		{"invalid json", "not json", 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores, err := parseScores(tt.input, tt.expected)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(scores) != tt.expected {
				t.Fatalf("expected %d scores, got %d", tt.expected, len(scores))
			}
			for _, s := range scores {
				if s < 0 || s > 1 {
					t.Fatalf("score %f out of [0,1]", s)
				}
			}
		})
	}
}

// @sk-test reranker-llm-based#T3.1: TestLLMReranker_Rerank_AllScoresZero (AC-004)

// TestLLMReranker_Rerank_AllScoresZero проверяет, что при всех score=0 сохраняется исходный порядок.
func TestLLMReranker_Rerank_AllScoresZero(t *testing.T) {
	llm := &mockLLM{reply: `[0, 0, 0]`}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("c doc", "a doc", "b doc")
	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	// При равных score (все 0) порядок сохраняется исходный (sort stable).
	if result[0].Chunk.Content != "c doc" || result[1].Chunk.Content != "a doc" || result[2].Chunk.Content != "b doc" {
		t.Fatalf("expected original order, got: %s, %s, %s",
			result[0].Chunk.Content, result[1].Chunk.Content, result[2].Chunk.Content)
	}
}

// @sk-test reranker-llm-based#T3.1: TestLLMReranker_Rerank_UnparseableResponse (AC-001)

// TestLLMReranker_Rerank_UnparseableResponse проверяет непарсимый ответ LLM → score=0.
func TestLLMReranker_Rerank_UnparseableResponse(t *testing.T) {
	llm := &mockLLM{reply: `this is not json`}
	r, err := NewLLMReranker(llm, "", 10, 1)
	if err != nil {
		t.Fatal(err)
	}

	chunks := makeChunks("doc a", "doc b", "doc c")
	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	for i, ch := range result {
		if ch.Score != 0 {
			t.Fatalf("chunk %d: expected score 0 for unparseable response, got %f", i, ch.Score)
		}
	}
}
