package draftrag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/embedder"
)

// OllamaEmbedderOptions задаёт параметры для Ollama Embedder.
type OllamaEmbedderOptions struct {
	// BaseURL — базовый URL Ollama API. Если пустая строка, используется http://localhost:11434.
	BaseURL string
	// Model — имя модели эмбеддингов (обязательно).
	Model string
	// APIKey — опциональный ключ доступа (для кастомных инстансов с авторизацией).
	APIKey string

	// HTTPClient — опциональный клиент; если nil, используется http.DefaultClient.
	HTTPClient *http.Client
	// Timeout — опциональный таймаут на один вызов Embed.
	Timeout time.Duration
}

type ollamaEmbedder struct {
	opts OllamaEmbedderOptions
	impl *embedder.OllamaEmbedder
}

// NewOllamaEmbedder создаёт Ollama реализацию Embedder.
//
// Ошибки конфигурации возвращаются из Embed и сопоставимы через errors.Is с ErrInvalidEmbedderConfig.
func NewOllamaEmbedder(opts OllamaEmbedderOptions) Embedder {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &ollamaEmbedder{
		opts: opts,
		impl: embedder.NewOllamaEmbedder(client, opts.BaseURL, opts.APIKey, opts.Model),
	}
}

func (e *ollamaEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}
	if err := validateOllamaEmbedderOptions(e.opts); err != nil {
		return nil, err
	}
	if e.opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.opts.Timeout)
		defer cancel()
	}
	return e.impl.Embed(ctx, text)
}

func validateOllamaEmbedderOptions(opts OllamaEmbedderOptions) error {
	if strings.TrimSpace(opts.Model) == "" {
		return fmt.Errorf("%w: Model is empty", ErrInvalidEmbedderConfig)
	}
	if opts.Timeout < 0 {
		return fmt.Errorf("%w: Timeout must be >= 0", ErrInvalidEmbedderConfig)
	}
	return nil
}
