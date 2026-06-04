package draftrag

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/embedder"
)

const (
	defaultMistralEmbedModel = "mistral-embed"
)

// @sk-task llm-providers-mistral-deepseek#T2.3: MistralEmbedderOptions + NewMistralEmbedder (AC-008, AC-011)
// MistralEmbedderOptions задаёт параметры для Mistral Embedder (embeddings endpoint).
type MistralEmbedderOptions struct {
	// BaseURL — базовый URL Mistral API. Если пустая строка, используется https://api.mistral.ai.
	BaseURL string
	// APIKey — ключ доступа. Передаётся в заголовке Authorization: Bearer.
	APIKey string
	// Model — имя embeddings модели. Если пустая строка, используется mistral-embed.
	Model string

	// HTTPClient — опциональный клиент; если nil, используется http.DefaultClient.
	HTTPClient *http.Client
	// Timeout — опциональный таймаут на один вызов Embed.
	Timeout time.Duration
}

// @sk-task llm-providers-mistral-deepseek#T2.3: mistralEmbedder структура (AC-008)
type mistralEmbedder struct {
	opts MistralEmbedderOptions
	impl *embedder.OpenAICompatibleEmbedder
}

// NewMistralEmbedder создаёт Mistral реализацию Embedder через OpenAI‑совместимый embeddings endpoint.
//
// Ошибки конфигурации возвращаются из Embed и сопоставимы через errors.Is с ErrInvalidEmbedderConfig.
func NewMistralEmbedder(opts MistralEmbedderOptions) Embedder {
	if opts.BaseURL == "" {
		opts.BaseURL = defaultMistralBaseURL
	}
	if opts.Model == "" {
		opts.Model = defaultMistralEmbedModel
	}

	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &mistralEmbedder{
		opts: opts,
		impl: embedder.NewOpenAICompatibleEmbedder(client, opts.BaseURL, opts.APIKey, opts.Model),
	}
}

func (e *mistralEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return embedWithValidation(
		ctx,
		text,
		e.opts.Timeout,
		func() error { return validateMistralEmbedderOptions(e.opts) },
		e.impl.Embed,
	)
}

// @sk-task llm-providers-mistral-deepseek#T2.3: validateMistralEmbedderOptions (AC-010)
func validateMistralEmbedderOptions(opts MistralEmbedderOptions) error {
	if strings.TrimSpace(opts.APIKey) == "" {
		return fmt.Errorf("%w: APIKey is empty", ErrInvalidEmbedderConfig)
	}
	if strings.TrimSpace(opts.BaseURL) == "" {
		return fmt.Errorf("%w: BaseURL is empty", ErrInvalidEmbedderConfig)
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
