package draftrag

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// reverseReranker переворачивает порядок чанков (для проверки что reranker вызывается).
type reverseReranker struct{ called bool }

func (r *reverseReranker) Rerank(_ context.Context, _ string, chunks []domain.RetrievedChunk) ([]domain.RetrievedChunk, error) {
	r.called = true
	out := make([]domain.RetrievedChunk, len(chunks))
	for i, c := range chunks {
		out[len(chunks)-1-i] = c
	}
	return out, nil
}

// countReranker считает вызовы Rerank и НЕ реализует BatchReranker (для проверки fallback).
type countReranker struct {
	callCount atomic.Int32
}

func (c *countReranker) Rerank(_ context.Context, _ string, chunks []domain.RetrievedChunk) ([]domain.RetrievedChunk, error) {
	c.callCount.Add(1)
	return chunks, nil
}

func TestPipeline_Reranker_IsCalled(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "ok"}
	rr := &reverseReranker{}

	p, err := NewPipelineWithOptions(store, llm, emb, PipelineOptions{
		Reranker: rr,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "a", Content: "alpha", ParentID: "d1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "b", Content: "beta", ParentID: "d2",
		Embedding: []float64{0.9, 0.1, 0}, Position: 0,
	})

	result, err := p.Search("test").TopK(2).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !rr.called {
		t.Fatal("reranker was not called")
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected chunks after reranking")
	}
}

// @sk-test reranker-cross-encoder#T4.1: fallback to sequential Rerank when BatchReranker not implemented (AC-009)
func TestPipeline_Reranker_Fallback(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "variant1\nvariant2"}
	counter := &countReranker{}

	p, err := NewPipelineWithOptions(store, llm, emb, PipelineOptions{
		Reranker: counter,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{ID: "a", Content: "alpha", ParentID: "d1", Embedding: []float64{1, 0, 0}, Position: 0})
	_ = store.Upsert(ctx, domain.Chunk{ID: "b", Content: "beta", ParentID: "d1", Embedding: []float64{1, 0, 0}, Position: 1})
	_ = store.Upsert(ctx, domain.Chunk{ID: "c", Content: "gamma", ParentID: "d1", Embedding: []float64{1, 0, 0}, Position: 2})

	_, err = p.Search("test question").MultiQuery(2).TopK(3).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counter.callCount.Load() == 0 {
		t.Fatal("reranker was not called during multi-query fallback")
	}
}

func TestPipeline_NoReranker_Works(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "ok"}
	p, err := NewPipeline(store, llm, emb)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "a", Content: "alpha", ParentID: "d1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	result, err := p.Search("test").TopK(1).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected chunks")
	}
}
