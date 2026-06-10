package draftrag

import (
	"context"
	"net/http"
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
	return embedWithValidation(
		ctx,
		text,
		e.opts.Timeout,
		func() error { return validateOpenAICompatibleEmbedderOptions(e.opts) },
		e.impl.Embed,
	)
}

func validateOpenAICompatibleEmbedderOptions(opts OpenAICompatibleEmbedderOptions) error {
	return validateEmbedderOptions(opts.BaseURL, opts.APIKey, opts.Model, opts.Timeout)
}
