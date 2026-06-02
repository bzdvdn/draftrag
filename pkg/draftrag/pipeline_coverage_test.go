package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/application"
	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/llm"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type noDocStore struct{}

func (noDocStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (noDocStore) Delete(_ context.Context, _ string) error       { return nil }
func (noDocStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

type mockChunker struct{}

func (mockChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return []domain.Chunk{{
		ID: doc.ID + "_c0", Content: doc.Content,
		ParentID: doc.ID, Embedding: nil, Position: 0,
	}}, nil
}

func TestNewPipelineWithChunker_Constructs(t *testing.T) {
	p := NewPipelineWithChunker(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, mockChunker{})
	if p == nil {
		t.Fatal("expected non-nil Pipeline")
	}
}

func TestNewPipelineWithChunker_IndexesViaChunker(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipelineWithChunker(store, testLLM{}, testEmbedder{}, mockChunker{})
	ctx := context.Background()

	err := p.Index(ctx, []domain.Document{{ID: "doc-1", Content: "hello"}})
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}

	result, err := p.Search("hello").TopK(5).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected chunked result")
	}
}

func TestRetrieve_ReturnsResults(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go channels", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	result, err := p.Retrieve(ctx, "Go", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results")
	}
}

func TestRetrieve_EmptyQuestion(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	_, err := p.Retrieve(context.Background(), "", 5)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestRetrieve_InvalidTopK(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	_, err := p.Retrieve(context.Background(), "q", 0)
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

func TestRetrieve_NilContext(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _ = p.Retrieve(nil, "q", 5)
}

func TestUpdateDocument_Updates(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()

	_ = store.Upsert(ctx, domain.Chunk{
		ID: "doc-1_c0", Content: "old", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	err := p.UpdateDocument(ctx, domain.Document{ID: "doc-1", Content: "new content"})
	if err != nil {
		t.Fatalf("UpdateDocument failed: %v", err)
	}
}

func TestUpdateDocument_EmptyContent(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	err := p.UpdateDocument(context.Background(), domain.Document{ID: "doc-1", Content: ""})
	if !errors.Is(err, ErrEmptyDocument) {
		t.Fatalf("expected ErrEmptyDocument, got %v", err)
	}
}

func TestUpdateDocument_NilContext(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_ = p.UpdateDocument(nil, domain.Document{ID: "d", Content: "x"})
}

func TestDeleteDocument_Deletes(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()

	_ = store.Upsert(ctx, domain.Chunk{
		ID: "doc-1_c0", Content: "x", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	err := p.DeleteDocument(ctx, "doc-1")
	if err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}
}

func TestDeleteDocument_EmptyID(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	err := p.DeleteDocument(context.Background(), "")
	if !errors.Is(err, ErrEmptyDocumentID) {
		t.Fatalf("expected ErrEmptyDocumentID, got %v", err)
	}
}

func TestDeleteDocument_NilContext(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_ = p.DeleteDocument(nil, "doc-1")
}

func TestDeleteDocument_StoreWithoutCapability(t *testing.T) {
	p := NewPipeline(noDocStore{}, testLLM{}, testEmbedder{})
	err := p.DeleteDocument(context.Background(), "doc-1")
	if !errors.Is(err, ErrDeleteNotSupported) {
		t.Fatalf("expected ErrDeleteNotSupported, got %v", err)
	}
}

func TestIndexBatch_EmptyDocument(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	_, err := p.IndexBatch(context.Background(), []domain.Document{
		{ID: "doc-1", Content: ""},
	}, 2)
	if !errors.Is(err, ErrEmptyDocument) {
		t.Fatalf("expected ErrEmptyDocument, got %v", err)
	}
}

func TestIndexBatch_NilContext(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _ = p.IndexBatch(nil, []domain.Document{{ID: "d", Content: "x"}}, 2)
}

func TestSearch_Stream_FilterNotSupported(t *testing.T) {
	p := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, err := p.Search("q").TopK(5).Filter(filter).Stream(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_StreamSources_FilterNotSupported(t *testing.T) {
	p := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, _, err := p.Search("q").TopK(5).Filter(filter).StreamSources(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_StreamCite_FilterNotSupported(t *testing.T) {
	p := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, _, _, err := p.Search("q").TopK(5).Filter(filter).StreamCite(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_Stream_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p := NewPipeline(store, streamingLLM, emb)
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	tokens, err := p.Search("q").TopK(5).ParentIDs("doc-1").Stream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	for tok := range tokens {
		result += tok
	}
	if result == "" {
		t.Fatal("expected tokens")
	}
}

func TestSearch_Cite_FilterNotSupported(t *testing.T) {
	p := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, _, err := p.Search("q").TopK(5).Filter(filter).Cite(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_Cite_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})
	answer, sources, err := p.Search("q").TopK(5).ParentIDs("doc-1").Cite(ctx)
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

func TestSearch_InlineCite_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})
	answer, sources, citations, err := p.Search("q").TopK(5).ParentIDs("doc-1").InlineCite(ctx)
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

func TestMapValidationErr_Passthrough(t *testing.T) {
	customErr := errors.New("custom error")
	result := mapValidationErr(customErr)
	if !errors.Is(result, customErr) {
		t.Fatalf("expected passthrough, got %v", result)
	}
}

func TestPipelineOptions_PanicsOnNegativeTopK(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative DefaultTopK")
		}
	}()
	_ = NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		DefaultTopK: -1,
	})
}

func TestPipelineOptions_PanicsOnNegativeMaxContextChars(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative MaxContextChars")
		}
	}()
	_ = NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MaxContextChars: -1,
	})
}

func TestPipelineOptions_PanicsOnNegativeMaxContextChunks(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative MaxContextChunks")
		}
	}()
	_ = NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MaxContextChunks: -1,
	})
}

func TestPipelineOptions_PanicsOnInvalidMMR(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid MMRLambda")
		}
	}()
	_ = NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MMRLambda: 1.5,
	})
}

func TestPipelineOptions_PanicsOnNegativeCandidates(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative MMRCandidatePool")
		}
	}()
	_ = NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MMRCandidatePool: -1,
	})
}

func TestPipeline_NilContext_Index(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = p.Index(nil, []domain.Document{{ID: "d", Content: "x"}})
}

func TestPipeline_NilContext_Query(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_, _ = p.Query(nil, "q")
}

func TestSearch_NilContext_Cite(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _, _ = p.Search("q").TopK(5).Cite(nil)
}

func TestSearch_NilContext_InlineCite(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _, _, _ = p.Search("q").TopK(5).InlineCite(nil)
}

func TestSearch_NilContext_Stream(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _ = p.Search("q").TopK(5).Stream(nil)
}

func TestSearch_NilContext_StreamSources(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _, _ = p.Search("q").TopK(5).StreamSources(nil)
}

func TestSearch_NilContext_StreamCite(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	_, _, _, _ = p.Search("q").TopK(5).StreamCite(nil)
}

func TestSearch_Answer_WithResults(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	answer, err := p.Search("Go").TopK(5).Answer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

func TestSearch_Answer_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})
	answer, err := p.Search("q").TopK(5).ParentIDs("doc-1").Answer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

func TestSearch_Answer_MultiQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})
	answer, err := p.Search("q").TopK(5).MultiQuery(2).Answer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

func TestSearch_Answer_HyDE(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})
	answer, err := p.Search("q").TopK(5).HyDE().Answer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

func TestSearch_Cite_WithResults(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	answer, sources, err := p.Search("Go").TopK(5).Cite(ctx)
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

func TestPipeline_NilContext_Answer(t *testing.T) {
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_, _ = p.Answer(nil, "q")
}

// @sk-test hardening-2026q2#AC-007: IndexBatch happy path
func TestIndexBatch_Success(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})
	ctx := context.Background()

	result, err := p.IndexBatch(ctx, []domain.Document{
		{ID: "doc-1", Content: "hello world"},
	}, 1)
	if err != nil {
		t.Fatalf("IndexBatch failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Successful) != 1 {
		t.Fatalf("expected 1 successful, got %d", len(result.Successful))
	}
}

// @sk-test hardening-2026q2#AC-010: mapValidationErr streaming not supported mapping
func TestMapValidationErr_StreamingNotSupported(t *testing.T) {
	err := mapValidationErr(application.ErrStreamingNotSupported)
	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
}
