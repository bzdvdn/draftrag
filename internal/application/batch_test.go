package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type mockBatchStore struct {
	chunks []domain.Chunk
	mu     sync.Mutex
}

func (m *mockBatchStore) Health(_ context.Context) error { return nil }
func (m *mockBatchStore) Upsert(_ context.Context, chunk domain.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunks = append(m.chunks, chunk)
	return nil
}

func (m *mockBatchStore) Delete(_ context.Context, _ string) error { return nil }
func (m *mockBatchStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

type fixedBatchEmbedder struct {
	delay time.Duration
}

func (f fixedBatchEmbedder) Health(_ context.Context) error { return nil }
func (f fixedBatchEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

type countingBatchEmbedder struct {
	fixedBatchEmbedder
	count atomic.Int32
}

func (c *countingBatchEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	c.count.Add(1)
	return c.fixedBatchEmbedder.Embed(ctx, text)
}

type erroringBatchEmbedder struct {
	fixedBatchEmbedder
	errorText string
}

func (e erroringBatchEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if text == e.errorText {
		return nil, errors.New("embed failed")
	}
	return e.fixedBatchEmbedder.Embed(ctx, text)
}

type mockChunker struct{}

func (m mockChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return []domain.Chunk{
		{ID: doc.ID + "#0", Content: doc.Content + " part1", ParentID: doc.ID, Position: 0},
		{ID: doc.ID + "#1", Content: doc.Content + " part2", ParentID: doc.ID, Position: 1},
	}, nil
}

type okBatchLLM struct{}

func (okBatchLLM) Health(_ context.Context) error { return nil }
func (okBatchLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "ok", nil
}

func TestPipeline_IndexBatch_ParallelProcessing(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := &countingBatchEmbedder{
		fixedBatchEmbedder: fixedBatchEmbedder{delay: 50 * time.Millisecond},
	}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{
			IndexConcurrency:    5,
			IndexBatchRateLimit: 1000,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := make([]domain.Document, 10)
	for i := 0; i < 10; i++ {
		docs[i] = domain.Document{ID: "doc" + string(rune('0'+i)), Content: "content " + string(rune('0'+i))}
	}

	start := time.Now()
	result, err := p.IndexBatch(context.Background(), docs, 0)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected parallel processing (< 500ms), got %v", elapsed)
	}

	if result.ProcessedCount != 10 {
		t.Fatalf("expected 10 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 10 {
		t.Fatalf("expected 10 successful docs, got %d", len(result.Successful))
	}

	if len(store.chunks) != 10 {
		t.Fatalf("expected 10 chunks in store, got %d", len(store.chunks))
	}

	if embedder.count.Load() != 10 {
		t.Fatalf("expected 10 embed calls, got %d", embedder.count.Load())
	}
}

func TestPipeline_IndexBatch_RateLimiting(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{
			IndexConcurrency:    10,
			IndexBatchRateLimit: 10,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := make([]domain.Document, 20)
	for i := 0; i < 20; i++ {
		docs[i] = domain.Document{ID: "doc" + string(rune('0'+i%10)), Content: "content"}
	}

	start := time.Now()
	result, err := p.IndexBatch(context.Background(), docs, 0)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed < 1900*time.Millisecond {
		t.Fatalf("expected rate limiting (>= 1.9s for 20 docs at 10/sec), got %v", elapsed)
	}

	if result.ProcessedCount != 20 {
		t.Fatalf("expected 20 processed docs, got %d", result.ProcessedCount)
	}
}

func TestPipeline_IndexBatch_PartialErrors(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := erroringBatchEmbedder{
		errorText: "error content",
	}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{IndexConcurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{ID: "doc1", Content: "ok content 1"},
		{ID: "doc2", Content: "error content"},
		{ID: "doc3", Content: "ok content 2"},
		{ID: "doc4", Content: "error content"},
		{ID: "doc5", Content: "ok content 3"},
	}

	result, err := p.IndexBatch(context.Background(), docs, 0)

	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}

	if result.ProcessedCount != 5 {
		t.Fatalf("expected 5 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 3 {
		t.Fatalf("expected 3 successful docs, got %d", len(result.Successful))
	}

	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(result.Errors))
	}

	for _, err := range result.Errors {
		if err.DocumentID == "" {
			t.Fatalf("expected non-empty DocumentID in error")
		}
		if err.Error == nil {
			t.Fatalf("expected non-nil Error in IndexBatchError")
		}
	}

	successfulIDs := make(map[string]bool)
	for _, doc := range result.Successful {
		successfulIDs[doc.ID] = true
	}
	if !successfulIDs["doc1"] || !successfulIDs["doc3"] || !successfulIDs["doc5"] {
		t.Fatalf("expected doc1, doc3, doc5 to be successful")
	}
}

func TestPipeline_IndexBatch_ContextCancellation(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{delay: 200 * time.Millisecond}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{
			IndexConcurrency:    5,
			IndexBatchRateLimit: 1000,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	docs := make([]domain.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = domain.Document{ID: "doc" + string(rune('0'+i%10)), Content: "content"}
	}

	result, err := p.IndexBatch(ctx, docs, 0)

	if err == nil {
		t.Fatalf("expected error from cancelled context, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error, got %v", err)
	}

	if result == nil {
		t.Fatalf("expected partial result, got nil")
	}

	if result.ProcessedCount != len(result.Successful)+len(result.Errors) {
		t.Fatalf("invariant violated: ProcessedCount=%d, Successful=%d, Errors=%d",
			result.ProcessedCount, len(result.Successful), len(result.Errors))
	}
}

func TestPipeline_IndexBatch_WithChunker(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{
			IndexConcurrency: 2,
			Chunker:          mockChunker{},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{ID: "doc1", Content: "content1"},
		{ID: "doc2", Content: "content2"},
		{ID: "doc3", Content: "content3"},
	}

	result, err := p.IndexBatch(context.Background(), docs, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcessedCount != 3 {
		t.Fatalf("expected 3 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 3 {
		t.Fatalf("expected 3 successful docs, got %d", len(result.Successful))
	}

	if len(store.chunks) != 6 {
		t.Fatalf("expected 6 chunks (3 docs * 2 chunks each), got %d", len(store.chunks))
	}

	for _, chunk := range store.chunks {
		if chunk.ID == "" {
			t.Fatalf("expected non-empty chunk ID")
		}
		if chunk.ParentID == "" {
			t.Fatalf("expected non-empty chunk ParentID")
		}
		if chunk.Embedding == nil {
			t.Fatalf("expected non-nil chunk Embedding")
		}
	}
}

func TestPipeline_IndexBatch_EmptyDocs(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{IndexConcurrency: 4},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := p.IndexBatch(context.Background(), []domain.Document{}, 0)

	if err != nil {
		t.Fatalf("unexpected error for empty docs: %v", err)
	}

	if result.ProcessedCount != 0 {
		t.Fatalf("expected 0 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 0 {
		t.Fatalf("expected 0 successful docs, got %d", len(result.Successful))
	}

	if len(result.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(result.Errors))
	}
}

func TestPipeline_IndexBatch_InvalidDocument(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p, err := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineOptions{IndexConcurrency: 2},
	)
	if err != nil {
		t.Fatal(err)
	}

	docs := []domain.Document{
		{ID: "doc1", Content: "valid content"},
		{ID: "", Content: "invalid doc - no ID"},
		{ID: "doc3", Content: "valid content"},
	}

	result, err := p.IndexBatch(context.Background(), docs, 0)

	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}

	if result.ProcessedCount != 3 {
		t.Fatalf("expected 3 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 2 {
		t.Fatalf("expected 2 successful docs, got %d", len(result.Successful))
	}

	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].Error == nil {
		t.Fatalf("expected validation error, got nil")
	}
}
