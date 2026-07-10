// @sk-task pii-guardrails#T2.5: пример PII-guardrails с InMemoryStore (AC-001, AC-002).
//
// Быстрый старт:
//
//	cd examples/pii-guardrails && cp ../memory/.env.example .env && go run .
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

	detector := draftrag.NewDefaultPIIDetector(draftrag.PIICategories{
		Email: true,
		Phone: true,
		SSN:   true,
	})

	p, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		PIIDetector: detector,
	})
	if err != nil {
		fmt.Println("Error creating pipeline:", err)
		os.Exit(1)
	}

	docs := []draftrag.Document{
		{
			ID:      "doc1",
			Content: "Клиент Иванов: email ivanov@example.com, телефон +7-900-123-45-67, паспорт 123-45-6789.",
		},
		{
			ID:      "doc2",
			Content: "Сотрудник Петров: email petrov@test.org, SSN 987-65-4321.",
		},
	}

	if err := p.Index(ctx, docs); err != nil {
		fmt.Println("Error indexing:", err)
		os.Exit(1)
	}

	// Проверка PII-redaction через Query
	result, err := p.Query(ctx, "контакты клиентов")
	if err != nil {
		fmt.Println("Error querying:", err)
		os.Exit(1)
	}

	fmt.Println("=== Query Results (PII should be redacted) ===")
	for _, ch := range result.Chunks {
		fmt.Printf("Content: %s\n", ch.Chunk.Content)
		fmt.Println("---")
	}
}
