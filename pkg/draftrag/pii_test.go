package draftrag

import (
	"context"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/infrastructure/piidetector"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// mockEmbedder — примитивный embedder для тестов (фиксированный вектор).
type mockPIIEmbedder struct{}

func (mockPIIEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	return []float64{1.0, 0.0, 0.0}, nil
}

func (mockPIIEmbedder) Health(_ context.Context) error { return nil }

// mockPIILLM — заглушка LLM, возвращающая фиксированный ответ.
type mockPIILLM struct{}

func (mockPIILLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "answer", nil
}

func (mockPIILLM) Health(_ context.Context) error { return nil }

// @sk-test pii-guardrails#T2.4: TestPIIRedactIndex (AC-001, RQ-001)
func TestPIIRedactIndex(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	det := piidetector.NewDefaultPIIDetector(piidetector.PIICategories{Email: true, Phone: true})

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector: det,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{
			ID:      "test1",
			Content: "contact: user@example.com, phone: +1-555-123-4567",
		},
	}

	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	raw, err := store.Search(ctx, []float64{1.0, 0.0, 0.0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range raw.Chunks {
		if strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("email was not redacted in store")
		}
		if strings.Contains(ch.Chunk.Content, "+1-555-123-4567") {
			t.Error("phone was not redacted in store")
		}
		if !strings.Contains(ch.Chunk.Content, "<redacted>") {
			t.Error("expected <redacted> marker in stored content")
		}
	}
}

// @sk-test pii-guardrails#T2.4: TestPIIRedactQuery (AC-002, RQ-002)
func TestPIIRedactQuery(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	det := piidetector.NewDefaultPIIDetector(piidetector.PIICategories{Email: true})

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector: det,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Index a document with PII, then query
	docs := []Document{
		{ID: "test1", Content: "email: user@example.com"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	result, err := p.Query(ctx, "email")
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range result.Chunks {
		if strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("email was not redacted in query result")
		}
	}
}

// @sk-test pii-guardrails#T2.4: TestPIIRedactBackwardCompat (AC-006, RQ-005)
func TestPIIRedactBackwardCompat(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()

	// nil PIIDetector — no-op
	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{ID: "test1", Content: "email: user@example.com"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	result, err := p.Query(ctx, "email")
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range result.Chunks {
		if !strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("expected content unchanged without PIIDetector")
		}
	}
}

// @sk-test pii-guardrails#T3.4: TestPIIRedactCustomDetector (AC-005, RQ-004)
func TestPIIRedactCustomDetector(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()

	// кастомный детектор: цензурирует passport numbers
	customDet := &passportDetector{}

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector: customDet,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{ID: "test1", Content: "passport: AB123456"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	raw, err := store.Search(ctx, []float64{1.0, 0.0, 0.0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range raw.Chunks {
		if strings.Contains(ch.Chunk.Content, "AB123456") {
			t.Error("passport number was not redacted by custom detector")
		}
	}
}

// @sk-test pii-guardrails#T4.1: TestPIIRedactCite (AC-003)
func TestPIIRedactCite(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	det := piidetector.NewDefaultPIIDetector(piidetector.PIICategories{Email: true})

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector: det,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{ID: "test1", Content: "email: user@example.com"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	_, sources, err := p.Search("email").TopK(5).Cite(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range sources.Chunks {
		if strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("email was not redacted in Cite sources")
		}
	}
}

// @sk-test pii-guardrails#T4.1: TestPIIRedactSelectiveCategories (AC-004, RQ-006)
func TestPIIRedactSelectiveCategories(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()

	// Only email, no phone
	det := piidetector.NewDefaultPIIDetector(piidetector.PIICategories{Email: true})

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector: det,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{
			ID:      "test1",
			Content: "email: user@example.com, phone: +1-555-123-4567",
		},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	raw, err := store.Search(ctx, []float64{1.0, 0.0, 0.0}, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range raw.Chunks {
		if strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("email was not redacted")
		}
		if !strings.Contains(ch.Chunk.Content, "+1-555-123-4567") {
			t.Error("phone was unexpectedly redacted (should remain)")
		}
	}
}

// passportDetector — пример кастомного PII-детектора для тестов.
type passportDetector struct{}

func (d *passportDetector) Detect(text string) string {
	// Упрощённый pattern для passport: 2 буквы + 6 цифр
	var result strings.Builder
	i := 0
	for i < len(text) {
		if i+8 <= len(text) && isLetter(text[i]) && isLetter(text[i+1]) &&
			isDigit(text[i+2]) && isDigit(text[i+3]) && isDigit(text[i+4]) &&
			isDigit(text[i+5]) && isDigit(text[i+6]) && isDigit(text[i+7]) {
			result.WriteString("<redacted>")
			i += 8
		} else {
			result.WriteByte(text[i])
			i++
		}
	}
	return result.String()
}

func isLetter(c byte) bool { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') }
func isDigit(c byte) bool  { return c >= '0' && c <= '9' }

// @sk-test pii-guardrails#T5.2: TestPIIRedactRewrittenQuery (AC-007, RQ-007)
//
// Прямой тест AC-007: Pipeline с QueryRewriter + PIIDetector.
// Mock-rewriter возвращает rewritten queries с PII; rewriterResult
// применяет PIIDetector, цензурируя PII перед retrieval.
func TestPIIRedactRewrittenQuery(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	det := piidetector.NewDefaultPIIDetector(piidetector.PIICategories{Email: true, Phone: true})

	// rewriter возвращает переформулированные запросы, содержащие PII
	rw := &mockRewriter{
		rewriteFn: func(_ context.Context, q string, _ QueryHistory) ([]RewrittenQuery, error) {
			return []RewrittenQuery{
				{Query: "email: user@example.com phone: +1-555-123-4567", Weight: 1.0},
				{Query: q, Weight: 0.5},
			}, nil
		},
	}

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector: det,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{ID: "test1", Content: "some content about contacts"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	// routeRewriter: через SearchBuilder.Rewriter
	result, err := p.Search("original query").TopK(5).Rewriter(rw).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// PII не должно быть в результатах (redaction после retrieval)
	for _, ch := range result.Chunks {
		if strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("email was not redacted in rewriter-retrieve result")
		}
		if strings.Contains(ch.Chunk.Content, "+1-555-123-4567") {
			t.Error("phone was not redacted in rewriter-retrieve result")
		}
	}
}

// @sk-test pii-guardrails#T5.2: TestPIIRedactRewrittenQueryPipelineLevel (AC-007)
//
// AC-007 с pipeline-level rewriter (не per-request).
func TestPIIRedactRewrittenQueryPipelineLevel(t *testing.T) {
	ctx := context.Background()
	store := vectorstore.NewInMemoryStore()
	det := piidetector.NewDefaultPIIDetector(piidetector.PIICategories{Email: true})

	rw := &mockRewriter{
		rewriteFn: func(_ context.Context, q string, _ QueryHistory) ([]RewrittenQuery, error) {
			return []RewrittenQuery{
				{Query: "email: user@example.com", Weight: 1.0},
			}, nil
		},
	}

	p, err := NewPipelineWithOptions(store, mockPIILLM{}, mockPIIEmbedder{}, PipelineOptions{
		PIIDetector:   det,
		QueryRewriter: rw,
	})
	if err != nil {
		t.Fatal(err)
	}

	docs := []Document{
		{ID: "test1", Content: "contacts info"},
	}
	if err := p.Index(ctx, docs); err != nil {
		t.Fatal(err)
	}

	// pipeline-level rewriter (без per-request Rewriter)
	result, err := p.Search("original query").TopK(5).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, ch := range result.Chunks {
		if strings.Contains(ch.Chunk.Content, "user@example.com") {
			t.Error("email was not redacted with pipeline-level rewriter")
		}
	}
}
