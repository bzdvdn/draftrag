package chunker

import (
	"context"
	"fmt"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// BasicRuneChunker реализует базовый чанкинг по рунам с overlap и лимитом MaxChunks.
//
// Важно: реализация статлесс и не читает env vars. Контракты валидации options находятся
// в публичном wrapper'е (pkg/draftrag). Здесь предполагается, что параметры уже валидированы.
type BasicRuneChunker struct {
	chunkSize int
	overlap   int
	maxChunks int
}

// NewBasicRuneChunker создаёт базовый чанкер по рунам.
func NewBasicRuneChunker(chunkSize, overlap, maxChunks int) *BasicRuneChunker {
	return &BasicRuneChunker{
		chunkSize: chunkSize,
		overlap:   overlap,
		maxChunks: maxChunks,
	}
}

// Chunk разбивает документ на чанки.
func (c *BasicRuneChunker) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Преобразование в []rune делается после проверки ctx.Err(), чтобы отменённый/просроченный ctx
	// завершался максимально быстро (как требуется в acceptance criteria).
	contentRunes := []rune(doc.Content)
	if len(contentRunes) == 0 {
		return []domain.Chunk{}, nil
	}

	var chunks []domain.Chunk
	start := 0
	position := 0

	for start < len(contentRunes) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if c.maxChunks > 0 && len(chunks) >= c.maxChunks {
			break
		}

		end := start + c.chunkSize
		if end > len(contentRunes) {
			end = len(contentRunes)
		}

		part := strings.TrimSpace(string(contentRunes[start:end]))
		if part != "" {
			chunks = append(chunks, domain.Chunk{
				ID:       fmt.Sprintf("%s:%d", doc.ID, position),
				Content:  part,
				ParentID: doc.ID,
				Position: position,
			})
			position++
		}

		if end == len(contentRunes) {
			break
		}

		if c.overlap > 0 {
			nextStart := end - c.overlap
			// Дополнительная защита от зависания, даже если валидатор пропущен.
			if nextStart <= start {
				nextStart = start + 1
			}
			start = nextStart
		} else {
			start = end
		}
	}

	return chunks, nil
}
