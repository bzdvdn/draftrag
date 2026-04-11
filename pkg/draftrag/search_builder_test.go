package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// mockLLM возвращает фиксированный ответ.
type mockLLM struct{ reply string }

func (m *mockLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return m.reply, nil
}

// errLLM всегда возвращает ошибку.
type errLLM struct{}

func (errLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "", errors.New("llm error")
}

// fixedEmbedder возвращает фиксированный вектор.
type fixedEmbedder struct{ vec []float64 }

func (f *fixedEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return f.vec, nil
}

func setupPipeline(t *testing.T) (*Pipeline, *vectorstore.InMemoryStore) {
	t.Helper()
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}
	p := NewPipeline(store, llm, emb)

	ctx := context.Background()
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c1", Content: "Go concurrency", ParentID: "doc-1",
		Embedding: []float64{1, 0, 0}, Position: 0,
	})
	_ = store.Upsert(ctx, domain.Chunk{
		ID: "c2", Content: "Go channels", ParentID: "doc-2",
		Embedding: []float64{0.9, 0.1, 0}, Position: 0,
	})
	return p, store
}

// --- Validation ---

func TestSearchBuilder_EmptyQuestion(t *testing.T) {
	p, _ := setupPipeline(t)
	ctx := context.Background()

	_, err := p.Search("  ").TopK(5).Retrieve(ctx)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
	_, err = p.Search("  ").TopK(5).Answer(ctx)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestSearchBuilder_InvalidTopK(t *testing.T) {
	p, _ := setupPipeline(t)
	ctx := context.Background()

	_, err := p.Search("q").TopK(0).Retrieve(ctx)
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
	_, err = p.Search("q").TopK(-1).Answer(ctx)
	if !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK, got %v", err)
	}
}

func TestSearchBuilder_NilContext(t *testing.T) {
	p, _ := setupPipeline(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	//nolint:staticcheck
	_, _ = p.Search("q").TopK(5).Retrieve(nil)
}

// --- Basic Retrieve ---

func TestSearchBuilder_Retrieve(t *testing.T) {
	p, _ := setupPipeline(t)
	result, err := p.Search("concurrency").TopK(2).Retrieve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results")
	}
}

// --- ParentIDs routing ---

func TestSearchBuilder_ParentIDs(t *testing.T) {
	p, _ := setupPipeline(t)
	result, err := p.Search("Go").TopK(5).ParentIDs("doc-1").Retrieve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, rc := range result.Chunks {
		if rc.Chunk.ParentID != "doc-1" {
			t.Fatalf("expected doc-1, got %s", rc.Chunk.ParentID)
		}
	}
}

// --- Filter routing (in-memory supports it) ---

func TestSearchBuilder_Filter(t *testing.T) {
	p, _ := setupPipeline(t)
	filter := MetadataFilter{Fields: map[string]string{"nonexistent": "value"}}
	_, err := p.Search("Go").TopK(5).Filter(filter).Retrieve(context.Background())
	// in-memory supports filters; empty result is fine
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Answer ---

func TestSearchBuilder_Answer(t *testing.T) {
	p, _ := setupPipeline(t)
	answer, err := p.Search("Go").TopK(2).Answer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
}

// --- Cite ---

func TestSearchBuilder_Cite(t *testing.T) {
	p, _ := setupPipeline(t)
	answer, sources, err := p.Search("Go").TopK(2).Cite(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected answer")
	}
	if len(sources.Chunks) == 0 {
		t.Fatal("expected sources")
	}
}

// --- Stream ---

func TestSearchBuilder_StreamContextCancel(t *testing.T) {
	p, _ := setupPipeline(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.Search("Go").TopK(2).Stream(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
}

// --- HyDE ---

func TestSearchBuilder_HyDE(t *testing.T) {
	p, _ := setupPipeline(t)
	result, err := p.Search("concurrency").TopK(2).HyDE().Retrieve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results from HyDE")
	}
}

func TestSearchBuilder_HyDE_Answer(t *testing.T) {
	p, _ := setupPipeline(t)
	answer, err := p.Search("concurrency").TopK(2).HyDE().Answer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected answer")
	}
}

// --- MultiQuery ---

func TestSearchBuilder_MultiQuery(t *testing.T) {
	p, _ := setupPipeline(t)
	result, err := p.Search("Go concurrency").TopK(2).MultiQuery(2).Retrieve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected results")
	}
}

func TestSearchBuilder_MultiQuery_Answer(t *testing.T) {
	p, _ := setupPipeline(t)
	answer, err := p.Search("Go concurrency").TopK(2).MultiQuery(2).Answer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if answer == "" {
		t.Fatal("expected answer")
	}
}

// noFilterStore реализует только domain.VectorStore (без VectorStoreWithFilters).
// Используется для проверки маппинга ErrFiltersNotSupported в SearchBuilder.
type noFilterStore struct{}

func (noFilterStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (noFilterStore) Delete(_ context.Context, _ string) error        { return nil }
func (noFilterStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

// @sk-task T2.1: тест маппинга ErrFiltersNotSupported в InlineCite (AC-001, AC-003)
func TestSearchBuilder_InlineCite_FilterNotSupported(t *testing.T) {
	p := NewPipeline(noFilterStore{}, &mockLLM{reply: "answer"}, &fixedEmbedder{vec: []float64{1, 0, 0}})
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}

	_, _, _, err := p.Search("вопрос").TopK(5).Filter(filter).InlineCite(context.Background())

	// AC-001: публичный ErrFiltersNotSupported возвращается (не internal application-ошибка).
	// AC-003: маппинг через errors.Is корректно обрабатывает и обёрнутые ошибки.
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("ожидался ErrFiltersNotSupported, получен %v", err)
	}
}

// @sk-task T3.1: тест маппинга ErrStreamingNotSupported в StreamSources (AC-003)
func TestSearchBuilder_StreamSources_StreamingNotSupported(t *testing.T) {
	// mockLLM реализует только LLMProvider (не StreamingLLMProvider),
	// поэтому StreamSources должен вернуть ErrStreamingNotSupported.
	p := NewPipeline(vectorstore.NewInMemoryStore(), &mockLLM{reply: "answer"}, &fixedEmbedder{vec: []float64{1, 0, 0}})

	ch, sources, err := p.Search("вопрос").TopK(5).StreamSources(context.Background())

	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Fatalf("ожидался ErrStreamingNotSupported, получен %v", err)
	}
	if ch != nil {
		t.Error("канал должен быть nil при ошибке")
	}
	if len(sources.Chunks) != 0 {
		t.Error("RetrievalResult должен быть пустым при ошибке")
	}
}
