// @sk-task middleware-chain#T2.6: пример middleware-цепочки с built-in LoggingMiddleware
// и PIIDetectorMiddleware (AC-001, AC-004).
//
// Быстрый старт:
//
//	cd examples/middleware && go run .
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
	ctx := context.Background()

	llm := shared.NewMockLLM()
	embedder := shared.NewMockEmbedder(3)
	store := draftrag.NewInMemoryStore()

	// Используем встроенные middleware: логирование + PII redaction
	piiDetector := draftrag.NewDefaultPIIDetector(draftrag.PIICategories{})

	p, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		Middleware: []draftrag.Middleware{
			draftrag.NewLoggingMiddleware(),
			draftrag.NewPIIDetectorMiddleware(piiDetector),
		},
	})
	if err != nil {
		fmt.Println("Error creating pipeline:", err)
		os.Exit(1)
	}

	docs := []draftrag.Document{
		{ID: "doc1", Content: "Привет, это тестовый документ. Email: test@example.com"},
	}
	if err := p.Index(ctx, docs); err != nil {
		fmt.Println("Error indexing:", err)
		os.Exit(1)
	}
	fmt.Println("--- Index done ---")

	result, err := p.Query(ctx, "тестовый запрос")
	if err != nil {
		fmt.Println("Error querying:", err)
		os.Exit(1)
	}
	fmt.Printf("Query results: %d chunks\n", len(result.Chunks))

	answer, err := p.Answer(ctx, "тестовый запрос")
	if err != nil {
		fmt.Println("Error answering:", err)
		os.Exit(1)
	}
	fmt.Printf("Answer: %s\n", answer)

	if err := p.Close(); err != nil {
		fmt.Println("Error closing pipeline:", err)
	}
}
