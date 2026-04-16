package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type testEmbedder struct{}

func (testEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	text = strings.ToLower(text)
	if strings.Contains(text, "cat") {
		return []float64{1, 0}, nil
	}
	return []float64{0, 1}, nil
}

type testLLM struct{}

func (testLLM) Generate(ctx context.Context, _, _ string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "ok", nil
}

func TestPipeline_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})

	_, err := p.Query(ctx, "cat", 5)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPipeline_FullCycle(t *testing.T) {
	ctx := context.Background()
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})

	docs := []domain.Document{
		{
			ID:      "doc-1",
			Content: "cat",
		},
	}

	if err := p.Index(ctx, docs); err != nil {
		t.Fatalf("index: %v", err)
	}

	result, err := p.Query(ctx, "cat", 5)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if result.TotalFound == 0 || len(result.Chunks) == 0 {
		t.Fatalf("expected results, got total=%d len=%d", result.TotalFound, len(result.Chunks))
	}
	if result.QueryText != "cat" {
		t.Fatalf("expected QueryText=cat, got %q", result.QueryText)
	}
}

func TestPipeline_QueryWithParentIDs_FiltersNotSupported(t *testing.T) {
	// InMemoryStore теперь реализует VectorStoreWithFilters; используем non-filter store.
	ctx := context.Background()
	p := NewPipeline(&noFilterStore{}, testLLM{}, testEmbedder{})

	_, err := p.QueryWithParentIDs(ctx, "cat", 5, []string{"doc-1"})
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestPipeline_QueryWithParentIDs_EmptyFilterFallsBack(t *testing.T) {
	ctx := context.Background()
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})

	// Пустой фильтр не должен требовать capability.
	_, err := p.QueryWithParentIDs(ctx, "cat", 5, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// @sk-task T4.1: Unit-тесты QueryWithMetadataFilter и AnswerWithMetadataFilter (AC-002, AC-003, DEC-003)

// TestPipeline_QueryWithMetadataFilter_FiltersNotSupported проверяет, что вызов с непустым фильтром
// на store без VectorStoreWithFilters возвращает ErrFiltersNotSupported (AC-003, DEC-003).
func TestPipeline_QueryWithMetadataFilter_FiltersNotSupported(t *testing.T) {
	// InMemoryStore теперь реализует VectorStoreWithFilters, используем минимальный non-filter store.
	ctx := context.Background()
	p := NewPipeline(&noFilterStore{}, testLLM{}, testEmbedder{})

	_, err := p.QueryWithMetadataFilter(ctx, "cat", 5, domain.MetadataFilter{
		Fields: map[string]string{"category": "legal"},
	})
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

// TestPipeline_QueryWithMetadataFilter_EmptyFilterFallsBack проверяет, что пустой фильтр
// не требует VectorStoreWithFilters capability и возвращает результат без ошибки (AC-002).
func TestPipeline_QueryWithMetadataFilter_EmptyFilterFallsBack(t *testing.T) {
	ctx := context.Background()
	p := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})

	_, err := p.QueryWithMetadataFilter(ctx, "cat", 5, domain.MetadataFilter{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestPipeline_QueryWithMetadataFilter_PassesFilterToStore проверяет, что фильтр передаётся
// в SearchWithMetadataFilter и возвращаются только совпадающие чанки (AC-003).
func TestPipeline_QueryWithMetadataFilter_PassesFilterToStore(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLM{}, testEmbedder{})

	docs := []domain.Document{
		{ID: "doc-legal", Content: "cat"},
		{ID: "doc-finance", Content: "cat"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatalf("index: %v", err)
	}

	// Проставляем метаданные напрямую через Upsert — индексация не propagates metadata в чанки.
	if err := store.Upsert(ctx, domain.Chunk{
		ID:        "doc-legal#0",
		Content:   "cat",
		ParentID:  "doc-legal",
		Embedding: []float64{1, 0},
		Metadata:  map[string]string{"category": "legal"},
	}); err != nil {
		t.Fatalf("upsert legal: %v", err)
	}
	if err := store.Upsert(ctx, domain.Chunk{
		ID:        "doc-finance#0",
		Content:   "cat",
		ParentID:  "doc-finance",
		Embedding: []float64{1, 0},
		Metadata:  map[string]string{"category": "finance"},
	}); err != nil {
		t.Fatalf("upsert finance: %v", err)
	}

	result, err := p.QueryWithMetadataFilter(ctx, "cat", 10, domain.MetadataFilter{
		Fields: map[string]string{"category": "legal"},
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	for _, rc := range result.Chunks {
		if rc.Chunk.Metadata["category"] != "legal" {
			t.Errorf("unexpected chunk category: %s (ID=%s)", rc.Chunk.Metadata["category"], rc.Chunk.ID)
		}
	}
}

// TestPipeline_AnswerWithMetadataFilter_FiltersNotSupported проверяет, что AnswerWithMetadataFilter
// возвращает ErrFiltersNotSupported на non-filter store (DEC-003).
func TestPipeline_AnswerWithMetadataFilter_FiltersNotSupported(t *testing.T) {
	ctx := context.Background()
	p := NewPipeline(&noFilterStore{}, testLLM{}, testEmbedder{})

	_, err := p.AnswerWithMetadataFilter(ctx, "cat", 5, domain.MetadataFilter{
		Fields: map[string]string{"category": "legal"},
	})
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

// noFilterStore — минимальный VectorStore без VectorStoreWithFilters capability.
type noFilterStore struct{}

func (noFilterStore) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (noFilterStore) Delete(_ context.Context, _ string) error       { return nil }
func (noFilterStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}
