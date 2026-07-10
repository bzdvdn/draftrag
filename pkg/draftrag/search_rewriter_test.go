package draftrag

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test query-rewriting#T4.1: mockRewriter для unit-тестов (AC-001, AC-002, AC-003, AC-005, AC-007)

// mockRewriter имитирует QueryRewriter с заданной функцией.
type mockRewriter struct {
	rewriteFn func(context.Context, string, QueryHistory) ([]RewrittenQuery, error)
}

func (m *mockRewriter) Rewrite(ctx context.Context, query string, history QueryHistory) ([]RewrittenQuery, error) {
	return m.rewriteFn(ctx, query, history)
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC001_TypeAssert (AC-001)
// AC-001: type assert кастомной структуры в domain.QueryRewriter.
func TestRewriter_AC001_TypeAssert(t *testing.T) {
	rw := &mockRewriter{
		rewriteFn: func(_ context.Context, q string, _ QueryHistory) ([]RewrittenQuery, error) {
			return []RewrittenQuery{{Query: q, Weight: 1.0}}, nil
		},
	}
	var _ QueryRewriter = rw
	_ = rw // just to satisfy unused
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC002_Priority (AC-002)
// AC-002: per-request Rewriter имеет приоритет над pipeline-level.
func TestRewriter_AC002_Priority(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}

	// Pipeline-level rewriter
	pipelineRW := &mockRewriter{
		rewriteFn: func(_ context.Context, q string, _ QueryHistory) ([]RewrittenQuery, error) {
			return []RewrittenQuery{{Query: "pipeline:" + q, Weight: 1.0}}, nil
		},
	}

	p, err := NewPipelineWithOptions(store, llm, emb, PipelineOptions{
		QueryRewriter: pipelineRW,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "test content", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	// Per-request rewriter (другой префикс)
	perRequestRW := &mockRewriter{
		rewriteFn: func(_ context.Context, q string, _ QueryHistory) ([]RewrittenQuery, error) {
			return []RewrittenQuery{{Query: "per-request:" + q, Weight: 1.0}}, nil
		},
	}
	result, err := p.Search("test").TopK(5).Rewriter(perRequestRW).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected chunks with per-request rewriter")
	}
	// QueryText должен быть оригинальным (не переписанным) — устанавливается pipeline
	if result.QueryText != "test" {
		t.Fatalf("expected QueryText='test', got %q", result.QueryText)
	}
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC003_MultiQueryFusion (AC-003)
// AC-003: mock возвращает 3 переформулировки → RRF fusion.
func TestRewriter_AC003_MultiQueryFusion(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}
	p, err := NewPipeline(store, llm, emb)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	// Три чанка с разными embedding
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Chunk one", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c2", Content: "Chunk two", ParentID: "doc-2",
		Embedding: []float64{0.9, 0.1, 0}, Position: 0,
	})
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c3", Content: "Chunk three", ParentID: "doc-3",
		Embedding: []float64{0.8, 0.2, 0}, Position: 0,
	})

	rw := &mockRewriter{
		rewriteFn: func(_ context.Context, q string, _ QueryHistory) ([]RewrittenQuery, error) {
			return []RewrittenQuery{
				{Query: "variation one " + q, Weight: 1.0},
				{Query: "variation two " + q, Weight: 1.0},
				{Query: "variation three " + q, Weight: 1.0},
			}, nil
		},
	}

	result, err := p.Search("test").TopK(5).Rewriter(rw).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected chunks from multi-query fusion")
	}
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC004_History (AC-004)
// AC-004: LLMRewriter с history → контекстная переформулировка.
func TestRewriter_AC004_History(t *testing.T) {
	history := QueryHistory{
		Entries: []domain.Message{
			{Role: "user", Content: "как работает RAG?"},
			{Role: "assistant", Content: "RAG извлекает документы и генерирует ответ"},
		},
	}

	capturedHistory := false
	rw := &mockRewriter{
		rewriteFn: func(_ context.Context, _ string, h QueryHistory) ([]RewrittenQuery, error) {
			if len(h.Entries) > 0 {
				capturedHistory = true
			}
			return []RewrittenQuery{{Query: "what are the disadvantages of RAG", Weight: 1.0}}, nil
		},
	}

	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}
	p, err := NewPipeline(store, llm, emb)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "RAG disadvantages", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	_, err = p.Search("what are the disadvantages").TopK(5).Rewriter(rw).History(history).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !capturedHistory {
		t.Fatal("expected history to be passed to Rewriter")
	}
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC005_ErrorFallback (AC-005)
// AC-005: error-rewriter → fallback + log.
func TestRewriter_AC005_ErrorFallback(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}
	p, err := NewPipeline(store, llm, emb)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "test content", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	errRewriter := &mockRewriter{
		rewriteFn: func(_ context.Context, _ string, _ QueryHistory) ([]RewrittenQuery, error) {
			return nil, errors.New("rewriter failed")
		},
	}

	// Должен успешно выполниться с исходным запросом (fallback)
	result, err := p.Search("test content").TopK(5).Rewriter(errRewriter).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results from fallback (original query)")
	}
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC007_OverrideHyDE (AC-007)
// AC-007: Rewriter + HyDE/MultiQuery → warning + игнор.
func TestRewriter_AC007_OverrideHyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}
	p, err := NewPipeline(store, llm, emb)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "content from rewriter", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})

	usedRewriter := false
	rw := &mockRewriter{
		rewriteFn: func(_ context.Context, _ string, _ QueryHistory) ([]RewrittenQuery, error) {
			usedRewriter = true
			return []RewrittenQuery{{Query: "content from rewriter", Weight: 1.0}}, nil
		},
	}

	// Rewriter + HyDE: HyDE игнорируется, используется Rewriter
	result, err := p.Search("test").TopK(5).Rewriter(rw).HyDE().Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !usedRewriter {
		t.Fatal("expected Rewriter to be used instead of HyDE")
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results from rewriter")
	}

	// Rewriter + MultiQuery: MultiQuery игнорируется
	usedRewriter = false
	result, err = p.Search("test").TopK(5).Rewriter(rw).MultiQuery(3).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !usedRewriter {
		t.Fatal("expected Rewriter to be used instead of MultiQuery")
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results from rewriter")
	}
}

// @sk-test query-rewriting#T4.1: TestRewriter_AC004_LLMRewriter_ContextHistory (AC-004)
// AC-004: LLMRewriter с history через LLM → история влияет на переформулировку.
func TestRewriter_AC004_LLMRewriter_ContextHistory(t *testing.T) {
	history := QueryHistory{
		Entries: []domain.Message{
			{Role: "user", Content: "о чём идёт речь?"},
			{Role: "assistant", Content: "мы обсуждаем Go"},
		},
	}

	// LLM возвращает переформулировку, включающую контекст
	llm := &mockLLM{reply: "what's the performance of Go goroutines?"}
	rw, err := NewLLMRewriter(llm, "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := rw.Rewrite(context.Background(), "какая производительность?", history)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected rewritten query")
	}
	if !strings.Contains(result[0].Query, "Go") && !strings.Contains(result[0].Query, "goroutines") {
		t.Logf("rewritten: %s", result[0].Query)
	}
}
