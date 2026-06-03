package shared

import (
	"fmt"
	"os"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// @sk-task docs-and-examples#T1.4: PrintAnswer форматирует и выводит ответ RAG-пайплайна в stdout (AC-006).
// Формат: [Q] <truncated question>\n[A] <answer>\n[Sources] <N> chunks
// Где N — суммарное количество чанков по всем RetrievalResult'ам в sources.
func PrintAnswer(question, answer string, sources []draftrag.RetrievalResult) {
	q := strings.TrimSpace(question)
	if len(q) > 200 {
		q = q[:200] + "..."
	}
	fmt.Printf("[Q] %s\n", q)
	fmt.Printf("[A] %s\n", strings.TrimSpace(answer))
	totalChunks := 0
	for _, s := range sources {
		totalChunks += len(s.Chunks)
	}
	fmt.Printf("[Sources] %d chunks\n", totalChunks)
	if totalChunks == 0 {
		return
	}
	for i, s := range sources {
		if i >= 3 {
			fmt.Printf("  ... and %d more results\n", len(sources)-3)
			break
		}
		// Извлекаем parentID из первого чанка для краткости.
		parentID := ""
		var score float64
		if len(s.Chunks) > 0 {
			parentID = s.Chunks[0].Chunk.ParentID
			score = s.Chunks[0].Score
		}
		fmt.Printf("  %d. parent_id=%s score=%.4f\n", i+1, parentID, score)
	}
}

// @sk-task docs-and-examples#T1.4: PrintInfo выводит информационное сообщение (AC-006).
// Используется для логирования этапов: индексация, поиск, генерация.
func PrintInfo(format string, args ...any) {
	fmt.Printf("[info] "+format+"\n", args...)
}

// @sk-task docs-and-examples#T1.4: PrintError выводит сообщение об ошибке в stderr (AC-011).
// Формат единый: [error] <message>.
func PrintError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[error] "+format+"\n", args...)
}
