package chunker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type stubChunker struct {
	chunks []domain.Chunk
	err    error
}

func (s *stubChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make([]domain.Chunk, len(s.chunks))
	for i, c := range s.chunks {
		result[i] = c
		if result[i].ID == "" {
			result[i].ID = doc.ID
		}
		if result[i].ParentID == "" {
			result[i].ParentID = doc.ID
		}
	}
	return result, nil
}

func makeTestChunks(contents ...string) []domain.Chunk {
	chunks := make([]domain.Chunk, len(contents))
	for i, c := range contents {
		chunks[i] = domain.Chunk{
			ID:       string(rune('a' + i)),
			Content:  c,
			ParentID: "doc-1",
			Position: i,
		}
	}
	return chunks
}

// @sk-test contextual-chunking#T2.1: TestContextualChunker_DefaultTemplate (AC-001)
func TestContextualChunker_DefaultTemplate(t *testing.T) {
	base := &stubChunker{chunks: makeTestChunks("hello world", "second chunk")}
	ch := NewContextualChunker(base, "title", "[CONTEXT] {context}\n{content}")

	doc := domain.Document{
		ID:       "doc-1",
		Content:  "hello world second chunk",
		Metadata: map[string]string{"title": "Research Paper"},
	}
	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if !strings.HasPrefix(c.Content, "[CONTEXT] Research Paper\n") {
			t.Fatalf("chunk[%d] expected context prefix, got %q", i, c.Content)
		}
	}
}

// @sk-test contextual-chunking#T2.1: TestContextualChunker_CustomTemplate (AC-002)
func TestContextualChunker_CustomTemplate(t *testing.T) {
	base := &stubChunker{chunks: makeTestChunks("hello world")}
	ch := NewContextualChunker(base, "title", "Doc: {context} --- {content}")

	doc := domain.Document{
		ID:       "doc-1",
		Content:  "hello world",
		Metadata: map[string]string{"title": "Report"},
	}
	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !strings.HasPrefix(chunks[0].Content, "Doc: ") {
		t.Fatalf("expected prefix 'Doc: ', got %q", chunks[0].Content)
	}
	if !strings.Contains(chunks[0].Content, " --- ") {
		t.Fatalf("expected separator ' --- ', got %q", chunks[0].Content)
	}
	if !strings.HasSuffix(chunks[0].Content, "hello world") {
		t.Fatalf("expected content suffix 'hello world', got %q", chunks[0].Content)
	}
}

// @sk-test contextual-chunking#T2.1: TestContextualChunker_EmptyMetadata (AC-003)
func TestContextualChunker_EmptyMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
	}{
		{"nil metadata", nil},
		{"missing key", map[string]string{"other": "val"}},
		{"empty value", map[string]string{"title": ""}},
	}

	base := &stubChunker{chunks: makeTestChunks("hello world")}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := NewContextualChunker(base, "title", "[CONTEXT] {context}\n{content}")
			doc := domain.Document{ID: "doc-1", Content: "hello world", Metadata: tc.metadata}
			chunks, err := ch.Chunk(context.Background(), doc)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if len(chunks) != 1 {
				t.Fatalf("expected 1 chunk, got %d", len(chunks))
			}
			if chunks[0].Content != "hello world" {
				t.Fatalf("expected unchanged content %q, got %q", "hello world", chunks[0].Content)
			}
		})
	}
}

// @sk-test contextual-chunking#T2.1: TestContextualChunker_ContextCancel (AC-004)
func TestContextualChunker_ContextCancel(t *testing.T) {
	base := &stubChunker{chunks: makeTestChunks("hello world")}
	ch := NewContextualChunker(base, "title", "[CONTEXT] {context}\n{content}")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := ch.Chunk(ctx, domain.Document{
		ID:       "doc-1",
		Content:  "hello world",
		Metadata: map[string]string{"title": "test"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}

// @sk-test contextual-chunking#T2.1: TestContextualChunker_CustomContextKey (AC-006)
func TestContextualChunker_CustomContextKey(t *testing.T) {
	base := &stubChunker{chunks: makeTestChunks("hello world")}
	ch := NewContextualChunker(base, "description", "[CONTEXT] {context}\n{content}")

	doc := domain.Document{
		ID:       "doc-1",
		Content:  "hello world",
		Metadata: map[string]string{"description": "Annual Report 2025"},
	}
	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !strings.HasPrefix(chunks[0].Content, "[CONTEXT] Annual Report 2025\n") {
		t.Fatalf("expected context from description key, got %q", chunks[0].Content)
	}
}
