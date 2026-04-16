package application

import (
	"context"
	"testing"
)

type streamingLLM struct {
	mockLLMProvider
}

func (m *streamingLLM) GenerateStream(_ context.Context, _, _ string) (<-chan string, error) {
	ch := make(chan string, 3)
	ch <- "hello"
	ch <- " world"
	ch <- "!"
	close(ch)
	return ch, nil
}

func TestPipeline_AnswerStream_NotSupported(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{} // не реализует StreamingLLMProvider
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerStream(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for unsupported streaming, got nil")
	}
}

func TestPipeline_AnswerStream_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, err := p.AnswerStream(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerStream_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, err := p.AnswerStream(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Собираем токены из канала
	tokens := []string{}
	for token := range stream {
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		t.Error("expected at least one token")
	}
}

func TestPipeline_AnswerStreamWithSources_NotSupported(t *testing.T) {
	store := &mockVectorStore{}
	llm := &mockLLMProvider{} // не реализует StreamingLLMProvider
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerStreamWithSources(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for unsupported streaming, got nil")
	}
}

func TestPipeline_AnswerStreamWithSources_EmbedError(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &errorEmbedder{}

	p := NewPipeline(store, llm, embedder)

	_, _, err := p.AnswerStreamWithSources(context.Background(), "test query", 5)
	if err == nil {
		t.Fatal("expected error for embed failure, got nil")
	}
}

func TestPipeline_AnswerStreamWithSources_Success(t *testing.T) {
	store := &mockVectorStore{}
	llm := &streamingLLM{}
	embedder := &mockEmbedder{}

	p := NewPipeline(store, llm, embedder)

	stream, result, err := p.AnswerStreamWithSources(context.Background(), "test query", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Собираем токены из канала
	tokens := []string{}
	for token := range stream {
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		t.Error("expected at least one token")
	}
	_ = result
}
