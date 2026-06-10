// @sk-task docs-and-examples#T2.2: pgvector example — RAG-чат с PostgreSQL + pgvector (AC-001).
// Использует публичный API draftrag напрямую. Shared только для mock/print.
//
// Быстрый старт с Docker:
//
//	docker compose up -d
//	PGVECTOR_DSN="postgres://draftrag:draftrag@localhost:5432/draftrag?sslmode=disable" \
//	  go run ./examples/pgvector/
package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bzdvdn/draftrag/examples/shared"
	"github.com/bzdvdn/draftrag/pkg/draftrag"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// @sk-task docs-and-examples#T2.2: демо-документы про Go.
var documents = []draftrag.Document{
	{
		ID: "go-goroutines", Content: "Горутины — лёгковесные потоки выполнения в Go, управляемые рантаймом. Создаются ключевым словом go перед вызовом функции. Горутины мультиплексируются на операционных потоках и потребляют минимум ресурсов (начальный стек ~2 КБ).",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-channels", Content: "Каналы — основной механизм синхронизации и передачи данных между горутинами. Небуферизованный канал блокирует отправителя до готовности получателя. Буферизованный канал блокирует только при заполнении буфера.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-context", Content: "Пакет context предоставляет механизм передачи сроков, сигналов отмены и значений через границы API. context.WithCancel создаёт отменяемый контекст. context.WithTimeout добавляет ограничение по времени.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-sync", Content: "Пакет sync: Mutex и RWMutex для взаимного исключения, WaitGroup для ожидания горутин, Once для однократного выполнения, Pool для переиспользования объектов.",
		Metadata: map[string]string{"topic": "concurrency"},
	},
	{
		ID: "go-errors", Content: "Обработка ошибок в Go строится на возвращаемых значениях типа error. errors.Is проверяет совпадение в цепочке обёрток, errors.As извлекает конкретный тип.",
		Metadata: map[string]string{"topic": "errors"},
	},
	{
		ID: "go-interfaces", Content: "Интерфейсы в Go — неявная реализация. Любой тип, имеющий все методы интерфейса, автоматически его реализует. Пустой интерфейс any принимает любое значение.",
		Metadata: map[string]string{"topic": "types"},
	},
	{
		ID: "go-defer", Content: "Отложенные вызовы (defer) гарантируют выполнение функции перед возвратом. Defer выполняется в порядке LIFO. Panic прерывает нормальный поток, recover восстанавливает контроль.",
		Metadata: map[string]string{"topic": "control-flow"},
	},
}

func main() {
	ctx := context.Background()

	provider := envOr("LLM_PROVIDER", "mock")
	dim := envIntOr("EMBEDDING_DIM", 1536)
	dsn := os.Getenv("PGVECTOR_DSN")
	if dsn == "" {
		shared.PrintError("error: required env var PGVECTOR_DSN not set")
		os.Exit(1)
	}

	llm, embedder := buildComponents(provider, dim)
	if embedder == nil {
		shared.PrintError("error: %s не предоставляет embedder; используйте ollama/openai для эмбеддингов или LLM_PROVIDER=mock", provider)
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		shared.PrintError("ошибка открытия БД: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		shared.PrintError("ошибка подключения к БД: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("подключено к PostgreSQL")

	tableName := envOr("TABLE_NAME", "draftrag_chunks")
	shared.PrintInfo("применяем миграции pgvector")
	if err := draftrag.MigratePGVector(ctx, db, draftrag.PGVectorMigrateOptions{
		PGVectorOptions: draftrag.PGVectorOptions{
			TableName:          tableName,
			EmbeddingDimension: dim,
			CreateExtension:    true,
		},
	}); err != nil {
		shared.PrintError("ошибка миграции: %v", err)
		os.Exit(1)
	}
	shared.PrintInfo("схема готова")

	store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: dim,
	})

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

	fmt.Println("\nRAG-чат с pgvector готов. Введите вопрос (Ctrl+C для выхода):")
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
