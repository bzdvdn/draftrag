package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/llm"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test hardening-2026q2#T3.3: Stream methods (Stream, StreamSources, StreamCite)
func TestSearchBuilder_Stream_EmptyQuestion(t *testing.T) {
	p, _ := setupPipeline(t)
	_, err := p.Search("  ").TopK(5).Stream(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearchBuilder_StreamSources_EmptyQuestion(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, err := p.Search("  ").TopK(5).StreamSources(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearchBuilder_StreamCite_EmptyQuestion(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, _, err := p.Search("  ").TopK(5).StreamCite(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearchBuilder_Stream_UsesStreamingLLM(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"hello", " ", "world"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, err := p.Search("Go").TopK(5).Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestSearchBuilder_StreamSources_UsesStreamingLLM(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, err := p.Search("Go").TopK(5).StreamSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

func TestSearchBuilder_StreamCite_UsesStreamingLLM(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, citations, err := p.Search("Go").TopK(5).StreamCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

func TestSearchBuilder_Cite_EmptyQuestion(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, err := p.Search("  ").TopK(5).Cite(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearchBuilder_InlineCite_EmptyQuestion(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, _, err := p.Search("  ").TopK(5).InlineCite(context.Background())
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearchBuilder_InlineCite_WithResults(t *testing.T) {
	p, _ := setupPipeline(t)
	answer, sources, citations, err := p.Search("Go").TopK(2).InlineCite(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	if len(citations) == 0 {
		t.Fatal("expected citations")
	}
}

func TestSearchBuilder_Hybrid_InvalidConfig(t *testing.T) {
	p, _ := setupPipeline(t)
	_, err := p.Search("Go").TopK(5).Hybrid(HybridConfig{}).Retrieve(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid hybrid config")
	}
}

func TestSearchBuilder_Stream_InvalidTopK(t *testing.T) {
	p, _ := setupPipeline(t)
	_, err := p.Search("Go").TopK(0).Stream(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

func TestSearchBuilder_StreamSources_InvalidTopK(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, err := p.Search("Go").TopK(-1).StreamSources(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

func TestSearchBuilder_StreamCite_InvalidTopK(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, _, err := p.Search("Go").TopK(0).StreamCite(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: Stream routing — HyDE
func TestSearchBuilder_Stream_HyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens:         []string{"hello", " ", "world"},
		GenerateResult: "hypothetical document",
	}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, err := p.Search("q").TopK(5).HyDE().Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

// @sk-test hardening-2026q2#AC-007: Stream routing — MultiQuery
func TestSearchBuilder_Stream_MultiQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens:         []string{"answer"},
		GenerateResult: "Go concurrency",
	}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, err := p.Search("q").TopK(5).MultiQuery(2).Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
}

// @sk-test hardening-2026q2#AC-007: Stream routing — Hybrid (not supported on InMemoryStore)
func TestSearchBuilder_Stream_Hybrid_NotSupported(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), &llm.MockStreamingLLM{}, &fixedEmbedder{})
	_, err := p.Search("q").TopK(5).Hybrid(HybridConfig{SemanticWeight: 0.5, RRFK: 60}).Stream(context.Background())
	if !errors.Is(err, ErrHybridNotSupported) {
		t.Fatalf("expected ErrHybridNotSupported, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: Stream routing — Filter (success path)
func TestSearchBuilder_Stream_Filter(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
		Metadata:  map[string]string{"key": "value"},
	})

	tokens, err := p.Search("q").TopK(5).Filter(MetadataFilter{Fields: map[string]string{"key": "value"}}).Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
}

// @sk-test hardening-2026q2#AC-007: StreamSources routing — HyDE
func TestSearchBuilder_StreamSources_HyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens:         []string{"answer"},
		GenerateResult: "hypothetical document",
	}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, err := p.Search("q").TopK(5).HyDE().StreamSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// @sk-test hardening-2026q2#AC-007: StreamSources routing — MultiQuery
func TestSearchBuilder_StreamSources_MultiQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens:         []string{"answer"},
		GenerateResult: "Go concurrency",
	}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, err := p.Search("q").TopK(5).MultiQuery(2).StreamSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// @sk-test hardening-2026q2#AC-007: StreamSources routing — Hybrid (not supported)
func TestSearchBuilder_StreamSources_Hybrid_NotSupported(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), &llm.MockStreamingLLM{}, &fixedEmbedder{})
	_, _, err := p.Search("q").TopK(5).Hybrid(HybridConfig{SemanticWeight: 0.5, RRFK: 60}).StreamSources(context.Background())
	if !errors.Is(err, ErrHybridNotSupported) {
		t.Fatalf("expected ErrHybridNotSupported, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: StreamSources routing — ParentIDs
func TestSearchBuilder_StreamSources_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, err := p.Search("q").TopK(5).ParentIDs("doc-1").StreamSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// @sk-test hardening-2026q2#AC-007: StreamSources routing — Filter
func TestSearchBuilder_StreamSources_Filter(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
		Metadata:  map[string]string{"key": "value"},
	})

	tokens, sources, err := p.Search("q").TopK(5).Filter(MetadataFilter{Fields: map[string]string{"key": "value"}}).StreamSources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// @sk-test hardening-2026q2#AC-007: StreamCite routing — HyDE
func TestSearchBuilder_StreamCite_HyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens:         []string{"answer"},
		GenerateResult: "hypothetical document",
	}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, citations, err := p.Search("q").TopK(5).HyDE().StreamCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

// @sk-test hardening-2026q2#AC-007: StreamCite routing — MultiQuery
func TestSearchBuilder_StreamCite_MultiQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{
		Tokens:         []string{"answer"},
		GenerateResult: "Go concurrency",
	}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, citations, err := p.Search("q").TopK(5).MultiQuery(2).StreamCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

// @sk-test hardening-2026q2#AC-007: StreamCite routing — Hybrid (not supported)
func TestSearchBuilder_StreamCite_Hybrid_NotSupported(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), &llm.MockStreamingLLM{}, &fixedEmbedder{})
	_, _, _, err := p.Search("q").TopK(5).Hybrid(HybridConfig{SemanticWeight: 0.5, RRFK: 60}).StreamCite(context.Background())
	if !errors.Is(err, ErrHybridNotSupported) {
		t.Fatalf("expected ErrHybridNotSupported, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: StreamCite routing — ParentIDs
func TestSearchBuilder_StreamCite_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	tokens, sources, citations, err := p.Search("q").TopK(5).ParentIDs("doc-1").StreamCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

// @sk-test hardening-2026q2#AC-007: StreamCite routing — Filter
func TestSearchBuilder_StreamCite_Filter(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
		Metadata:  map[string]string{"key": "value"},
	})

	tokens, sources, citations, err := p.Search("q").TopK(5).Filter(MetadataFilter{Fields: map[string]string{"key": "value"}}).StreamCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result != "answer" {
		t.Fatalf("expected 'answer', got %q", result)
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

func TestSearchBuilder_Cite_InvalidTopK(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, err := p.Search("Go").TopK(0).Cite(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

func TestSearchBuilder_InlineCite_InvalidTopK(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, _, err := p.Search("Go").TopK(-1).InlineCite(context.Background())
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: Answer routing — Hybrid (not supported on InMemoryStore)
func TestSearchBuilder_Answer_Hybrid_NotSupported(t *testing.T) {
	p, _ := setupPipeline(t)
	_, err := p.Search("q").TopK(5).Hybrid(HybridConfig{SemanticWeight: 0.5, RRFK: 60}).Answer(context.Background())
	if !errors.Is(err, ErrHybridNotSupported) {
		t.Fatalf("expected ErrHybridNotSupported, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: Answer routing — Filter success path
func TestSearchBuilder_Answer_Filter(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
		Metadata: map[string]string{"key": "value"},
	})
	answer, err := p.Search("q").TopK(5).Filter(MetadataFilter{Fields: map[string]string{"key": "value"}}).Answer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

// @sk-test hardening-2026q2#AC-007: Cite routing — HyDE
func TestSearchBuilder_Cite_HyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	p := NewPipeline(store, &mockLLM{reply: "answer"}, emb)
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	answer, sources, err := p.Search("q").TopK(5).HyDE().Cite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// @sk-test hardening-2026q2#AC-007: Cite routing — MultiQuery
func TestSearchBuilder_Cite_MultiQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	p := NewPipeline(store, &mockLLM{reply: "answer"}, emb)
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	answer, sources, err := p.Search("q").TopK(5).MultiQuery(2).Cite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// @sk-test hardening-2026q2#AC-007: Cite routing — Hybrid (not supported)
func TestSearchBuilder_Cite_Hybrid_NotSupported(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, err := p.Search("q").TopK(5).Hybrid(HybridConfig{SemanticWeight: 0.5, RRFK: 60}).Cite(context.Background())
	if !errors.Is(err, ErrHybridNotSupported) {
		t.Fatalf("expected ErrHybridNotSupported, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: InlineCite routing — HyDE
func TestSearchBuilder_InlineCite_HyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	p := NewPipeline(store, &mockLLM{reply: "answer"}, emb)
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	answer, sources, citations, err := p.Search("q").TopK(5).HyDE().InlineCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

// @sk-test hardening-2026q2#AC-007: InlineCite routing — MultiQuery
func TestSearchBuilder_InlineCite_MultiQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	p := NewPipeline(store, &mockLLM{reply: "answer"}, emb)
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	answer, sources, citations, err := p.Search("q").TopK(5).MultiQuery(2).InlineCite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
	_ = citations
}

// @sk-test hardening-2026q2#AC-007: InlineCite routing — Hybrid (not supported)
func TestSearchBuilder_InlineCite_Hybrid_NotSupported(t *testing.T) {
	p, _ := setupPipeline(t)
	_, _, _, err := p.Search("q").TopK(5).Hybrid(HybridConfig{SemanticWeight: 0.5, RRFK: 60}).InlineCite(context.Background())
	if !errors.Is(err, ErrHybridNotSupported) {
		t.Fatalf("expected ErrHybridNotSupported, got %v", err)
	}
}
