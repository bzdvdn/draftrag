package draftrag_test

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

type mockLLM struct{}

func (m *mockLLM) Health(_ context.Context) error { return nil }
func (m *mockLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "The capital of France is Paris.", nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) Health(_ context.Context) error { return nil }
func (m *mockEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}

func Example() {
	ctx := context.Background()
	store := draftrag.NewInMemoryStore()
	llm := &mockLLM{}
	embedder := &mockEmbedder{}

	pipeline, err := draftrag.NewPipeline(store, llm, embedder)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer pipeline.Close()

	pipeline.Index(ctx, []draftrag.Document{
		{ID: "paris", Content: "Paris is the capital of France."},
	})

	answer, err := pipeline.Answer(ctx, "What is the capital of France?")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(answer)
	// Output: The capital of France is Paris.
}

func ExamplePipeline_Close() {
	ctx := context.Background()
	store := draftrag.NewInMemoryStore()
	llm := &mockLLM{}
	embedder := &mockEmbedder{}

	pipeline, _ := draftrag.NewPipeline(store, llm, embedder)
	pipeline.Index(ctx, []draftrag.Document{{ID: "doc1", Content: "test"}})

	if err := pipeline.Close(); err != nil {
		fmt.Println("Close error:", err)
		return
	}
	fmt.Println("pipeline closed")
	// Output: pipeline closed
}
