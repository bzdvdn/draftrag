package application

import (
	"context"
	"errors"
	"fmt"
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

	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Query(ctx, "cat", 5)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPipeline_FullCycle(t *testing.T) {
	ctx := context.Background()
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

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
	p, err := NewPipeline(&noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.QueryWithParentIDs(ctx, "cat", 5, []string{"doc-1"})
	if !errors.Is(err, ErrFiltersNotSupported) {
		t.Fatalf("expected ErrFiltersNotSupported, got %v", err)
	}
}

func TestPipeline_QueryWithParentIDs_EmptyFilterFallsBack(t *testing.T) {
	ctx := context.Background()
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	// Пустой фильтр не должен требовать capability.
	_, err = p.QueryWithParentIDs(ctx, "cat", 5, nil)
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
	p, err := NewPipeline(&noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.QueryWithMetadataFilter(ctx, "cat", 5, domain.MetadataFilter{
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
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.QueryWithMetadataFilter(ctx, "cat", 5, domain.MetadataFilter{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// TestPipeline_QueryWithMetadataFilter_PassesFilterToStore проверяет, что фильтр передаётся
// в SearchWithMetadataFilter и возвращаются только совпадающие чанки (AC-003).
func TestPipeline_QueryWithMetadataFilter_PassesFilterToStore(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

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
	p, err := NewPipeline(&noFilterStore{}, testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.AnswerWithMetadataFilter(ctx, "cat", 5, domain.MetadataFilter{
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

// failOnEmbedder — тестовый Embedder, который возвращает ошибку для текста,
// содержащего failOn. Используется для проверки best-effort UpdateDocument.
type failOnEmbedder struct {
	failOn string
}

func (e *failOnEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.failOn != "" && text == e.failOn {
		return nil, fmt.Errorf("embed failed for %q", text)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

// @sk-test api-consistency-pass#T3.2: best-effort UpdateDocument на in-memory store
// (без TransactionalDocumentStore capability) должен вернуть ErrUpdateNotAtomic
// при ошибке Embed ПОСЛЕ успешного DeleteByParentID (DEC-005, RQ-005, AC-009).
//
// Сценарий:
//  1. Index документа с успешным embedder — store содержит 1 чанк.
//  2. UpdateDocument с failing embedder — DeleteByParentID успешен, Embed падает.
//  3. Возвращённая ошибка классифицируется через errors.Is(err, domain.ErrUpdateNotAtomic) == true.
//  4. Underlying error (от embed) сохранён в error chain (для диагностики).
func TestPipeline_UpdateDocument_BestEffort_ReturnsErrUpdateNotAtomic(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, &failOnEmbedder{failOn: "boom"})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Index исходного документа — успешный.
	if err := p.Index(ctx, []domain.Document{
		{ID: "doc-1", Content: "ok"},
	}); err != nil {
		t.Fatalf("index: %v", err)
	}

	// 2. UpdateDocument с failing embedder — должен вернуть ErrUpdateNotAtomic.
	err = p.UpdateDocument(ctx, domain.Document{ID: "doc-1", Content: "boom"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrUpdateNotAtomic) {
		t.Fatalf("expected ErrUpdateNotAtomic, got %v", err)
	}
	// Underlying error от embed должен быть в chain.
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected underlying error to mention 'boom', got %v", err)
	}
}

// @sk-test api-consistency-pass#T3.2: best-effort path — при ошибке DeleteByParentID
// (до Index) возвращается underlying error БЕЗ wrapping в ErrUpdateNotAtomic.
// Семантика: ErrUpdateNotAtomic применим только если delete успел, а index — нет.
func TestPipeline_UpdateDocument_BestEffort_DeleteErrorPropagatesRaw(t *testing.T) {
	ctx := context.Background()
	// noFilterStore не реализует DocumentStore, поэтому DeleteByParentID вернёт
	// ErrDeleteNotSupported напрямую.
	p, err := NewPipeline(&noFilterStore{}, testLLM{}, &failOnEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	err = p.UpdateDocument(ctx, domain.Document{ID: "doc-1", Content: "ok"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, domain.ErrUpdateNotAtomic) {
		t.Fatalf("expected ErrDeleteNotSupported (raw), got wrapped ErrUpdateNotAtomic: %v", err)
	}
	if !errors.Is(err, ErrDeleteNotSupported) {
		t.Fatalf("expected ErrDeleteNotSupported, got %v", err)
	}
}

// @sk-test api-consistency-pass#T3.2: UpdateDocument валидирует doc ПЕРЕД delete.
// Пустой doc.Content → ErrEmptyDocumentContent, store не трогается.
func TestPipeline_UpdateDocument_ValidationFailsBeforeDelete(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLM{}, &failOnEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	// Сначала проиндексируем валидный doc, чтобы в store было что удалять.
	if err := p.Index(ctx, []domain.Document{
		{ID: "doc-1", Content: "ok"},
	}); err != nil {
		t.Fatalf("index: %v", err)
	}

	// UpdateDocument с пустым Content → должен вернуть ErrEmptyDocumentContent
	// и НЕ удалить существующие чанки.
	err = p.UpdateDocument(ctx, domain.Document{ID: "doc-1", Content: ""})
	if !errors.Is(err, domain.ErrEmptyDocumentContent) {
		t.Fatalf("expected ErrEmptyDocumentContent, got %v", err)
	}
	// Чанк остался в store (delete не выполнился).
	result, err := store.Search(ctx, []float64{0.1, 0.2, 0.3}, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected old chunks to remain in store after validation failure")
	}
}
