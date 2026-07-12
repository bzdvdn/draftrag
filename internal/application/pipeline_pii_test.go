package application

import (
	"context"
	"sync"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type counterPIIDetector struct {
	mu    sync.Mutex
	calls int
}

func (c *counterPIIDetector) Detect(text string) string {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return text
}

func (c *counterPIIDetector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type piiTestLLM struct{}

func (piiTestLLM) Health(_ context.Context) error { return nil }
func (piiTestLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "safe answer", nil
}

type piiTestEmbedder struct{}

func (piiTestEmbedder) Health(_ context.Context) error { return nil }
func (piiTestEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	return []float64{1, 0}, nil
}

func TestPipeline_PIIRedactInProcessDocumentOp(t *testing.T) {
	pii := &counterPIIDetector{}
	p, err := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		piiTestLLM{},
		piiTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = p.processDocumentOp(ctx, "Index", domain.Document{ID: "doc-1", Content: "hello my email is test@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if pii.Count() != 1 {
		t.Fatalf("expected 1 PIIDetector call, got %d", pii.Count())
	}
}

func TestPipeline_PIIRedactInQuery(t *testing.T) {
	pii := &counterPIIDetector{}
	p, err := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		piiTestLLM{},
		piiTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Query(ctx, "show me user@example.com data", 5)
	if err != nil {
		t.Fatal(err)
	}

	// redact on question + RedactRetrievalResult on result (0 chunks = 0 Detect calls from RedactRetrievalResult)
	// Но redact на question всегда 1 вызов независимо от результата
	if pii.Count() < 1 {
		t.Fatalf("expected at least 1 PIIDetector call (question), got %d", pii.Count())
	}
}

func TestPipeline_PIIRedactInAnswer(t *testing.T) {
	pii := &counterPIIDetector{}
	p, err := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		piiTestLLM{},
		piiTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = p.Answer(ctx, "show me user@example.com data", 5)
	if err != nil {
		t.Fatal(err)
	}

	// redact на question (1) + RedactRetrievalResult (0 chunks = 0 дополнительных)
	if pii.Count() < 1 {
		t.Fatalf("expected at least 1 PIIDetector call (question), got %d", pii.Count())
	}
}

func TestPipeline_PIINilIsNoop(t *testing.T) {
	p, err := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		piiTestLLM{},
		piiTestEmbedder{},
		PipelineOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = p.processDocumentOp(ctx, "Index", domain.Document{ID: "doc-1", Content: "test@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Query(ctx, "test@example.com", 5)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPipeline_PIIRedactInQueryWithResults(t *testing.T) {
	pii := &counterPIIDetector{}
	store := vectorstore.NewInMemoryStore()
	p, err := NewPipelineWithConfig(
		store,
		piiTestLLM{},
		piiTestEmbedder{},
		PipelineOptions{PIIDetector: pii},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = p.Index(ctx, []domain.Document{{ID: "doc-1", Content: "cat"}})
	if err != nil {
		t.Fatal(err)
	}

	pii2 := &counterPIIDetector{}
	p2, err := NewPipelineWithConfig(
		store,
		piiTestLLM{},
		piiTestEmbedder{},
		PipelineOptions{PIIDetector: pii2},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := p2.Query(ctx, "cat", 5)
	if err != nil {
		t.Fatal(err)
	}

	expectedCalls := 1 // redact(question)
	if len(result.Chunks) > 0 {
		expectedCalls += len(result.Chunks) // RedactRetrievalResult на каждый чанк
	}
	if pii2.Count() != expectedCalls {
		t.Fatalf("expected %d PIIDetector calls, got %d", expectedCalls, pii2.Count())
	}
}
