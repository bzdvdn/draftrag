// @sk-task docs-and-examples#T2.3: qdrant example — RAG-чат с Qdrant (AC-002).
// Использует публичный API draftrag напрямую. Shared только для mock/print.
//
// Быстрый старт с Docker:
//
//	docker compose up -d
//	cp .env.example .env && go run .
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// @sk-task docs-and-examples#T2.3: демо-документы — туристические направления.
var documents = []draftrag.Document{
	{
		ID: "destination-bali", Content: "Бали — остров богов в Индонезии. Лучшее время: апрель-октябрь. Главные достопримечательности: рисовые террасы Тегалаланг, храм Тананах Лот, вулкан Батур. Денпасар — столица и транспортный хаб.",
		Metadata: map[string]string{"region": "asia", "country": "indonesia"},
	},
	{
		ID: "destination-iceland", Content: "Исландия — страна льда и огня. Северное сияние с сентября по март. Золотое кольцо: гейзер Гейсир, водопад Гюдльфосс, разломы Тингвеллир. Голубая лагуна — геотермальный спа.",
		Metadata: map[string]string{"region": "europe", "country": "iceland"},
	},
	{
		ID: "destination-japan", Content: "Япония сочетает древние традиции и современные технологии. Токио — крупнейший мегаполис. Сакура в марте-апреле. Киото — древняя столица с храмами. Фудзи — символ страны.",
		Metadata: map[string]string{"region": "asia", "country": "japan"},
	},
	{
		ID: "destination-peru", Content: "Перу — страна инков. Мачу-Пикчу на высоте 2430 м. Тропа инков — 4-дневный треккинг. Куско — историческая столица инков. Озеро Титикака — высокогорное судоходное озеро.",
		Metadata: map[string]string{"region": "americas", "country": "peru"},
	},
	{
		ID: "destination-morocco", Content: "Марокко — ворота Африки. Медина Марракеша, площадь Джемаа-эль-Фна. Голубой город Шефшауэн. Пустыня Сахара. Фес — древнейший медресе. Кухня: тажин, кускус.",
		Metadata: map[string]string{"region": "africa", "country": "morocco"},
	},
	{
		ID: "destination-norway", Content: "Норвегия — страна фьордов. Гейрангерфьорд — UNESCO. Берген — ворота в фьорды. Тролльтунга на высоте 1100 м. Лофотенские острова. Осло — столица.",
		Metadata: map[string]string{"region": "europe", "country": "norway"},
	},
}

func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	dim := envIntOr("EMBEDDING_DIM", 1536)
	qdrantURL := envOr("QDRANT_URL", "http://localhost:6333")
	collection := envOr("COLLECTION_NAME", "draftrag_chunks")

	llm, embedder := buildComponents(provider, dim)
	if embedder == nil {
		shared.PrintError("error: %s не предоставляет embedder; используйте ollama/openai для эмбеддингов или LLM_PROVIDER=mock", provider)
		os.Exit(1)
	}

	qdrantOpts := draftrag.QdrantOptions{
		URL: qdrantURL, Collection: collection, Dimension: dim,
	}

	shared.PrintInfo("подключаемся к Qdrant: %s", qdrantURL)

	exists, err := draftrag.CollectionExists(ctx, qdrantOpts)
	if err != nil {
		shared.PrintError("ошибка проверки коллекции: %v", err)
		os.Exit(1)
	}
	if !exists {
		shared.PrintInfo("создаём коллекцию %q (dim=%d)", collection, dim)
		if err := draftrag.CreateCollection(ctx, qdrantOpts); err != nil {
			shared.PrintError("ошибка создания коллекции: %v", err)
			os.Exit(1)
		}
		shared.PrintInfo("коллекция создана")
	} else {
		shared.PrintInfo("коллекция %q уже существует", collection)
	}

	store, err := draftrag.NewQdrantStore(qdrantOpts)
	if err != nil {
		shared.PrintError("ошибка создания store: %v", err)
		os.Exit(1)
	}

	pipeline, err := draftrag.NewPipelineWithChunker(store, llm, embedder, draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
		ChunkSize: 1000,
		Overlap:   100,
	}))
	if err != nil {
		shared.PrintError("pipeline creation: %v", err)
		os.Exit(1)
	}

	shared.PrintInfo("индексируем %d документов", len(documents))
	if err := pipeline.Index(ctx, documents); err != nil {
		if errors.Is(err, draftrag.ErrEmptyDocument) {
			shared.PrintError("ошибка: пустой документ в наборе")
			os.Exit(1)
		}
		shared.PrintError("ошибка индексации: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("индексация завершена")

	fmt.Println("\nRAG-чат с Qdrant готов. Введите вопрос (Ctrl+C для выхода):")
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
			if errors.Is(err, draftrag.ErrFiltersNotSupported) {
				shared.PrintError("фильтры не поддерживаются этим store")
				continue
			}
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

func buildComponents(provider string, dim int) (draftrag.LLMProvider, draftrag.Embedder) {
	switch provider {
	case "mock":
		return shared.NewMockLLM(), shared.NewMockEmbedder(dim)
	case "ollama":
		host := envOr("OLLAMA_HOST", "http://localhost:11434")
		return draftrag.NewOllamaLLM(draftrag.OllamaLLMOptions{
			BaseURL: host, Model: envOr("OLLAMA_LLM_MODEL", "llama3.2"),
		}), draftrag.NewOllamaEmbedder(draftrag.OllamaEmbedderOptions{
			BaseURL: host, Model: envOr("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
		})
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			shared.PrintError("error: required env var OPENAI_API_KEY not set; set LLM_PROVIDER=mock to run without API key")
			os.Exit(1)
		}
		return draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
			APIKey: key, BaseURL: envOr("OPENAI_BASE_URL", "https://api.openai.com"),
			Model: envOr("OPENAI_LLM_MODEL", "gpt-4o-mini"),
		}), draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
			APIKey: key, BaseURL: envOr("OPENAI_BASE_URL", "https://api.openai.com"),
			Model: envOr("OPENAI_EMBED_MODEL", "text-embedding-3-small"),
		})
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			shared.PrintError("error: required env var ANTHROPIC_API_KEY not set; set LLM_PROVIDER=mock to run without API key")
			os.Exit(1)
		}
		return draftrag.NewAnthropicLLM(draftrag.AnthropicLLMOptions{
			APIKey: key, Model: envOr("ANTHROPIC_LLM_MODEL", "claude-3-5-sonnet-latest"),
		}), nil
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
