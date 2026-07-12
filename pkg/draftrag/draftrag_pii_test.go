package draftrag

import (
	"context"
	"sync"
	"testing"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type publicCounterPII struct {
	mu    sync.Mutex
	calls int
}

func (c *publicCounterPII) Detect(text string) string {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return text
}

func (c *publicCounterPII) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type piiPublicTestLLM struct{}

func (piiPublicTestLLM) Health(_ context.Context) error { return nil }
func (piiPublicTestLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "safe", nil
}

type piiPublicTestEmbedder struct{}

func (piiPublicTestEmbedder) Health(_ context.Context) error { return nil }
func (piiPublicTestEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	return []float64{1, 0}, nil
}

func TestPublicPipeline_PIIRedactInIndex(t *testing.T) {
	pii := &publicCounterPII{}
	p, err := NewPipelineWithOptions(
		vectorstore.NewInMemoryStore(),
		piiPublicTestLLM{},
		piiPublicTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = p.Index(ctx, []Document{{ID: "doc-1", Content: "my email is test@example.com"}})
	if err != nil {
		t.Fatal(err)
	}

	// application.processDocumentOp вызывает redact() ровно 1 раз
	if pii.Count() != 1 {
		t.Fatalf("expected 1 PIIDetector call via public Index, got %d", pii.Count())
	}
}

func TestPublicPipeline_PIIRedactInQuery(t *testing.T) {
	store := vectorstore.NewInMemoryStore()

	// pre-index
	preP, err := NewPipelineWithOptions(store, piiPublicTestLLM{}, piiPublicTestEmbedder{}, PipelineOptions{})
	if err != nil {
		t.Fatal(err)
	}
	err = preP.Index(context.Background(), []Document{{ID: "doc-1", Content: "cat"}})
	if err != nil {
		t.Fatal(err)
	}

	pii := &publicCounterPII{}
	p, err := NewPipelineWithOptions(
		store,
		piiPublicTestLLM{},
		piiPublicTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := p.Query(ctx, "cat")
	if err != nil {
		t.Fatal(err)
	}

	// application.Query: redact(question) = 1 + RedactRetrievalResult на каждый чанк
	expected := 1 + len(result.Chunks)
	if pii.Count() != expected {
		t.Fatalf("expected %d PIIDetector calls via public Query, got %d", expected, pii.Count())
	}
}

func TestPublicPipeline_PIINilIsNoop(t *testing.T) {
	p, err := NewPipelineWithOptions(
		vectorstore.NewInMemoryStore(),
		piiPublicTestLLM{},
		piiPublicTestEmbedder{},
		PipelineOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = p.Index(ctx, []Document{{ID: "doc-1", Content: "test@example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Query(ctx, "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublicPipeline_PIIRedactInSearchBuilderRetrieve(t *testing.T) {
	store := vectorstore.NewInMemoryStore()

	preP, err := NewPipelineWithOptions(store, piiPublicTestLLM{}, piiPublicTestEmbedder{}, PipelineOptions{})
	if err != nil {
		t.Fatal(err)
	}
	err = preP.Index(context.Background(), []Document{{ID: "doc-1", Content: "cat"}})
	if err != nil {
		t.Fatal(err)
	}

	pii := &publicCounterPII{}
	p, err := NewPipelineWithOptions(
		store,
		piiPublicTestLLM{},
		piiPublicTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Search("cat").TopK(5).Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// SearchBuilder.Retrieve → routeBasic → application.Query:
	//   redact(question) + RedactRetrievalResult + SearchBuilder's own redactRetrievalResult (catch-all)
	// Проверяем что PII вызван хотя бы 1 раз — точное количество
	// зависит от числа чанков и routing path
	if pii.Count() < 1 {
		t.Fatalf("expected at least 1 PIIDetector call via SearchBuilder.Retrieve, got %d", pii.Count())
	}
}
