// @sk-task docs-and-examples#T3.3: milvus example — RAG-чат с Milvus (AC-005).
// Использует публичный API draftrag напрямую. Shared только для mock/print.
// MilvusStore — внутренний API (internal/infrastructure/vectorstore),
// документирован как "API в разработке".
//
// Быстрый старт с Docker:
//
//	docker compose up -d
//	cp .env.example .env && go run .
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// @sk-task docs-and-examples#T3.3: демо-документы — продукт SmartHome Hub.
var documents = []draftrag.Document{
	{
		ID: "smarthome-overview", Content: "SmartHome Hub — центральный контроллер умного дома. Поддерживает Zigbee, Z-Wave, Wi-Fi и Matter. Управление через мобильное приложение. Локальная обработка без интернета.",
		Metadata: map[string]string{"category": "overview"},
	},
	{
		ID: "smarthome-setup", Content: "Настройка SmartHome Hub за 5–10 минут. Подключите к роутеру через Ethernet или Wi-Fi, установите приложение, создайте аккаунт, отсканируйте QR-код.",
		Metadata: map[string]string{"category": "setup"},
	},
	{
		ID: "smarthome-zigbee", Content: "Добавление Zigbee-устройства: «Добавить устройство» → «Zigbee», переведите устройство в режим сопряжения. Хаб обнаружит за 30 секунд. До 200 устройств.",
		Metadata: map[string]string{"category": "zigbee"},
	},
	{
		ID: "smarthome-automations", Content: "Автоматизации: триггеры по времени, восходу/закату, состоянию устройства, геолокации. Действия: управление устройствами, push-уведомления, запуск сцен. Работают локально.",
		Metadata: map[string]string{"category": "automations"},
	},
	{
		ID: "smarthome-security", Content: "Модуль безопасности: датчики открытия, движения PIR, сирены, умные замки. Режимы охраны: «Дома», «Ушёл», «Ночь». Уведомления в приложение.",
		Metadata: map[string]string{"category": "security"},
	},
	{
		ID: "smarthome-voice", Content: "Интеграция с Алисой (Яндекс), Google Home и Amazon Alexa. Настройка: приложение → «Голосовые ассистенты» → авторизация. Команды: включение, яркость, температура.",
		Metadata: map[string]string{"category": "voice"},
	},
}

func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	dim := envIntOr("EMBEDDING_DIM", 1536)
	milvusAddr := envOr("MILVUS_ADDR", "localhost:19530")
	collection := envOr("COLLECTION_NAME", "draftrag_chunks")

	llm, embedder := buildComponents(provider, dim)
	if embedder == nil {
		shared.PrintError("error: %s не предоставляет embedder; используйте ollama/openai для эмбеддингов или LLM_PROVIDER=mock", provider)
		os.Exit(1)
	}

	milvusBase := milvusAddr
	if !strings.HasPrefix(milvusBase, "http://") && !strings.HasPrefix(milvusBase, "https://") {
		milvusBase = "http://" + milvusBase
	}

	shared.PrintInfo("подключаемся к Milvus: %s", milvusBase)

	shared.PrintInfo("создаём коллекцию %q (dim=%d)", collection, dim)
	if err := vectorstore.CreateMilvusCollection(ctx, milvusBase, collection, "", dim); err != nil {
		shared.PrintError("ошибка создания коллекции: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("коллекция готова")

	store := vectorstore.NewMilvusStore(milvusBase, collection, "", dim)
	shared.PrintInfo("коллекция: %s", collection)

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
		shared.PrintError("ошибка индексации: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("индексация завершена")

	fmt.Println("\nRAG-чат с Milvus готов. Введите вопрос (Ctrl+C для выхода):")
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
