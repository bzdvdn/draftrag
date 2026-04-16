package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type fixedChunker struct {
	chunks []domain.Chunk
}

func (c fixedChunker) Chunk(ctx context.Context, _ domain.Document) ([]domain.Chunk, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.chunks, nil
}

type countingEmbedder struct {
	calls int
}

func (e *countingEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	e.calls++
	return []float64{1, float64(e.calls)}, nil
}

func TestPipeline_Index_UsesChunker_UpsertsMultipleChunks(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &countingEmbedder{}

	ch := fixedChunker{
		chunks: []domain.Chunk{
			{ID: "doc-1:0", ParentID: "doc-1", Position: 0, Content: "a"},
			{ID: "doc-1:1", ParentID: "doc-1", Position: 1, Content: "b"},
		},
	}

	p := NewPipelineWithChunker(store, testLLM{}, emb, ch)

	err := p.Index(context.Background(), []domain.Document{{ID: "doc-1", Content: "ignored"}})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if emb.calls != 2 {
		t.Fatalf("expected 2 Embed calls, got %d", emb.calls)
	}

	res, err := store.Search(context.Background(), []float64{1, 1}, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.TotalFound != 2 || len(res.Chunks) != 2 {
		t.Fatalf("expected 2 chunks in store, got total=%d len=%d", res.TotalFound, len(res.Chunks))
	}
}

func TestPipeline_Index_BackwardCompatibility_OneChunkPerDoc(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &countingEmbedder{}

	p := NewPipeline(store, testLLM{}, emb)

	err := p.Index(context.Background(), []domain.Document{{ID: "doc-1", Content: "cat"}})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if emb.calls != 1 {
		t.Fatalf("expected 1 Embed call, got %d", emb.calls)
	}
}

func TestPipeline_Index_ContextCancelFast(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &countingEmbedder{}

	ch := fixedChunker{
		chunks: []domain.Chunk{
			{ID: "doc-1:0", ParentID: "doc-1", Position: 0, Content: "a"},
			{ID: "doc-1:1", ParentID: "doc-1", Position: 1, Content: "b"},
		},
	}

	p := NewPipelineWithChunker(store, testLLM{}, emb, ch)

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	err := p.Index(ctx, []domain.Document{{ID: "doc-1", Content: strings.Repeat("x", 100)}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}
