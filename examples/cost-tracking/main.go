// @sk-task cost-tracking: T2.2 — пример CostTracker с mock LLM.
//
// Быстрый старт:
//
//	cd examples/cost-tracking && go run .
package main

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// mockCostLLM — mock LLM, реализующий UsageAwareLLMProvider с фиксированным token usage.
type mockCostLLM struct{}

func (m *mockCostLLM) Health(_ context.Context) error { return nil }

func (m *mockCostLLM) Generate(_ context.Context, _, userMessage string) (string, error) {
	return fmt.Sprintf("[mock] %s", userMessage), nil
}

func (m *mockCostLLM) GenerateWithUsage(_ context.Context, _, userMessage string) (string, draftrag.TokenUsage, error) {
	return fmt.Sprintf("[mock] %s", userMessage), draftrag.TokenUsage{
		PromptTokens:     50,
		CompletionTokens: 30,
		TotalTokens:      80,
	}, nil
}

func (m *mockCostLLM) ModelName() string { return "mock-model" }

func main() {
	pricing := map[string]draftrag.ModelPricing{
		"mock-model": {InputCostPer1K: 0.01, OutputCostPer1K: 0.03},
	}

	ct := draftrag.NewCostTracker(&mockCostLLM{}, pricing)

	answer, err := ct.Generate(context.Background(), "You are a helpful assistant.", "What is Go?")
	if err != nil {
		panic(err)
	}
	fmt.Println("Answer:", answer)

	snap := ct.Snapshot()
	fmt.Printf("\n=== Cost Snapshot ===\n")
	fmt.Printf("Calls:           %d\n", snap.CallsCount)
	fmt.Printf("Prompt tokens:   %d\n", snap.PromptTokens)
	fmt.Printf("Completion tokens: %d\n", snap.CompletionTokens)
	fmt.Printf("Total tokens:    %d\n", snap.TotalTokens)
	fmt.Printf("Total cost:      $%.6f\n", snap.TotalCost)

	cp1 := ct.Checkpoint()
	_, _ = ct.Generate(context.Background(), "You are a helpful assistant.", "What is Rust?")
	cp2 := ct.Checkpoint()
	diff := draftrag.Diff(cp1, cp2)
	fmt.Printf("\n=== Last call delta ===\n")
	fmt.Printf("Calls:           %d\n", diff.CallsCount)
	fmt.Printf("Total cost:      $%.6f\n", diff.TotalCost)

	ct.Reset()
	snap = ct.Snapshot()
	fmt.Printf("\n=== After Reset ===\n")
	fmt.Printf("Calls: %d (expected 0)\n", snap.CallsCount)
}
