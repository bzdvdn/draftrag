package chunker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestBasicRuneChunker_Chunk_DeterministicAndFields(t *testing.T) {
	doc := domain.Document{ID: "doc-1", Content: "abcdefg"}

	ch := NewBasicRuneChunker(3, 0, 0)
	got1, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	got2, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got1) != len(got2) {
		t.Fatalf("expected deterministic length, got %d vs %d", len(got1), len(got2))
	}

	wantContents := []string{"abc", "def", "g"}
	if len(got1) != len(wantContents) {
		t.Fatalf("expected %d chunks, got %d", len(wantContents), len(got1))
	}
	for i, c := range got1 {
		if c.ParentID != doc.ID {
			t.Fatalf("chunk[%d] ParentID: expected %q, got %q", i, doc.ID, c.ParentID)
		}
		if c.Position != i {
			t.Fatalf("chunk[%d] Position: expected %d, got %d", i, i, c.Position)
		}
		wantID := fmt.Sprintf("%s:%d", doc.ID, i)
		if c.ID != wantID {
			t.Fatalf("chunk[%d] ID: expected %q, got %q", i, wantID, c.ID)
		}
		if strings.TrimSpace(c.Content) == "" {
			t.Fatalf("chunk[%d] Content expected non-empty", i)
		}
		if c.Content != wantContents[i] {
			t.Fatalf("chunk[%d] Content: expected %q, got %q", i, wantContents[i], c.Content)
		}
		if got2[i].ID != c.ID ||
			got2[i].ParentID != c.ParentID ||
			got2[i].Position != c.Position ||
			got2[i].Content != c.Content {
			t.Fatalf("chunk[%d] expected deterministic chunk, got %#v vs %#v", i, got2[i], c)
		}
	}
}

func TestBasicRuneChunker_Chunk_Overlap(t *testing.T) {
	doc := domain.Document{ID: "doc-1", Content: "abcdef"}

	ch := NewBasicRuneChunker(4, 2, 0)
	got, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	if got[0].Content != "abcd" {
		t.Fatalf("chunk[0] expected %q, got %q", "abcd", got[0].Content)
	}
	if got[1].Content != "cdef" {
		t.Fatalf("chunk[1] expected %q, got %q", "cdef", got[1].Content)
	}
	if got[0].Content[len(got[0].Content)-2:] != got[1].Content[:2] {
		t.Fatalf("expected overlap of 2 runes between chunks")
	}
}

func TestBasicRuneChunker_Chunk_MaxChunksLimitsReturn(t *testing.T) {
	doc := domain.Document{ID: "doc-1", Content: "abcdef"}

	ch := NewBasicRuneChunker(2, 0, 2)
	got, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks due to MaxChunks, got %d", len(got))
	}
}

func TestBasicRuneChunker_Chunk_ContextCancelFast(t *testing.T) {
	doc := domain.Document{ID: "doc-1", Content: strings.Repeat("a", 1_000_000)}
	ch := NewBasicRuneChunker(100, 10, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := ch.Chunk(ctx, doc)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}

func TestBasicRuneChunker_Chunk_ContextDeadlineFast(t *testing.T) {
	doc := domain.Document{ID: "doc-1", Content: strings.Repeat("a", 1_000_000)}
	ch := NewBasicRuneChunker(100, 10, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	t.Cleanup(cancel)
	time.Sleep(2 * time.Millisecond)

	start := time.Now()
	_, err := ch.Chunk(ctx, doc)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected deadline within 100ms, took %v", time.Since(start))
	}
}
