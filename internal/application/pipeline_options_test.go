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

func (l *captureLLM) Health(_ context.Context) error { return nil }
func (l *captureLLM) Generate(_ context.Context, systemPrompt, _ string) (string, error) {
	l.systemPrompt = systemPrompt
	return "ok", nil
}

func TestPipelineOptions_SystemPromptOverride(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	llm := &captureLLM{}
	p, err := NewPipelineWithConfig(
		vectorstore.NewInMemoryStore(),
		llm,
		testEmbedder{},
		PipelineOptions{SystemPrompt: "X"},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Answer(context.Background(), "cat", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if llm.systemPrompt != "X" {
		t.Fatalf("expected system prompt %q, got %q", "X", llm.systemPrompt)
	}
}

func TestPipelineOptions_ChunkerEnablesChunkingPath(t *testing.T) {
	// @sk-test arch-quality-pass#T3.3: migrate to draftrag.PipelineOptions (AC-004)
	store := vectorstore.NewInMemoryStore()
	emb := &countingEmbedder{}

	ch := fixedChunker{
		chunks: []domain.Chunk{
			{ID: "doc-1:0", ParentID: "doc-1", Position: 0, Content: "a"},
			{ID: "doc-1:1", ParentID: "doc-1", Position: 1, Content: "b"},
		},
	}

	p, err := NewPipelineWithConfig(store, testLLM{}, emb, PipelineOptions{Chunker: ch})
	if err != nil {
		t.Fatal(err)
	}

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
