package reranker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func testChunks() []domain.RetrievedChunk {
	return []domain.RetrievedChunk{
		{Chunk: domain.Chunk{ID: "a", Content: "alpha"}, Score: 0.1},
		{Chunk: domain.Chunk{ID: "b", Content: "beta"}, Score: 0.2},
		{Chunk: domain.Chunk{ID: "c", Content: "gamma"}, Score: 0.3},
	}
}

// @sk-test reranker-cross-encoder#T4.1: empty key → ErrInvalidRerankerConfig (AC-003)
func TestNewCohereRerank_EmptyKey(t *testing.T) {
	_, err := NewCohereRerank(CohereRerankOptions{APIKey: ""})
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if !strings.Contains(err.Error(), "APIKey") {
		t.Fatalf("expected error about APIKey, got: %v", err)
	}
	if !strings.Contains(err.Error(), ErrInvalidRerankerConfig.Error()) {
		t.Fatalf("expected ErrInvalidRerankerConfig wrapping, got: %v", err)
	}
}

// @sk-test reranker-cross-encoder#T4.1: invalid BaseURL → error (AC-003)
func TestNewCohereRerank_InvalidBaseURL(t *testing.T) {
	_, err := NewCohereRerank(CohereRerankOptions{
		APIKey:  "valid-key",
		BaseURL: "://invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid BaseURL")
	}
}

// @sk-test reranker-cross-encoder#T4.1: defaults applied correctly (AC-003)
func TestNewCohereRerank_Defaults(t *testing.T) {
	r, err := NewCohereRerank(CohereRerankOptions{APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	if r.opts.Model != defaultCohereModel {
		t.Fatalf("expected model %s, got %s", defaultCohereModel, r.opts.Model)
	}
	if r.opts.BaseURL != defaultCohereBaseURL {
		t.Fatalf("expected base URL %s, got %s", defaultCohereBaseURL, r.opts.BaseURL)
	}
	if r.opts.MaxRetries != defaultMaxRetries {
		t.Fatalf("expected max retries %d, got %d", defaultMaxRetries, r.opts.MaxRetries)
	}
}

// @sk-test reranker-cross-encoder#T4.1: empty chunks → no-op (AC-002)
func TestCohereRerank_EmptyChunks(t *testing.T) {
	r, err := NewCohereRerank(CohereRerankOptions{APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.Rerank(context.Background(), "query", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d chunks", len(result))
	}
}

// @sk-test reranker-cross-encoder#T4.1: len(out) == len(in) invariant (AC-007)
func TestCohereRerank_NoFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(cohereRerankResponse{
			Results: []cohereRerankResult{
				{Index: 2, RelevanceScore: 0.9},
				{Index: 0, RelevanceScore: 0.5},
				{Index: 1, RelevanceScore: 0.3},
			},
		})
	}))
	defer ts.Close()

	r, err := NewCohereRerank(CohereRerankOptions{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	chunks := testChunks()
	result, err := r.Rerank(context.Background(), "query", chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != len(chunks) {
		t.Fatalf("expected %d chunks, got %d", len(chunks), len(result))
	}
}

// @sk-test reranker-cross-encoder#T4.1: Cohere re-ranking via mock HTTP (AC-001)
func TestCohereRerank_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req cohereRerankRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Query != "test query" {
			http.Error(w, fmt.Sprintf("unexpected query: %s", req.Query), http.StatusBadRequest)
			return
		}
		if len(req.Documents) != 3 {
			http.Error(w, fmt.Sprintf("expected 3 documents, got %d", len(req.Documents)), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(cohereRerankResponse{
			Results: []cohereRerankResult{
				{Index: 2, RelevanceScore: 0.9},
				{Index: 0, RelevanceScore: 0.5},
				{Index: 1, RelevanceScore: 0.3},
			},
		})
	}))
	defer ts.Close()

	r, err := NewCohereRerank(CohereRerankOptions{
		APIKey:  "test-key",
		BaseURL: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	chunks := testChunks()
	result, err := r.Rerank(context.Background(), "test query", chunks)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	if result[0].Chunk.ID != "c" {
		t.Fatalf("expected first chunk 'c' (highest score), got '%s'", result[0].Chunk.ID)
	}
	if result[1].Chunk.ID != "a" {
		t.Fatalf("expected second chunk 'a', got '%s'", result[1].Chunk.ID)
	}
	if result[2].Chunk.ID != "b" {
		t.Fatalf("expected third chunk 'b', got '%s'", result[2].Chunk.ID)
	}
	if result[0].Score != 0.9 {
		t.Fatalf("expected score 0.9 for first chunk, got %f", result[0].Score)
	}
}

// @sk-test reranker-cross-encoder#T4.1: 401 → error with code (AC-006)
func TestCohereRerank_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer ts.Close()

	r, err := NewCohereRerank(CohereRerankOptions{
		APIKey:     "bad-key",
		BaseURL:    ts.URL,
		MaxRetries: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Rerank(context.Background(), "query", testChunks())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected error to contain 401, got: %v", err)
	}
}

// @sk-test reranker-cross-encoder#T4.1: concurrent batch fan-out (AC-008)
func TestCohereRerank_BatchFanOut(t *testing.T) {
	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount.Add(1)
		time.Sleep(50 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(cohereRerankResponse{
			Results: []cohereRerankResult{
				{Index: 0, RelevanceScore: 1.0},
				{Index: 1, RelevanceScore: 0.5},
				{Index: 2, RelevanceScore: 0.0},
			},
		})
	}))
	defer ts.Close()

	r, err := NewCohereRerank(CohereRerankOptions{
		APIKey:     "test-key",
		BaseURL:    ts.URL,
		MaxRetries: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	chunks := testChunks()
	queries := []string{"q1", "q2", "q3", "q4", "q5"}

	start := time.Now()
	results, err := r.RerankBatch(context.Background(), queries, chunks)
	duration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	if callCount.Load() != 5 {
		t.Fatalf("expected 5 calls, got %d", callCount.Load())
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 result sets, got %d", len(results))
	}
	if duration > 150*time.Millisecond {
		t.Fatalf("expected concurrent execution < 150ms, got %v", duration)
	}
}

// @sk-test reranker-cross-encoder#T4.1: batch with empty chunks (AC-008)
func TestCohereRerank_BatchEmptyChunks(t *testing.T) {
	r, err := NewCohereRerank(CohereRerankOptions{APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	results, err := r.RerankBatch(context.Background(), []string{"q1", "q2"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 result sets, got %d", len(results))
	}
}

// @sk-test reranker-cross-encoder#T4.1: batch with empty queries (AC-008)
func TestCohereRerank_BatchEmptyQueries(t *testing.T) {
	r, err := NewCohereRerank(CohereRerankOptions{APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	results, err := r.RerankBatch(context.Background(), nil, testChunks())
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil for empty queries, got %v", results)
	}
}
