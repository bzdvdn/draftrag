// Использует публичный API draftrag напрямую. Shared для mock/print.
// Быстрый старт:
//
//	LLM_PROVIDER=mock go run .
//	LLM_PROVIDER=deepseek DEEPSEEK_API_KEY=xxx go run .
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

var documents = []draftrag.Document{
	{
		ID: "doc-1", Content: "DeepSeek — китайская AI-компания, разрабатывающая LLM. Известна моделями DeepSeek-V3 и DeepSeek-R1, которые показывают конкурентоспособные результаты в бенчмарках.",
		Metadata: map[string]string{"source": "about"},
	},
	{
		ID: "doc-2", Content: "DeepSeek-R1 — модель с улучшенными способностями к рассуждению (reasoning). Использует технику chain-of-thought и достигает результатов, сопоставимых с OpenAI o1, при существенно меньшей стоимости.",
		Metadata: map[string]string{"source": "models"},
	},
}

// @sk-task llm-providers-mistral-deepseek#T3.2: DeepSeek example (AC-007)
func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	apiKey := envOr("DEEPSEEK_API_KEY", "")

	llm := buildLLM(provider, apiKey)
	embedder := shared.NewMockEmbedder(1536)

	store := draftrag.NewInMemoryStore()
	pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 3,
		Chunker:     draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{ChunkSize: 500, Overlap: 50}),
	})
	if err != nil {
		shared.PrintError("pipeline creation: %v", err)
		os.Exit(1)
	}

	shared.PrintInfo("индексируем %d документов", len(documents))
	if err := pipeline.Index(ctx, documents); err != nil {
		shared.PrintError("индексация: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("готово")

	fmt.Println("\nRAG-чат с DeepSeek. Введите вопрос (Ctrl+C для выхода):")
	fmt.Println(strings.Repeat("─", 60))

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		question := strings.TrimSpace(scanner.Text())
		if question == "" {
			continue
		}

		answer, sources, err := pipeline.Search(question).TopK(3).Cite(ctx)
		if err != nil {
			shared.PrintError("ошибка: %v", err)
			continue
		}

		fmt.Printf("\n%s\n", answer)
		if len(sources.Chunks) > 0 {
			fmt.Println("\nИсточники:")
			for i, r := range sources.Chunks {
				fmt.Printf("  [%d] %s (score=%.3f)\n", i+1, r.Chunk.ParentID, r.Score)
			}
		}
		fmt.Println(strings.Repeat("─", 60))
	}
}

func buildLLM(provider, apiKey string) draftrag.LLMProvider {
	switch provider {
	case "mock":
		return shared.NewMockLLM()
	case "deepseek":
		return draftrag.NewDeepSeekLLM(draftrag.DeepSeekLLMOptions{
			APIKey: apiKey,
		})
	default:
		shared.PrintError("unknown LLM_PROVIDER=%q", provider)
		os.Exit(1)
		return nil
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
