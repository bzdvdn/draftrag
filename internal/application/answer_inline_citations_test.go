// Package application tests the application-layer pipeline behaviors.
package application

import (
	"context"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

type assertLLMInline struct {
	t            *testing.T
	wantContains []string
	answer       string
}

func (l assertLLMInline) Generate(_ context.Context, _, userMessage string) (string, error) {
	l.t.Helper()

	for _, needle := range l.wantContains {
		if !strings.Contains(userMessage, needle) {
			l.t.Fatalf("expected userMessage to contain %q, got:\n%s", needle, userMessage)
		}
	}

	return l.answer, nil
}

func TestPipeline_AnswerWithInlineCitations_ReturnsCitationsMapping(t *testing.T) {
	expected := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A"}, Score: 0.9},
			{Chunk: domain.Chunk{Content: "B"}, Score: 0.8},
		},
		TotalFound: 2,
	}

	p := NewPipelineWithConfig(
		fixedSearchStore2{result: expected},
		assertLLMInline{
			t: t,
			wantContains: []string{
				"Инструкция:",
				"Источники:",
				"[1] A",
				"[2] B",
				"Вопрос:\nQ",
			},
			answer: "ok [1]",
		},
		fixedEmbedder2{},
		PipelineConfig{},
	)

	answer, gotRetrieval, citations, err := p.AnswerWithInlineCitations(context.Background(), "Q", 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if answer != "ok [1]" {
		t.Fatalf("expected answer %q, got %q", "ok [1]", answer)
	}
	if gotRetrieval.QueryText != "Q" || gotRetrieval.TotalFound != 2 || len(gotRetrieval.Chunks) != 2 {
		t.Fatalf("unexpected retrieval: %#v", gotRetrieval)
	}

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %#v", citations)
	}
	if citations[0].Number != 1 || citations[0].Chunk.Chunk.Content != "A" {
		t.Fatalf("unexpected citations[0]: %#v", citations[0])
	}
	if citations[1].Number != 2 || citations[1].Chunk.Chunk.Content != "B" {
		t.Fatalf("unexpected citations[1]: %#v", citations[1])
	}
}

func TestPipeline_AnswerWithInlineCitations_RespectsMaxContextChunks(t *testing.T) {
	expected := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{Chunk: domain.Chunk{Content: "A"}, Score: 0.9},
			{Chunk: domain.Chunk{Content: "B"}, Score: 0.8},
		},
		TotalFound: 2,
	}

	p := NewPipelineWithConfig(
		fixedSearchStore2{result: expected},
		assertLLMInline{
			t: t,
			wantContains: []string{
				"[1] A",
			},
			answer: "ok [1]",
		},
		fixedEmbedder2{},
		PipelineConfig{MaxContextChunks: 1},
	)

	_, _, citations, err := p.AnswerWithInlineCitations(context.Background(), "Q", 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(citations) != 1 || citations[0].Number != 1 {
		t.Fatalf("unexpected citations: %#v", citations)
	}
}
