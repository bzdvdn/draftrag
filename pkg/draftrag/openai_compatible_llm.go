package draftrag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/llm"
)

// OpenAICompatibleLLMOptions задаёт параметры для OpenAI-compatible LLMProvider (Responses API).
type OpenAICompatibleLLMOptions struct {
	// BaseURL — базовый URL провайдера (например, "https://api.openai.com").
	BaseURL string
	// APIKey — ключ доступа. Передаётся в заголовке Authorization: Bearer.
	APIKey string
	// Model — имя модели.
	Model string

	// Temperature — параметр генерации; если nil, параметр не передаётся в запросе.
	Temperature *float64
	// MaxOutputTokens — лимит выходных токенов; если nil, параметр не передаётся в запросе.
	MaxOutputTokens *int

	// HTTPClient — опциональный клиент; если nil, используется http.DefaultClient.
	HTTPClient *http.Client
	// Timeout — опциональный таймаут на один вызов Generate.
	Timeout time.Duration
}

type openAICompatibleLLM struct {
	opts OpenAICompatibleLLMOptions
	impl *llm.OpenAICompatibleResponsesLLM
}

// NewOpenAICompatibleLLM создаёт OpenAI-compatible реализацию LLMProvider (Responses API).
//
// Возвращаемый тип реализует также StreamingLLMProvider — используйте type assertion для streaming.
// Ошибки конфигурации возвращаются из Generate/GenerateStream и сопоставимы через errors.Is с ErrInvalidLLMConfig.
func NewOpenAICompatibleLLM(opts OpenAICompatibleLLMOptions) LLMProvider {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &openAICompatibleLLM{
		opts: opts,
		impl: llm.NewOpenAICompatibleResponsesLLM(
			client,
			opts.BaseURL,
			opts.APIKey,
			opts.Model,
			opts.Temperature,
			opts.MaxOutputTokens,
		),
	}
}

func (p *openAICompatibleLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if strings.TrimSpace(userMessage) == "" {
		return "", errors.New("userMessage is empty")
	}

	if err := validateOpenAICompatibleLLMOptions(p.opts); err != nil {
		return "", err
	}

	if p.opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.opts.Timeout)
		defer cancel()
	}

	return p.impl.Generate(ctx, systemPrompt, userMessage)
}

// GenerateStream генерирует ответ токен за токеном через streaming (OpenAI Responses API).
func (p *openAICompatibleLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("userMessage is empty")
	}

	if err := validateOpenAICompatibleLLMOptions(p.opts); err != nil {
		return nil, err
	}

	// Timeout не применяется к streaming-вызовам — lifetime stream'а контролируется caller'ом через ctx.
	return p.impl.GenerateStream(ctx, systemPrompt, userMessage)
}

func validateOpenAICompatibleLLMOptions(opts OpenAICompatibleLLMOptions) error {
	if strings.TrimSpace(opts.BaseURL) == "" {
		return fmt.Errorf("%w: BaseURL is empty", ErrInvalidLLMConfig)
	}
	if strings.TrimSpace(opts.APIKey) == "" {
		return fmt.Errorf("%w: APIKey is empty", ErrInvalidLLMConfig)
	}
	if strings.TrimSpace(opts.Model) == "" {
		return fmt.Errorf("%w: Model is empty", ErrInvalidLLMConfig)
	}
	if opts.Timeout < 0 {
		return fmt.Errorf("%w: Timeout must be >= 0", ErrInvalidLLMConfig)
	}

	u, err := url.Parse(opts.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%w: BaseURL must include scheme and host", ErrInvalidLLMConfig)
	}

	if opts.Temperature != nil && *opts.Temperature < 0 {
		return fmt.Errorf("%w: Temperature must be >= 0", ErrInvalidLLMConfig)
	}
	if opts.MaxOutputTokens != nil && *opts.MaxOutputTokens <= 0 {
		return fmt.Errorf("%w: MaxOutputTokens must be > 0", ErrInvalidLLMConfig)
	}

	return nil
}
