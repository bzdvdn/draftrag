package draftrag

import (
	"context"
	"fmt"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/chunker"
)

// BasicChunkerOptions задаёт параметры для базового Chunker.
type BasicChunkerOptions struct {
	// ChunkSize — целевой размер чанка в рунах (обязательно > 0).
	ChunkSize int
	// Overlap — перекрытие между чанками в рунах (обязательно >= 0 и < ChunkSize).
	Overlap int
	// MaxChunks — максимальное количество возвращаемых чанков (>= 0). 0 означает “без лимита”.
	//
	// Если MaxChunks > 0, чанкер возвращает префикс первых MaxChunks чанков (best-effort, без ошибки).
	MaxChunks int
}

type basicChunker struct {
	opts BasicChunkerOptions
}

// NewBasicChunker создаёт базовую реализацию Chunker.
//
// Реализация детерминированно разбивает Document.Content на чанки фиксированного размера по рунам,
// поддерживает overlap и ограничение MaxChunks, уважает context отмену.
//
// Ошибки конфигурации возвращаются из Chunk и сопоставимы через errors.Is с ErrInvalidChunkerConfig.
func NewBasicChunker(opts BasicChunkerOptions) Chunker {
	return &basicChunker{opts: opts}
}

func (c *basicChunker) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := doc.Validate(); err != nil {
		return nil, err
	}
	if err := validateBasicChunkerOptions(c.opts); err != nil {
		return nil, err
	}

	impl := chunker.NewBasicRuneChunker(c.opts.ChunkSize, c.opts.Overlap, c.opts.MaxChunks)
	return impl.Chunk(ctx, doc)
}

func validateBasicChunkerOptions(opts BasicChunkerOptions) error {
	if opts.ChunkSize <= 0 {
		return fmt.Errorf("%w: ChunkSize must be > 0", ErrInvalidChunkerConfig)
	}
	if opts.Overlap < 0 {
		return fmt.Errorf("%w: Overlap must be >= 0", ErrInvalidChunkerConfig)
	}
	if opts.Overlap >= opts.ChunkSize {
		return fmt.Errorf("%w: Overlap must be < ChunkSize", ErrInvalidChunkerConfig)
	}
	if opts.MaxChunks < 0 {
		return fmt.Errorf("%w: MaxChunks must be >= 0", ErrInvalidChunkerConfig)
	}

	return nil
}
