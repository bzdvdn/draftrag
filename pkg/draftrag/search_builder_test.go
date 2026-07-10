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

func (m *mockLLM) Health(_ context.Context) error { return nil }
func (m *mockLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return m.reply, nil
}

// fixedEmbedder возвращает фиксированный вектор.
type fixedEmbedder struct{ vec []float64 }

func (f *fixedEmbedder) Health(_ context.Context) error { return nil }
func (f *fixedEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return f.vec, nil
}

func setupPipeline(t *testing.T) (*Pipeline, *vectorstore.InMemoryStore) {
	t.Helper()
	store := vectorstore.NewInMemoryStore()
	emb := &fixedEmbedder{vec: []float64{1, 0, 0}}
	llm := &mockLLM{reply: "answer"}
	p, err := NewPipeline(store, llm, emb)
	if err != nil {
		t.Fatal(err)
	}

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
	//nolint:staticcheck
	_, err := p.Search("q").TopK(5).Retrieve(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
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

func (noFilterStore) Health(_ context.Context) error                 { return nil }
func (noFilterStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (noFilterStore) Delete(_ context.Context, _ string) error       { return nil }
func (noFilterStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

// @sk-task T2.1: тест маппинга ErrFiltersNotSupported в InlineCite (AC-001, AC-003)
func TestSearchBuilder_InlineCite_FilterNotSupported(t *testing.T) {
	p, err := NewPipeline(noFilterStore{}, &mockLLM{reply: "answer"}, &fixedEmbedder{vec: []float64{1, 0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	filter := MetadataFilter{Fields: map[string]string{"key": "value"}}

	_, _, _, err = p.Search("вопрос").TopK(5).Filter(filter).InlineCite(context.Background())

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
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), &mockLLM{reply: "answer"}, &fixedEmbedder{vec: []float64{1, 0, 0}})
	if err != nil {
		t.Fatal(err)
	}

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

// @sk-test arch-generics#T3.1: table-driven test всех комбинаций маршрут × output-метод (AC-002)
func TestSearchBuilder_RouteMatrix(t *testing.T) {
	p, _ := setupPipeline(t)
	ctx := context.Background()

	routes := []struct {
		name  string
		setup func(*SearchBuilder) *SearchBuilder
	}{
		{"basic", func(b *SearchBuilder) *SearchBuilder { return b }},
		{"HyDE", func(b *SearchBuilder) *SearchBuilder { return b.HyDE() }},
		{"MultiQuery", func(b *SearchBuilder) *SearchBuilder { return b.MultiQuery(2) }},
		{"Hybrid", func(b *SearchBuilder) *SearchBuilder { return b.Hybrid(HybridConfig{SemanticWeight: 0.7}) }},
		{"ParentIDs", func(b *SearchBuilder) *SearchBuilder { return b.ParentIDs("doc-1") }},
		{"Filter", func(b *SearchBuilder) *SearchBuilder {
			return b.Filter(MetadataFilter{Fields: map[string]string{"key": "val"}})
		}},
	}

	methods := []struct {
		name string
		call func(*SearchBuilder) error
	}{
		{"Retrieve", func(b *SearchBuilder) error { _, err := b.TopK(3).Retrieve(ctx); return err }},
		{"Answer", func(b *SearchBuilder) error { _, err := b.TopK(3).Answer(ctx); return err }},
		{"Cite", func(b *SearchBuilder) error { _, _, err := b.TopK(3).Cite(ctx); return err }},
		{"InlineCite", func(b *SearchBuilder) error { _, _, _, err := b.TopK(3).InlineCite(ctx); return err }},
		{"Stream", func(b *SearchBuilder) error { _, err := b.TopK(3).Stream(ctx); return err }},
		{"StreamSources", func(b *SearchBuilder) error { _, _, err := b.TopK(3).StreamSources(ctx); return err }},
		{"StreamCite", func(b *SearchBuilder) error { _, _, _, err := b.TopK(3).StreamCite(ctx); return err }},
	}

	for _, rt := range routes {
		for _, mt := range methods {
			t.Run(rt.name+"_"+mt.name, func(t *testing.T) {
				sb := rt.setup(p.Search("test question"))
				err := mt.call(sb)
				if errors.Is(err, ErrEmptyQuery) || errors.Is(err, ErrInvalidTopK) {
					t.Fatalf("unexpected validation error for route %s: %v", rt.name, err)
				}
			})
		}
	}
}

// @sk-test arch-generics#T3.2: prototype добавления нового output-метода через router (AC-003)
func TestSearchBuilder_AnalyzePrototype(t *testing.T) {
	p, _ := setupPipeline(t)
	ctx := context.Background()

	// Определение result-типа для нового метода
	type rAnalyze struct{ Result string }

	// Регистрация handler-ов (6 маршрутов)
	analyzeRouter := router[rAnalyze]{handlers: map[route]func(context.Context, string, int, *SearchBuilder) (rAnalyze, error){
		routeBasic: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnalyze, error) {
			t, err := b.pipeline.core.Answer(ctx, q, topK)
			return rAnalyze{Result: t}, err
		},
		routeHyDE: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnalyze, error) {
			t, err := b.pipeline.core.AnswerHyDE(ctx, q, topK)
			return rAnalyze{Result: t}, err
		},
		routeMultiQuery: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnalyze, error) {
			t, err := b.pipeline.core.AnswerMulti(ctx, q, b.multiQuery, topK)
			return rAnalyze{Result: t}, err
		},
		routeHybrid: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnalyze, error) {
			t, err := b.pipeline.core.AnswerHybrid(ctx, q, topK, *b.hybrid)
			return rAnalyze{Result: t}, err
		},
		routeParentIDs: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnalyze, error) {
			t, err := b.pipeline.core.AnswerWithParentIDs(ctx, q, topK, b.parentIDs)
			return rAnalyze{Result: t}, err
		},
		routeFilter: func(ctx context.Context, q string, topK int, b *SearchBuilder) (rAnalyze, error) {
			t, err := b.pipeline.core.AnswerWithMetadataFilter(ctx, q, topK, b.filter)
			return rAnalyze{Result: t}, err
		},
	}}

	sb := p.Search("test").TopK(3)
	q, r, err := sb.pickRoute()
	if err != nil {
		t.Fatal(err)
	}
	res, err := analyzeRouter.execute(ctx, q, 3, r, sb)
	if err != nil {
		t.Fatal(err)
	}
	if res.Result == "" {
		t.Fatal("expected non-empty result")
	}
}
