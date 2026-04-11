// Индексация директории с текстовыми файлами — пример использования draftRAG.
//
// Рекурсивно обходит директорию, читает .txt файлы, чанкирует их и индексирует
// в in-memory store. После индексации отвечает на заданный вопрос.
//
// Переменные окружения:
//
//	EMBEDDER_BASE_URL   — базовый URL embedder API (по умолчанию: https://api.openai.com)
//	EMBEDDER_API_KEY    — ключ API для embedder (обязательно)
//	EMBEDDER_MODEL      — модель embeddings (по умолчанию: text-embedding-ada-002)
//	LLM_BASE_URL        — базовый URL LLM API (по умолчанию: https://api.openai.com)
//	LLM_API_KEY         — ключ API для LLM (обязательно)
//	LLM_MODEL           — модель LLM (по умолчанию: gpt-4o-mini)
//
// Флаги:
//
//	-dir    директория с .txt файлами (по умолчанию: .)
//	-query  вопрос для RAG (обязательно)
//	-topk   количество извлекаемых чанков (по умолчанию: 5)
//	-chunk  размер чанка в рунах (по умолчанию: 500)
//	-overlap перекрытие между чанками (по умолчанию: 60)
//
// Запуск:
//
//	EMBEDDER_API_KEY=sk-... LLM_API_KEY=sk-... \
//	  go run ./examples/index-dir/ -dir ./docs -query "Как настроить авторизацию?"
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

func main() {
	dir := flag.String("dir", ".", "директория с .txt файлами")
	query := flag.String("query", "", "вопрос для RAG (обязательно)")
	topK := flag.Int("topk", 5, "количество чанков для контекста")
	chunkSize := flag.Int("chunk", 500, "размер чанка в рунах")
	overlap := flag.Int("overlap", 60, "перекрытие между чанками в рунах")
	flag.Parse()

	if *query == "" {
		fmt.Fprintln(os.Stderr, "ошибка: флаг -query обязателен")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	docs, err := loadTextFiles(*dir)
	if err != nil {
		fatalf("ошибка загрузки файлов: %v\n", err)
	}
	if len(docs) == 0 {
		fatalf("в директории %q не найдено .txt файлов\n", *dir)
	}
	fmt.Printf("Найдено %d файлов в %q\n", len(docs), *dir)

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

	store := draftrag.NewInMemoryStore()
	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: *topK,
		Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
			ChunkSize: *chunkSize,
			Overlap:   *overlap,
		}),
	})

	fmt.Printf("Индексируем %d документов...\n", len(docs))
	if err := pipeline.Index(ctx, docs); err != nil {
		fatalf("ошибка индексации: %v\n", err)
	}
	fmt.Println("Индексация завершена.")
	fmt.Printf("\nВопрос: %s\n", *query)
	fmt.Println(strings.Repeat("─", 60))

	answer, sources, err := pipeline.Search(*query).TopK(*topK).Cite(ctx)
	if err != nil {
		fatalf("ошибка генерации ответа: %v\n", err)
	}

	fmt.Printf("\n%s\n", answer)

	if len(sources.Chunks) > 0 {
		fmt.Println("\nИсточники:")
		for i, r := range sources.Chunks {
			fmt.Printf("  [%d] %s (score=%.3f)\n      %s\n",
				i+1,
				r.Chunk.ParentID,
				r.Score,
				truncate(r.Chunk.Content, 100),
			)
		}
	}
}

func loadTextFiles(dir string) ([]draftrag.Document, error) {
	var docs []draftrag.Document
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".txt") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "предупреждение: не удалось прочитать %s: %v\n", path, err)
			return nil
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return nil
		}

		docs = append(docs, draftrag.Document{
			ID:      path,
			Content: content,
			Metadata: map[string]string{
				"filename": d.Name(),
				"path":     path,
			},
		})
		return nil
	})
	return docs, err
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
