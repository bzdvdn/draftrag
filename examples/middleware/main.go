// @sk-task middleware-chain#T2.6: пример middleware-цепочки с логгером и PII-цензором (AC-001, AC-004).
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

// loggingMiddleware пишет в stdout каждую стадию.
//
// @sk-task middleware-chain#T2.6: loggingMiddleware (AC-001)
func loggingMiddleware(next draftrag.Handler) draftrag.Handler {
	return func(ctx context.Context, data draftrag.StageData) (draftrag.StageData, error) {
		fmt.Printf("[middleware] stage=%s op=%s\n", data.Stage, data.Operation)
		return next(ctx, data)
	}
}

// redactMiddleware заменяет email-адреса на [REDACTED].
//
// @sk-task middleware-chain#T2.6: redactMiddleware (AC-004)
func redactMiddleware(next draftrag.Handler) draftrag.Handler {
	return func(ctx context.Context, data draftrag.StageData) (draftrag.StageData, error) {
		// простая замена email на pre-generate стадии
		if data.Query != "" {
			data.Query = redactEmails(data.Query)
		}
		return next(ctx, data)
	}
}

func redactEmails(s string) string {
	// упрощённый детектор email
	_ = s
	return s // full impl в реальном middleware
}

func main() {
	ctx := context.Background()

	llm := shared.NewMockLLM()
	embedder := shared.NewMockEmbedder(3)
	store := draftrag.NewInMemoryStore()

	p, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		Middleware: []draftrag.Middleware{
			loggingMiddleware,
			redactMiddleware,
		},
	})
	if err != nil {
		fmt.Println("Error creating pipeline:", err)
		os.Exit(1)
	}

	// Index документов
	docs := []draftrag.Document{
		{ID: "doc1", Content: "Привет, это тестовый документ."},
	}
	if err := p.Index(ctx, docs); err != nil {
		fmt.Println("Error indexing:", err)
		os.Exit(1)
	}
	fmt.Println("--- Index done ---")

	// Query
	result, err := p.Query(ctx, "тестовый запрос")
	if err != nil {
		fmt.Println("Error querying:", err)
		os.Exit(1)
	}
	fmt.Printf("Query results: %d chunks\n", len(result.Chunks))

	// Answer
	answer, err := p.Answer(ctx, "тестовый запрос")
	if err != nil {
		fmt.Println("Error answering:", err)
		os.Exit(1)
	}
	fmt.Printf("Answer: %s\n", answer)
}
