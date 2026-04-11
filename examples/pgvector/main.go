// Пример RAG с pgvector (PostgreSQL) — пример использования draftRAG.
//
// Запускает интерактивный RAG-чат с pgvector как векторным хранилищем.
// Схема БД создаётся автоматически через MigratePGVector при первом запуске.
//
// Быстрый старт с Docker:
//
//	docker compose up -d
//	PGVECTOR_DSN="postgres://draftrag:draftrag@localhost:5432/draftrag?sslmode=disable" \
//	  EMBEDDER_API_KEY=sk-... LLM_API_KEY=sk-... \
//	  go run ./examples/pgvector/
//
// Переменные окружения:
//
//	PGVECTOR_DSN        — DSN для PostgreSQL (обязательно)
//	EMBEDDER_BASE_URL   — базовый URL embedder API (по умолчанию: https://api.openai.com)
//	EMBEDDER_API_KEY    — ключ API для embedder (обязательно)
//	EMBEDDER_MODEL      — модель embeddings (по умолчанию: text-embedding-ada-002)
//	LLM_BASE_URL        — базовый URL LLM API (по умолчанию: https://api.openai.com)
//	LLM_API_KEY         — ключ API для LLM (обязательно)
//	LLM_MODEL           — модель LLM (по умолчанию: gpt-4o-mini)
//	TABLE_NAME          — имя таблицы pgvector (по умолчанию: draftrag_chunks)
//	EMBEDDING_DIM       — размерность векторов (по умолчанию: 1536 для ada-002)
package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Пример документов — техническая документация по Go.
var documents = []draftrag.Document{
	{
		ID:      "go-goroutines",
		Content: "Горутины — лёгковесные потоки выполнения в Go, управляемые рантаймом. Создаются ключевым словом go перед вызовом функции. Горутины мультиплексируются на операционных потоках и потребляют минимум ресурсов (начальный стек ~2 КБ). Рантаймовый планировщик использует модель M:N, позволяя запускать миллионы горутин. Горутины не имеют идентификаторов — это намеренное дизайн-решение, исключающее локальное хранилище потока.",
		Metadata: map[string]string{"topic": "concurrency", "lang": "go"},
	},
	{
		ID:      "go-channels",
		Content: "Каналы — основной механизм синхронизации и передачи данных между горутинами в Go. Создаются через make(chan T, [capacity]). Небуферизованный канал блокирует отправителя до готовности получателя. Буферизованный канал блокирует только при заполнении буфера. Оператор <- используется для отправки и получения. Закрытый канал возвращает нулевое значение без блокировки. Передача по каналу гарантирует happens-before.",
		Metadata: map[string]string{"topic": "concurrency", "lang": "go"},
	},
	{
		ID:      "go-select",
		Content: "Оператор select позволяет горутине ждать нескольких операций с каналами одновременно. Выполняется первый готовый case; при нескольких готовых выбирается случайный. Блок default делает select неблокирующим. Типичное применение: таймаут (time.After), отмена контекста (ctx.Done()), опрос нескольких источников. Select без case блокирует горутину навсегда.",
		Metadata: map[string]string{"topic": "concurrency", "lang": "go"},
	},
	{
		ID:      "go-context",
		Content: "Пакет context предоставляет механизм передачи сроков, сигналов отмены и значений через границы API и между горутинами. context.Background() — корневой контекст. context.WithCancel создаёт отменяемый контекст. context.WithDeadline и context.WithTimeout добавляют ограничение по времени. Контексты образуют дерево: отмена родителя отменяет всех потомков. Значения в контексте следует использовать только для запросо-специфичных данных.",
		Metadata: map[string]string{"topic": "concurrency", "lang": "go"},
	},
	{
		ID:      "go-sync",
		Content: "Пакет sync предоставляет примитивы синхронизации: Mutex и RWMutex для взаимного исключения, WaitGroup для ожидания группы горутин, Once для однократного выполнения, Cond для условных переменных, Pool для переиспользования объектов, Map для конкурентного доступа к map. Предпочитайте каналы для передачи данных и sync для защиты разделяемого состояния.",
		Metadata: map[string]string{"topic": "concurrency", "lang": "go"},
	},
	{
		ID:      "go-errors",
		Content: "Обработка ошибок в Go строится на возвращаемых значениях типа error. Функции fmt.Errorf с %w оборачивают ошибки для цепочки вызовов. errors.Is проверяет совпадение в цепочке обёрток, errors.As извлекает конкретный тип. Sentinel-ошибки (var ErrXxx = errors.New(...)) удобны для сравнения через errors.Is. Не используйте panic для обработки ожидаемых ошибок — только для программных ошибок.",
		Metadata: map[string]string{"topic": "errors", "lang": "go"},
	},
	{
		ID:      "go-interfaces",
		Content: "Интерфейсы в Go определяют поведение через набор методов. Реализация неявная — любой тип, имеющий все методы интерфейса, автоматически его реализует. Пустой интерфейс interface{} (any) принимает любое значение. Интерфейсы следует определять в пакете потребителя, а не производителя. Небольшие интерфейсы (1-2 метода) предпочтительнее: io.Reader, io.Writer, error.",
		Metadata: map[string]string{"topic": "types", "lang": "go"},
	},
}

func main() {
	ctx := context.Background()

	dsn := mustEnv("PGVECTOR_DSN")
	embeddingDim := envIntOrDefault("EMBEDDING_DIM", 1536)
	tableName := envOrDefault("TABLE_NAME", "draftrag_example")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fatalf("ошибка открытия БД: %v\n", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		fatalf("ошибка подключения к БД: %v\n", err)
	}
	fmt.Println("Подключено к PostgreSQL.")

	fmt.Println("Применяем миграции pgvector...")
	if err := draftrag.MigratePGVector(ctx, db, draftrag.PGVectorMigrateOptions{
		PGVectorOptions: draftrag.PGVectorOptions{
			TableName:          tableName,
			EmbeddingDimension: embeddingDim,
			CreateExtension:    true,
		},
	}); err != nil {
		fatalf("ошибка миграции: %v\n", err)
	}
	fmt.Println("Схема готова.")

	store := draftrag.NewPGVectorStore(db, draftrag.PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: embeddingDim,
	})

	embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
		BaseURL: envOrDefault("EMBEDDER_BASE_URL", "https://api.openai.com"),
		APIKey:  mustEnv("EMBEDDER_API_KEY"),
		Model:   envOrDefault("EMBEDDER_MODEL", "text-embedding-ada-002"),
	})

	llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
		BaseURL: envOrDefault("LLM_BASE_URL", "https://api.openai.com"),
		APIKey:  mustEnv("LLM_API_KEY"),
		Model:   envOrDefault("LLM_MODEL", "gpt-4o-mini"),
	})

	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 3,
		Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
			ChunkSize: 400,
			Overlap:   50,
		}),
	})

	fmt.Printf("Индексируем %d документов...\n", len(documents))
	if err := pipeline.Index(ctx, documents); err != nil {
		fatalf("ошибка индексации: %v\n", err)
	}
	fmt.Println("Индексация завершена.")
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
			fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
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

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("переменная окружения %s не задана\n", key)
	}
	return v
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
