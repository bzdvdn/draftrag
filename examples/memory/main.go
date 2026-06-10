// @sk-task docs-and-examples#T2.1: memory example — in-memory RAG на Go-документации (AC-006).
// Без Docker, использует публичный API draftrag + shared/mock.
//
// Быстрый старт:
//
//	cd examples/memory && cp .env.example .env && go run .
package main

import (
	"context"
	"os"
	"strconv"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// @sk-task docs-and-examples#T2.1: 10 demo-документов про Go.
var documents = []draftrag.Document{
	{
		ID: "go-goroutines", Content: "Горутины — лёгковесные потоки выполнения в Go, управляемые рантаймом. Создаются ключевым словом go перед вызовом функции. Горутины мультиплексируются на операционных потоках и потребляют минимум ресурсов (начальный стек ~2 КБ). Рантаймовый планировщик использует модель M:N, позволяя запускать миллионы горутин.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-channels", Content: "Каналы — основной механизм синхронизации и передачи данных между горутинами в Go. Небуферизованный канал блокирует отправителя до готовности получателя. Буферизованный канал блокирует только при заполнении буфера. Оператор <- используется для отправки и получения.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-context", Content: "Пакет context предоставляет механизм передачи сроков, сигналов отмены и значений через границы API. context.Background() — корневой контекст. context.WithCancel создаёт отменяемый контекст. context.WithDeadline и context.WithTimeout добавляют ограничение по времени.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-sync", Content: "Пакет sync предоставляет примитивы синхронизации: Mutex и RWMutex для взаимного исключения, WaitGroup для ожидания группы горутин, Once для однократного выполнения, Pool для переиспользования объектов, Map для конкурентного доступа.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-errors", Content: "Обработка ошибок в Go строится на возвращаемых значениях типа error. errors.Is проверяет совпадение в цепочке обёрток, errors.As извлекает конкретный тип. Sentinel-ошибки удобны для сравнения через errors.Is.",
		Metadata: map[string]string{"topic": "errors"},
	},
	{
		ID: "go-interfaces", Content: "Интерфейсы в Go определяют поведение через набор методов. Реализация неявная — любой тип, имеющий все методы, автоматически реализует интерфейс. Пустой интерфейс any принимает любое значение.",
		Metadata: map[string]string{"topic": "types"},
	},
	{
		ID: "go-defer", Content: "Отложенные вызовы (defer) гарантируют выполнение функции перед возвратом. Defer выполняется в порядке LIFO. Типичное применение: закрытие файлов, разблокировка мьютексов. Panic прерывает нормальный поток, recover восстанавливает контроль.",
		Metadata: map[string]string{"topic": "control-flow"},
	},
	{
		ID: "go-structs", Content: "Структуры — составные типы данных. Методы определяются через получатель (receiver). Value receiver копирует структуру, pointer receiver позволяет изменять оригинал. Встраивание (embedding) обеспечивает композицию без наследования.",
		Metadata: map[string]string{"topic": "types"},
	},
	{
		ID: "go-slices", Content: "Слайсы — динамические массивы. append увеличивает слайс, перевыделяя память при превышении ёмкости. Мапы — хеш-таблицы для быстрого поиска по ключу. Чтение из nil-мапы не паникует, запись — паникует.",
		Metadata: map[string]string{"topic": "data-structures"},
	},
	{
		ID: "go-select", Content: "Оператор select позволяет горутине ждать нескольких операций с каналами. Выполняется первый готовый case; при нескольких готовых выбирается случайный. Блок default делает select неблокирующим.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
}

func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	dim := envIntOr("EMBEDDING_DIM", 1536)

	llm, embedder := buildComponents(provider, dim)
	if embedder == nil {
		shared.PrintError("error: %s не предоставляет embedder; используйте ollama/openai для эмбеддингов или LLM_PROVIDER=mock", provider)
		os.Exit(1)
	}

	store := draftrag.NewInMemoryStore()

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
		shared.PrintError("index: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("индексация завершена")

	question := "Что такое goroutine?"
	shared.PrintInfo("вопрос: %s", question)

	answer, sources, err := pipeline.Search(question).TopK(3).Cite(ctx)
	if err != nil {
		shared.PrintError("search: %v", err)
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
	case "anthropic":
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			shared.PrintError("error: required env var ANTHROPIC_API_KEY not set; set LLM_PROVIDER=mock to run without API key")
			os.Exit(1)
		}
		llm := draftrag.NewAnthropicLLM(draftrag.AnthropicLLMOptions{
			APIKey: key,
			Model:  envOr("ANTHROPIC_LLM_MODEL", "claude-3-5-sonnet-latest"),
		})
		return llm, nil
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
