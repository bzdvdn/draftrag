package application

import (
	"context"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

type captureLLM struct {
	systemPrompt string
}

func (l *captureLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	l.systemPrompt = systemPrompt
	return "ok", nil
}

func TestPipelineConfig_SystemPromptOverride(t *testing.T) {
	llm := &captureLLM{}
	p := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		llm,
		testEmbedder{},
		PipelineConfig{SystemPrompt: "X"},
	)

	_, err := p.Answer(context.Background(), "cat", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if llm.systemPrompt != "X" {
		t.Fatalf("expected system prompt %q, got %q", "X", llm.systemPrompt)
	}
}

func TestPipelineConfig_ChunkerEnablesChunkingPath(t *testing.T) {
	store := vectorstore.NewInMemoryStore()
	emb := &countingEmbedder{}

	ch := fixedChunker{
		chunks: []domain.Chunk{
			{ID: "doc-1:0", ParentID: "doc-1", Position: 0, Content: "a"},
			{ID: "doc-1:1", ParentID: "doc-1", Position: 1, Content: "b"},
		},
	}

	p := NewPipelineWithConfig(store, testLLM{}, emb, PipelineConfig{Chunker: ch})

	if err := p.Index(context.Background(), []domain.Document{{ID: "doc-1", Content: "ignored"}}); err != nil {
		t.Fatalf("index: %v", err)
	}
	if emb.calls != 2 {
		t.Fatalf("expected 2 Embed calls, got %d", emb.calls)
	}

	res, err := store.Search(context.Background(), []float64{1, 1}, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.TotalFound != 2 || len(res.Chunks) != 2 {
		t.Fatalf("expected 2 chunks in store, got total=%d len=%d", res.TotalFound, len(res.Chunks))
	}
}
