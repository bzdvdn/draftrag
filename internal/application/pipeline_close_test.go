package application

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test arch-issues#T3.3: TestPipeline_CloseThenHealthReturnsSentinel (AC-008)
func TestPipeline_CloseThenHealthReturnsSentinel(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err = p.Health(context.Background())
	if !errors.Is(err, ErrPipelineClosed) {
		t.Fatalf("expected ErrPipelineClosed after close, got %v", err)
	}
}

// @sk-test arch-issues#T3.3: TestPipeline_DoubleCloseIsNoop (AC-008)
func TestPipeline_DoubleCloseIsNoop(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// @sk-test arch-issues#T3.3: TestPipeline_CloseThenIndexReturnsSentinel (AC-008)
func TestPipeline_CloseThenIndexReturnsSentinel(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = p.Index(ctx, nil)
	if !errors.Is(err, ErrPipelineClosed) {
		t.Fatalf("expected ErrPipelineClosed, got %v", err)
	}
}

// @sk-test arch-issues#T3.3: TestPipeline_CloseThenQueryReturnsSentinel (AC-008)
func TestPipeline_CloseThenQueryReturnsSentinel(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = p.Query(context.Background(), "test", 5)
	if !errors.Is(err, ErrPipelineClosed) {
		t.Fatalf("expected ErrPipelineClosed, got %v", err)
	}
}

// @sk-test arch-issues#T3.3: TestPipeline_CloseThenAnswerReturnsSentinel (AC-008)
func TestPipeline_CloseThenAnswerReturnsSentinel(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "test", 5)
	if !errors.Is(err, ErrPipelineClosed) {
		t.Fatalf("expected ErrPipelineClosed, got %v", err)
	}
}

// @sk-test arch-issues#T3.3: TestPipeline_CloseThenUpdateDocumentReturnsSentinel (AC-008)
func TestPipeline_CloseThenUpdateDocumentReturnsSentinel(t *testing.T) {
	p, err := NewPipeline(vectorstore.NewInMemoryStore(), testLLM{}, testEmbedder{})
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	err = p.UpdateDocument(context.Background(), domain.Document{ID: "doc-1", Content: "hello"})
	if !errors.Is(err, ErrPipelineClosed) {
		t.Fatalf("expected ErrPipelineClosed, got %v", err)
	}
}
