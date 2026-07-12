// @sk-task prod-issues#T3.3: Пример семантического чанкинга (AC-010)
//
// Демонстрирует использование SemanticChunker для интеллектуального
// разбиения документов на чанки на основе семантической близости предложений.
package main

import (
	"context"
	"os"
	"strconv"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

var documents = []draftrag.Document{
	{
		ID:      "go-intro",
		Content: "Go — это компилируемый язык программирования с открытым исходным кодом. Он был создан в Google в 2007 году. Go сочетает производительность C++ с читаемостью Python. Язык особенно популярен для веб-серверов и микросервисов. В Go используется статическая типизация, что повышает надёжность кода.",
	},
	{
		ID:      "go-concurrency",
		Content: "Горутины — лёгковесные потоки выполнения в Go. Они запускаются ключевым словом go. Горутины мультиплексируются на потоках ОС. Каналы обеспечивают безопасную передачу данных между горутинами. Оператор select позволяет ожидать несколько каналов одновременно.",
	},
}

func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	dim := envIntOr("EMBEDDING_DIM", 384)

	_, embedder := buildComponents(provider, dim)
	if embedder == nil {
		shared.PrintError("error: %s не предоставляет embedder; используйте LLM_PROVIDER=mock", provider)
		os.Exit(1)
	}

	chunker, err := draftrag.NewSemanticChunker(draftrag.SemanticChunkerOptions{
		Embedder:            embedder,
		SimilarityThreshold: 0.7,
		MinChunkSize:        20,
		MaxChunkSize:        500,
	})
	if err != nil {
		shared.PrintError("создание semantic chunker: %v", err)
		os.Exit(1)
	}

	store := draftrag.NewInMemoryStore()
	llm := shared.NewMockLLM()

	pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		Chunker: chunker,
	})
	if err != nil {
		shared.PrintError("создание pipeline: %v", err)
		os.Exit(1)
	}

	shared.PrintInfo("индексируем %d документов с семантическим чанкингом", len(documents))
	for _, doc := range documents {
		shared.PrintInfo("документ %q (%d символов)", doc.ID, len(doc.Content))
	}

	if err := pipeline.Index(ctx, documents); err != nil {
		shared.PrintError("индексация: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("индексация завершена")

	question := "Что такое горутины?"
	shared.PrintInfo("вопрос: %s", question)

	answer, sources, err := pipeline.Search(question).TopK(3).Cite(ctx)
	if err != nil {
		shared.PrintError("поиск: %v", err)
		os.Exit(1)
	}

	shared.PrintAnswer(question, answer, []draftrag.RetrievalResult{sources})
}

func buildComponents(provider string, dim int) (draftrag.LLMProvider, draftrag.Embedder) {
	switch provider {
	case "mock":
		return shared.NewMockLLM(), shared.NewMockEmbedder(dim)
	case "ollama":
		host := envOr("OLLAMA_HOST", "http://localhost:11434")
		llm := draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
			BaseURL: host,
			Model:   envOr("OLLAMA_LLM_MODEL", "llama3.2"),
		})
		emb := draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
			BaseURL: host,
			Model:   envOr("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
		})
		return llm, emb
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			shared.PrintError("error: required env var OPENAI_API_KEY not set; set LLM_PROVIDER=mock to run without API key")
			os.Exit(1)
		}
		llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
			APIKey:  key,
			BaseURL: envOr("OPENAI_BASE_URL", "https://api.openai.com"),
			Model:   envOr("OPENAI_LLM_MODEL", "gpt-4o-mini"),
		})
		emb := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
			APIKey:  key,
			BaseURL: envOr("OPENAI_BASE_URL", "https://api.openai.com"),
			Model:   envOr("OPENAI_EMBED_MODEL", "text-embedding-3-small"),
		})
		return llm, emb
	default:
		shared.PrintError("error: unknown LLM_PROVIDER=%q", provider)
		os.Exit(1)
		return nil, nil
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}
