package draftrag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/embedder"
)

// OpenAICompatibleEmbedderOptions задаёт параметры для OpenAI-compatible Embedder.
type OpenAICompatibleEmbedderOptions struct {
	// BaseURL — базовый URL провайдера (например, "https://api.openai.com").
	BaseURL string
	// APIKey — ключ доступа. Передаётся в заголовке Authorization: Bearer.
	APIKey string
	// Model — имя embeddings модели.
	Model string

	// HTTPClient — опциональный клиент; если nil, используется http.DefaultClient.
	HTTPClient *http.Client
	// Timeout — опциональный таймаут на один вызов Embed.
	Timeout time.Duration
}

type openAICompatibleEmbedder struct {
	opts OpenAICompatibleEmbedderOptions
	impl *embedder.OpenAICompatibleEmbedder
}

// NewOpenAICompatibleEmbedder создаёт OpenAI-compatible реализацию Embedder.
//
// Ошибки конфигурации возвращаются из Embed и сопоставимы через errors.Is с ErrInvalidEmbedderConfig.
func NewOpenAICompatibleEmbedder(opts OpenAICompatibleEmbedderOptions) Embedder {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &openAICompatibleEmbedder{
		opts: opts,
		impl: embedder.NewOpenAICompatibleEmbedder(client, opts.BaseURL, opts.APIKey, opts.Model),
	}
}

func (e *openAICompatibleEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}

	if err := validateOpenAICompatibleEmbedderOptions(e.opts); err != nil {
		return nil, err
	}

	if e.opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.opts.Timeout)
		defer cancel()
	}

	return e.impl.Embed(ctx, text)
}

func validateOpenAICompatibleEmbedderOptions(opts OpenAICompatibleEmbedderOptions) error {
	if strings.TrimSpace(opts.BaseURL) == "" {
		return fmt.Errorf("%w: BaseURL is empty", ErrInvalidEmbedderConfig)
	}
	if strings.TrimSpace(opts.APIKey) == "" {
		return fmt.Errorf("%w: APIKey is empty", ErrInvalidEmbedderConfig)
	}
	if strings.TrimSpace(opts.Model) == "" {
		return fmt.Errorf("%w: Model is empty", ErrInvalidEmbedderConfig)
	}
	if opts.Timeout < 0 {
		return fmt.Errorf("%w: Timeout must be >= 0", ErrInvalidEmbedderConfig)
	}

	u, err := url.Parse(opts.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: BaseURL must include scheme and host", ErrInvalidEmbedderConfig)
	}

	return nil
}
