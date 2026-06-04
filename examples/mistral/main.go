// Использует публичный API draftrag напрямую. Shared для mock/print.
// Быстрый старт:
//
//	LLM_PROVIDER=mock go run .
//	LLM_PROVIDER=mistral MISTRAL_API_KEY=xxx go run .
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
		ID: "doc-1", Content: "Mistral AI — французская компания, основанная в 2023 году бывшими исследователями DeepMind и Meta. Разрабатывает LLM с открытым весом: Mistral 7B, Mixtral 8x7B, Mistral Large.",
		Metadata: map[string]string{"source": "about"},
	},
	{
		ID: "doc-2", Content: "Mistral Large — флагманская модель Mistral, доступная через API. Поддерживает 128k контекст, мультиязычность, function calling. Конкурирует с GPT-4 и Claude 3.",
		Metadata: map[string]string{"source": "models"},
	},
}

// @sk-task llm-providers-mistral-deepseek#T3.1: Mistral example (AC-007)
func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	apiKey := envOr("MISTRAL_API_KEY", "")

	llm := buildLLM(provider, apiKey)
	embedder := shared.NewMockEmbedder(1536)

	store := draftrag.NewInMemoryStore()
	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 3,
		Chunker:     draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{ChunkSize: 500, Overlap: 50}),
	})

	shared.PrintInfo("индексируем %d документов", len(documents))
	if err := pipeline.Index(ctx, documents); err != nil {
		shared.PrintError("индексация: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("готово")

	fmt.Println("\nRAG-чат с Mistral. Введите вопрос (Ctrl+C для выхода):")
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
	case "mistral":
		return draftrag.NewMistralLLM(draftrag.MistralLLMOptions{
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
