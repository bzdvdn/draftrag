package draftrag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var _ Chunker = NewBasicChunker(BasicChunkerOptions{})

func TestBasicChunker_ConfigValidation_ErrorsIs(t *testing.T) {
	ch := NewBasicChunker(BasicChunkerOptions{
		ChunkSize: 0,
		Overlap:   0,
		MaxChunks: 0,
	})

	doc := domain.Document{ID: "doc-1", Content: "hello"}
	_, err := ch.Chunk(context.Background(), doc)
	if !errors.Is(err, ErrInvalidChunkerConfig) {
		t.Fatalf("expected ErrInvalidChunkerConfig, got %v", err)
	}
}

func TestBasicChunker_ContextCancel(t *testing.T) {
	ch := NewBasicChunker(BasicChunkerOptions{
		ChunkSize: 3,
		Overlap:   0,
		MaxChunks: 0,
	})

	doc := domain.Document{ID: "doc-1", Content: "abcdef"}

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	_, err := ch.Chunk(ctx, doc)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}
