// @sk-task prod-issues#T3.4: Пример sub-query decomposition (AC-011)
//
// Демонстрирует разбиение сложного запроса на под-вопросы для улучшения recall.
// Использует LLM-rewriter для генерации под-вопросов на основе QueryDecomposer.
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
		ID:      "go-goroutines",
		Content: "Горутины — лёгковесные потоки выполнения в Go, управляемые рантаймом. Создаются ключевым словом go перед вызовом функции. Горутины мультиплексируются на операционных потоках.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID:      "go-channels",
		Content: "Каналы — основной механизм синхронизации и передачи данных между горутинами в Go. Небуферизованный канал блокирует отправителя до готовности получателя. Буферизованный канал блокирует только при заполнении буфера.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID:      "go-context",
		Content: "Пакет context предоставляет механизм передачи сроков, сигналов отмены и значений через границы API. context.WithCancel создаёт отменяемый контекст. context.WithTimeout добавляет ограничение по времени.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID:      "go-sync",
		Content: "Пакет sync предоставляет примитивы синхронизации: Mutex для взаимного исключения, WaitGroup для ожидания группы горутин, Once для однократного выполнения.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID:      "go-errors",
		Content: "Обработка ошибок в Go строится на возвращаемых значениях типа error. errors.Is проверяет совпадение в цепочке обёрток. errors.As извлекает конкретный тип.",
		Metadata: map[string]string{"topic": "errors"},
	},
}

func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	dim := envIntOr("EMBEDDING_DIM", 1536)

	llm, embedder := buildComponents(provider, dim)
	if embedder == nil {
		shared.PrintError("error: %s не предоставляет embedder; используйте LLM_PROVIDER=mock", provider)
		os.Exit(1)
	}

	store := draftrag.NewInMemoryStore()

	// Создаём QueryDecomposer на основе LLM
	decomp, err := draftrag.NewLLMQueryDecomposer(llm,
		"Разбей следующий вопрос на 2-3 независимых под-вопроса. "+
			"Каждый под-вопрос должен быть самодостаточным. "+
			"Ответь в формате: каждый под-вопрос на отдельной строке.")
	if err != nil {
		shared.PrintError("создание decomposer: %v", err)
		os.Exit(1)
	}

	pipeline, err := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		Chunker:         draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{ChunkSize: 500, Overlap: 50}),
		QueryDecomposer: decomp,
	})
	if err != nil {
		shared.PrintError("создание pipeline: %v", err)
		os.Exit(1)
	}

	shared.PrintInfo("индексируем %d документов", len(documents))
	if err := pipeline.Index(ctx, documents); err != nil {
		shared.PrintError("индексация: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("индексация завершена")

	// Пример без декомпозиции
	question := "Какие механизмы синхронизации и конкурентности есть в Go?"
	shared.PrintInfo("вопрос: %s", question)

	shared.PrintInfo("--- без декомпозиции (single-query) ---")
	answer1, sources1, err := pipeline.Search(question).TopK(3).Cite(ctx)
	if err != nil {
		shared.PrintError("поиск без декомпозиции: %v", err)
	} else {
		shared.PrintAnswer(question, answer1, []draftrag.RetrievalResult{sources1})
	}

	// Пример с декомпозицией запроса
	shared.PrintInfo("--- с декомпозицией (SubDecompose) ---")
	answer2, sources2, err := pipeline.Search(question).SubDecompose().TopK(3).Cite(ctx)
	if err != nil {
		shared.PrintError("поиск с декомпозицией: %v", err)
	} else {
		shared.PrintAnswer(question, answer2, []draftrag.RetrievalResult{sources2})
	}
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
