package chunker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// topicEmbedder returns one of two topic vectors based on keyword dominance.
type topicEmbedder struct {
	aKeyword string
	aVec     []float64
	bKeyword string
	bVec     []float64
	mixedVec []float64
}

func (e *topicEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	lower := strings.ToLower(text)
	aCount := strings.Count(lower, e.aKeyword)
	bCount := strings.Count(lower, e.bKeyword)
	if aCount > bCount {
		return e.aVec, nil
	}
	if bCount > aCount {
		return e.bVec, nil
	}
	return e.mixedVec, nil
}

func (e *topicEmbedder) Health(_ context.Context) error { return nil }

type constEmbedder struct {
	vec []float64
	err error
}

func (m *constEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vec, nil
}

func (m *constEmbedder) Health(_ context.Context) error { return nil }

func doc(id, content string) domain.Document {
	return domain.Document{ID: id, Content: content}
}

// Vector arithmetic reference:
// pureA=[1,0,0.3], pureB=[0,1,0.3], mixed=[0.1,0.1,0.3] (aCount=bCount=1)
// cos(pureA, mixed) = (0.1+0.09)/(1.044*0.332) = 0.19/0.347 = 0.548
// cos(pureA, pureA) = 1.0
// cos(mixed, pureB) = 0.548
// cos(pureA, pureB) = 0.09/1.09 = 0.083
// Threshold 0.6: pureA→mixed splits (0.548<0.6), mixed→mixed keeps (1.0≥0.6)
// Threshold 0.1: pureA→mixed keeps (0.548≥0.1), mixed→mixed keeps

func topicEmb() *topicEmbedder {
	return &topicEmbedder{
		aKeyword: "weather",
		aVec:     []float64{1, 0, 0.3},
		bKeyword: "recursion",
		bVec:     []float64{0, 1, 0.3},
		mixedVec: []float64{0.1, 0.1, 0.3},
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_TwoTopics (AC-001)
func TestSemanticChunker_TwoTopics(t *testing.T) {
	emb := topicEmb()
	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder:            emb,
		SimilarityThreshold: 0.6,
		MinChunkSize:        0,
		MaxChunkSize:        0,
	})

	chunks, err := ch.Chunk(context.Background(), doc("d1",
		"The weather is cold and rainy. "+
			"Recursion is a technique in programming. A recursive function calls itself.",
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected >= 2 chunks for two topics, got %d", len(chunks))
	}
	if strings.Contains(strings.ToLower(chunks[0].Content), "recursion") {
		t.Fatalf("first chunk should not contain second topic, got: %s", chunks[0].Content)
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_ThresholdEffect (AC-002)
func TestSemanticChunker_ThresholdEffect(t *testing.T) {
	emb := topicEmb()

	chLow := NewSemanticChunker(SemanticChunkerOptions{
		Embedder: emb, SimilarityThreshold: 0.1, MinChunkSize: 0, MaxChunkSize: 0,
	})
	chHigh := NewSemanticChunker(SemanticChunkerOptions{
		Embedder: emb, SimilarityThreshold: 0.6, MinChunkSize: 0, MaxChunkSize: 0,
	})

	docContent := "The weather is cold and wet. " +
		"Recursion is a technique in programming. A recursive function calls itself."

	chunksLow, _ := chLow.Chunk(context.Background(), doc("d1", docContent))
	chunksHigh, _ := chHigh.Chunk(context.Background(), doc("d1", docContent))

	if len(chunksHigh) < len(chunksLow) {
		t.Fatalf("higher threshold (%d chunks) should produce >= chunks than lower threshold (%d chunks)",
			len(chunksHigh), len(chunksLow))
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_MinChunkSize (AC-003)
func TestSemanticChunker_MinChunkSize(t *testing.T) {
	emb := topicEmb()

	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder:            emb,
		SimilarityThreshold: 0.6,
		MinChunkSize:        120,
		MaxChunkSize:        0,
	})

	chunks, err := ch.Chunk(context.Background(), doc("d1",
		"The weather is nice today with clear skies. "+
			"Recursion is a deep concept in computer programming. "+
			"Weather patterns change dramatically with the seasons.",
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk due to MinChunkSize=120, got %d", len(chunks))
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_MaxChunkSize (AC-004)
func TestSemanticChunker_MaxChunkSize(t *testing.T) {
	emb := &constEmbedder{vec: []float64{0.5, 0.5, 0.5}}
	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder:            emb,
		SimilarityThreshold: 0.5,
		MinChunkSize:        0,
		MaxChunkSize:        30,
	})

	chunks, err := ch.Chunk(context.Background(), doc("d1",
		"Short sentence one. Short sentence two. Short sentence three. Short sentence four.",
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, c := range chunks {
		if len([]rune(c.Content)) > 30 {
			t.Fatalf("chunk[%d] exceeds MaxChunkSize: %d chars (%q)", i, len([]rune(c.Content)), c.Content)
		}
		if !strings.HasSuffix(strings.TrimSpace(c.Content), ".") {
			t.Fatalf("chunk[%d] does not end with sentence boundary: %q", i, c.Content)
		}
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_SentenceIntegrity (AC-005)
func TestSemanticChunker_SentenceIntegrity(t *testing.T) {
	emb := &constEmbedder{vec: []float64{0.3, 0.3, 0.3}}
	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder: emb, SimilarityThreshold: 0.5, MinChunkSize: 0, MaxChunkSize: 0,
	})

	chunks, err := ch.Chunk(context.Background(), doc("d1",
		"First sentence here. Second sentence here. Third sentence here.",
	))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range chunks {
		content := strings.TrimSpace(c.Content)
		if !strings.HasSuffix(content, ".") && !strings.HasSuffix(content, "!") && !strings.HasSuffix(content, "?") {
			if c.Position != len(chunks)-1 {
				t.Fatalf("chunk[%d] does not end with sentence boundary: %q", c.Position, content)
			}
		}
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_ContextCancel (AC-006)
func TestSemanticChunker_ContextCancel(t *testing.T) {
	emb := &constEmbedder{vec: []float64{1, 0, 0}}
	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder: emb, SimilarityThreshold: 0.5, MinChunkSize: 0, MaxChunkSize: 0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ch.Chunk(ctx, doc("d1", "Some content. More content."))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_EmptyDoc (AC-008)
func TestSemanticChunker_EmptyDoc(t *testing.T) {
	emb := &constEmbedder{vec: []float64{1, 0, 0}}
	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder: emb, SimilarityThreshold: 0.5, MinChunkSize: 0, MaxChunkSize: 0,
	})

	chunks, err := ch.Chunk(context.Background(), doc("d1", ""))
	if err != nil {
		t.Fatalf("expected no error for empty doc, got %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty doc, got %d", len(chunks))
	}
}

// @sk-test chunker-semantic#T4.1: TestSemanticChunker_WhitespaceOnlyDoc (AC-008)
func TestSemanticChunker_WhitespaceOnlyDoc(t *testing.T) {
	emb := &constEmbedder{vec: []float64{1, 0, 0}}
	ch := NewSemanticChunker(SemanticChunkerOptions{
		Embedder: emb, SimilarityThreshold: 0.5, MinChunkSize: 0, MaxChunkSize: 0,
	})

	chunks, err := ch.Chunk(context.Background(), doc("d1", "   \n\n   \t  "))
	if err != nil {
		t.Fatalf("expected no error for whitespace doc, got %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for whitespace doc, got %d", len(chunks))
	}
}
