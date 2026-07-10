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

func (noDocStore) Health(_ context.Context) error                 { return nil }
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
	p, err := NewPipelineWithChunker(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, mockChunker{})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil Pipeline")
	}
}

func TestNewPipelineWithChunker_IndexesViaChunker(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipelineWithChunker(store, testLLM{}, testEmbedder{}, mockChunker{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	err = p.Index(ctx, []domain.Document{{ID: "doc-1", Content: "hello"}})
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
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Retrieve(context.Background(), "", 5)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestRetrieve_InvalidTopK(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Retrieve(context.Background(), "q", 0)
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestRetrieve_NilContext(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, err = p.Retrieve(nil, "q", 5)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestUpdateDocument_Updates(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	_ = store.Upsert(ctx, domain.Chunk{
		ID: "doc-1_c0", Content: "old", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	err = p.UpdateDocument(ctx, domain.Document{ID: "doc-1", Content: "new content"})
	if err != nil {
		t.Fatalf("UpdateDocument failed: %v", err)
	}
}

func TestUpdateDocument_EmptyContent(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	err = p.UpdateDocument(context.Background(), domain.Document{ID: "doc-1", Content: ""})
	if !errors.Is(err, ErrEmptyDocument) {
		t.Fatalf("expected ErrEmptyDocument, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestUpdateDocument_NilContext(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	err = p.UpdateDocument(nil, domain.Document{ID: "d", Content: "x"})
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestDeleteDocument_Deletes(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	_ = store.Upsert(ctx, domain.Chunk{
		ID: "doc-1_c0", Content: "x", ParentID: "doc-1",
		Embedding: []float64{1, 0}, Position: 0,
	})

	err = p.DeleteDocument(ctx, "doc-1")
	if err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}
}

func TestDeleteDocument_EmptyID(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	err = p.DeleteDocument(context.Background(), "")
	if !errors.Is(err, ErrEmptyDocumentID) {
		t.Fatalf("expected ErrEmptyDocumentID, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestDeleteDocument_NilContext(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	err = p.DeleteDocument(nil, "doc-1")
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestDeleteDocument_StoreWithoutCapability(t *testing.T) {
	p, err := NewPipeline(noDocStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	err = p.DeleteDocument(context.Background(), "doc-1")
	if !errors.Is(err, ErrDeleteNotSupported) {
		t.Fatalf("expected ErrDeleteNotSupported, got %v", err)
	}
}

func TestIndexBatch_EmptyDocument(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.IndexBatch(context.Background(), []domain.Document{
		{ID: "doc-1", Content: ""},
	}, 2)
	if !errors.Is(err, ErrEmptyDocument) {
		t.Fatalf("expected ErrEmptyDocument, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestIndexBatch_NilContext(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, err = p.IndexBatch(nil, []domain.Document{{ID: "d", Content: "x"}}, 2)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestSearch_Stream_FilterNotSupported(t *testing.T) {
	p, err := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, err = p.Search("q").TopK(5).Filter(filter).Stream(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_StreamSources_FilterNotSupported(t *testing.T) {
	p, err := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, _, err = p.Search("q").TopK(5).Filter(filter).StreamSources(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_StreamCite_FilterNotSupported(t *testing.T) {
	p, err := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, _, _, err = p.Search("q").TopK(5).Filter(filter).StreamCite(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_Stream_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	streamingLLM := &llm.MockStreamingLLM{Tokens: []string{"answer"}}
	p, err := NewPipeline(store, streamingLLM, emb)
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}
	_, _, err = p.Search("q").TopK(5).Filter(filter).Cite(context.Background())
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestSearch_Cite_ParentIDs(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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

func TestMapAppError_Passthrough(t *testing.T) {
	customErr := errors.New("custom error")
	result := mapAppError(customErr)
	if !errors.Is(result, customErr) {
		t.Fatalf("expected passthrough, got %v", result)
	}
}

// @sk-test arch-quality-pass#T4.1: DefaultTopK < 0 → error (AC-002)
func TestNewPipelineWithOptions_DefaultTopK_Invalid(t *testing.T) {
	_, err := NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		DefaultTopK: -1,
	})
	if err == nil {
		t.Fatal("expected error on negative DefaultTopK")
	}
}

// @sk-test arch-quality-pass#T4.1: MaxContextChars < 0 → error (AC-002)
func TestNewPipelineWithOptions_MaxContextChars_Invalid(t *testing.T) {
	_, err := NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MaxContextChars: -1,
	})
	if err == nil {
		t.Fatal("expected error on negative MaxContextChars")
	}
}

// @sk-test arch-quality-pass#T4.1: MaxContextChunks < 0 → error (AC-002)
func TestNewPipelineWithOptions_MaxContextChunks_Invalid(t *testing.T) {
	_, err := NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MaxContextChunks: -1,
	})
	if err == nil {
		t.Fatal("expected error on negative MaxContextChunks")
	}
}

// @sk-test arch-quality-pass#T4.1: MMRLambda < 0 и > 1 → error (AC-002)
func TestNewPipelineWithOptions_MMRLambda_Invalid(t *testing.T) {
	_, err := NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MMRLambda: 1.5,
	})
	if err == nil {
		t.Fatal("expected error on invalid MMRLambda")
	}
	_, err = NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MMRLambda: -0.5,
	})
	if err == nil {
		t.Fatal("expected error on negative MMRLambda")
	}
}

// @sk-test arch-quality-pass#T4.1: MMRCandidatePool < 0 → error (AC-002)
func TestNewPipelineWithOptions_MMRCandidatePool_Invalid(t *testing.T) {
	_, err := NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		MMRCandidatePool: -1,
	})
	if err == nil {
		t.Fatal("expected error on negative MMRCandidatePool")
	}
}

// @sk-test arch-quality-pass#T4.1: StreamBufferSize < 0 → error (AC-002)
func TestNewPipelineWithOptions_StreamBufferSize_Invalid(t *testing.T) {
	_, err := NewPipelineWithOptions(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{}, PipelineOptions{
		StreamBufferSize: -1,
	})
	if err == nil {
		t.Fatal("expected error on negative StreamBufferSize")
	}
}

// @sk-test arch-quality-pass#T4.1: все поля zero → (p, nil) с дефолтами (AC-003)
func TestNewPipelineWithOptions_ValidZeroConfig(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipelineWithOptions(store, testLLM{}, testEmbedder{}, PipelineOptions{})
	if err != nil {
		t.Fatalf("expected no error for zero config, got %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil Pipeline")
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestPipeline_NilContext_Index(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	err = p.Index(nil, []domain.Document{{ID: "d", Content: "x"}})
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestPipeline_NilContext_Query(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, err = p.Query(nil, "q")
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestSearch_NilContext_Cite(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, _, err = p.Search("q").TopK(5).Cite(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestSearch_NilContext_InlineCite(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, _, _, err = p.Search("q").TopK(5).InlineCite(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestSearch_NilContext_Stream(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, err = p.Search("q").TopK(5).Stream(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestSearch_NilContext_StreamSources(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, _, err = p.Search("q").TopK(5).StreamSources(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestSearch_NilContext_StreamCite(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, _, _, err = p.Search("q").TopK(5).StreamCite(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestSearch_Answer_WithResults(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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

// @sk-test arch-generics#T4.1: nil context guard вместо panic (AC-002)
func TestPipeline_NilContext_Answer(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // intentional: testing nil context
	_, err = p.Answer(nil, "q")
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

// @sk-test hardening-2026q2#AC-007: IndexBatch happy path
func TestIndexBatch_Success(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}
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

// @sk-test hardening-2026q2#AC-010: mapAppError streaming not supported mapping
// @sk-test api-consistency-pass#T2.2: ErrStreamingNotSupported reachable через mapAppError (AC-005, RQ-003)
func TestMapAppError_StreamingNotSupported(t *testing.T) {
	err := mapAppError(application.ErrStreamingNotSupported)
	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("expected ErrStreamingNotSupported, got %v", err)
	}
}
