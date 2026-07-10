package draftrag

import (
	"context"
	"errors"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubEmbedder struct{}

func (stubEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{0.5, 0.5}, nil
}

func (stubEmbedder) Health(_ context.Context) error { return nil }

var _ Chunker = (*semanticChunkerImpl)(nil)

// @sk-test chunker-semantic#T4.1: TestNewSemanticChunker_InvalidConfig (AC-007)
func TestNewSemanticChunker_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		opts SemanticChunkerOptions
	}{
		{"nil embedder", SemanticChunkerOptions{Embedder: nil, SimilarityThreshold: 0.5}},
		{"threshold below 0", SemanticChunkerOptions{Embedder: stubEmbedder{}, SimilarityThreshold: -0.1}},
		{"threshold above 1", SemanticChunkerOptions{Embedder: stubEmbedder{}, SimilarityThreshold: 1.1}},
		{"min chunk negative", SemanticChunkerOptions{Embedder: stubEmbedder{}, SimilarityThreshold: 0.5, MinChunkSize: -1}},
		{"max chunk negative", SemanticChunkerOptions{Embedder: stubEmbedder{}, SimilarityThreshold: 0.5, MaxChunkSize: -1}},
		{"min > max", SemanticChunkerOptions{Embedder: stubEmbedder{}, SimilarityThreshold: 0.5, MinChunkSize: 100, MaxChunkSize: 50}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSemanticChunker(tc.opts)
			if !errors.Is(err, ErrInvalidChunkerConfig) {
				t.Fatalf("expected ErrInvalidChunkerConfig, got %v", err)
			}
		})
	}
}

// @sk-test chunker-semantic#T4.1: TestNewSemanticChunker_ValidConfig (AC-007)
func TestNewSemanticChunker_ValidConfig(t *testing.T) {
	ch, err := NewSemanticChunker(SemanticChunkerOptions{
		Embedder:            stubEmbedder{},
		SimilarityThreshold: 0.7,
		MinChunkSize:        50,
		MaxChunkSize:        1000,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil chunker")
	}
}

// @sk-test chunker-semantic#T4.1: TestNewSemanticChunker_ChunkWithStubEmbedder (AC-001 smoke)
func TestNewSemanticChunker_ChunkWithStubEmbedder(t *testing.T) {
	ch, err := NewSemanticChunker(SemanticChunkerOptions{
		Embedder:            stubEmbedder{},
		SimilarityThreshold: 0.5,
		MinChunkSize:        0,
		MaxChunkSize:        0,
	})
	require.NoError(t, err)

	chunks, err := ch.Chunk(context.Background(), domain.Document{
		ID:      "test",
		Content: "First sentence here. Second sentence here. Third sentence here.",
	})
	require.NoError(t, err)

	// With stub embedder returning all-same vectors and threshold 0.5,
	// all sentences are "similar" → one chunk.
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (all similar), got %d", len(chunks))
	}
}

// @sk-test chunker-semantic#T4.2: TestPipelineFromConfig_SemanticChunker (AC-009)
func TestPipelineFromConfig_SemanticChunker(t *testing.T) {
	yamlContent := `
store:
  type: memory
embedder:
  type: ollama
  ollama:
    model: nomic-embed-text
    base_url: http://localhost:11434
llm:
  type: ollama
  ollama:
    model: llama3
    base_url: http://localhost:11434
chunker:
  type: semantic
  semantic:
    threshold: 0.75
    min_chunk_size: 100
    max_chunk_size: 2000
`
	path := writeTempYAML(t, yamlContent)

	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "semantic", cfg.Chunker.Type)
	require.NotNil(t, cfg.Chunker.Semantic)
	assert.Equal(t, 0.75, cfg.Chunker.Semantic.SimilarityThreshold)
	assert.Equal(t, 100, cfg.Chunker.Semantic.MinChunkSize)
	assert.Equal(t, 2000, cfg.Chunker.Semantic.MaxChunkSize)
}
