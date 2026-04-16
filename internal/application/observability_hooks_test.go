package application

import (
	"context"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type recordHooks struct {
	events []string
	ends   []domain.StageEndEvent
}

func (h *recordHooks) StageStart(_ context.Context, ev domain.StageStartEvent) {
	h.events = append(h.events, "start:"+ev.Operation+":"+string(ev.Stage))
}

func (h *recordHooks) StageEnd(_ context.Context, ev domain.StageEndEvent) {
	h.events = append(h.events, "end:"+ev.Operation+":"+string(ev.Stage))
	h.ends = append(h.ends, ev)
}

type okStoreForIndex struct{}

func (okStoreForIndex) Upsert(_ context.Context, _ domain.Chunk) error { return nil }
func (okStoreForIndex) Delete(_ context.Context, _ string) error       { return nil }
func (okStoreForIndex) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

type oneChunkChunker struct{}

func (oneChunkChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return []domain.Chunk{
		{
			ID:       doc.ID + "#0",
			Content:  doc.Content,
			ParentID: doc.ID,
			Position: 0,
		},
	}, nil
}

func TestPipeline_Hooks_AnswerStages_Order(t *testing.T) {
	expected := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A", ParentID: "p1"}, Score: 0.9},
		},
		TotalFound: 1,
	}

	hooks := &recordHooks{}

	p := NewPipelineWithConfig(
		fixedSearchStore2{result: expected},
		okLLM2{},
		fixedEmbedder2{},
		PipelineConfig{Hooks: hooks},
	)

	_, err := p.Answer(context.Background(), "Q", 3)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []string{
		"start:Answer:embed",
		"end:Answer:embed",
		"start:Answer:search",
		"end:Answer:search",
		"start:Answer:generate",
		"end:Answer:generate",
	}
	if len(hooks.events) != len(want) {
		t.Fatalf("unexpected events: %#v", hooks.events)
	}
	for i := range want {
		if hooks.events[i] != want[i] {
			t.Fatalf("unexpected events[%d]: want %q, got %q (all=%#v)", i, want[i], hooks.events[i], hooks.events)
		}
	}

	for _, ev := range hooks.ends {
		if ev.Duration < 0 {
			t.Fatalf("unexpected negative duration: %#v", ev)
		}
	}
}

func TestPipeline_Hooks_IndexChunkingStage_CalledWhenChunkerEnabled(t *testing.T) {
	hooks := &recordHooks{}

	p := NewPipelineWithConfig(
		okStoreForIndex{},
		okLLM2{},
		fixedEmbedder2{},
		PipelineConfig{
			Chunker: oneChunkChunker{},
			Hooks:   hooks,
		},
	)

	err := p.Index(context.Background(), []domain.Document{
		{ID: "doc1", Content: "Hello"},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Важно только, что chunking присутствует и что embed тоже был вызван.
	// Duration проверяем косвенно: hookEnd выставляет duration.
	if len(hooks.events) < 4 {
		t.Fatalf("unexpected events: %#v", hooks.events)
	}
	if hooks.events[0] != "start:Index:chunking" || hooks.events[1] != "end:Index:chunking" {
		t.Fatalf("unexpected chunking events: %#v", hooks.events)
	}
	if hooks.events[2] != "start:Index:embed" || hooks.events[3] != "end:Index:embed" {
		t.Fatalf("unexpected embed events: %#v", hooks.events)
	}

	for _, ev := range hooks.ends {
		// Длительность может быть 0 на очень быстрых машинах, но не должна быть отрицательной.
		if ev.Duration < 0*time.Nanosecond {
			t.Fatalf("unexpected duration: %#v", ev)
		}
	}
}
