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

// AnthropicLLMOptions задаёт параметры для Anthropic (Claude) LLMProvider.
type AnthropicLLMOptions struct {
	// BaseURL — базовый URL Anthropic API (например, "https://api.anthropic.com").
	BaseURL string
	// APIKey — ключ доступа. Передаётся в заголовке X-API-Key.
	APIKey string
	// Model — имя модели. Если пустая строка, используется claude-3-haiku-20240307.
	Model string
	// AnthropicVersion — версия API. Если пустая строка, используется "2023-06-01".
	AnthropicVersion string

	// Temperature — параметр генерации; если nil, параметр не передаётся в запросе.
	Temperature *float64
	// MaxTokens — лимит выходных токенов; если nil, используется дефолт (1024).
	MaxTokens *int

	// HTTPClient — опциональный клиент; если nil, используется http.DefaultClient.
	HTTPClient *http.Client
	// Timeout — опциональный таймаут на один вызов Generate (не применяется к GenerateStream).
	Timeout time.Duration
}

type anthropicLLM struct {
	opts AnthropicLLMOptions
	impl *llm.ClaudeLLM
}

// NewAnthropicLLM создаёт Anthropic (Claude) реализацию LLMProvider.
//
// Возвращаемый тип реализует также StreamingLLMProvider — используйте type assertion для streaming.
// Ошибки конфигурации возвращаются из Generate/GenerateStream и сопоставимы через errors.Is с ErrInvalidLLMConfig.
func NewAnthropicLLM(opts AnthropicLLMOptions) LLMProvider {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &anthropicLLM{
		opts: opts,
		impl: llm.NewClaudeLLM(
			client,
			opts.BaseURL,
			opts.APIKey,
			opts.Model,
			opts.AnthropicVersion,
			opts.Temperature,
			opts.MaxTokens,
		),
	}
}

func (p *anthropicLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", errors.New("userMessage is empty")
	}
	if err := validateAnthropicLLMOptions(p.opts); err != nil {
		return "", err
	}
	if p.opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.opts.Timeout)
		defer cancel()
	}
	return p.impl.Generate(ctx, systemPrompt, userMessage)
}

// GenerateStream генерирует ответ токен за токеном через SSE streaming (Anthropic Messages API).
func (p *anthropicLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("userMessage is empty")
	}
	if err := validateAnthropicLLMOptions(p.opts); err != nil {
		return nil, err
	}
	// Timeout не применяется к streaming-вызовам — lifetime stream'а контролируется caller'ом через ctx.
	return p.impl.GenerateStream(ctx, systemPrompt, userMessage)
}

func validateAnthropicLLMOptions(opts AnthropicLLMOptions) error {
	if strings.TrimSpace(opts.BaseURL) == "" {
		return fmt.Errorf("%w: BaseURL is empty", ErrInvalidLLMConfig)
	}
	if strings.TrimSpace(opts.APIKey) == "" {
		return fmt.Errorf("%w: APIKey is empty", ErrInvalidLLMConfig)
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
	if opts.MaxTokens != nil && *opts.MaxTokens <= 0 {
		return fmt.Errorf("%w: MaxTokens must be > 0", ErrInvalidLLMConfig)
	}
	return nil
}
