package draftrag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	return generateWithValidation(
		ctx,
		systemPrompt,
		userMessage,
		p.opts.Timeout,
		func() error { return validateOpenAICompatibleLLMOptions(p.opts) },
		p.impl.Generate,
	)
}

// @sk-task arch-generics#T4.1: nil context guard вместо panic (AC-002)
func (p *openAICompatibleLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
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

// @sk-task health-check-interface#T3.5: Health на openAICompatibleLLM (RQ-006)
func (p *openAICompatibleLLM) Health(ctx context.Context) error {
	return p.impl.Health(ctx)
}

func validateOpenAICompatibleLLMOptions(opts OpenAICompatibleLLMOptions) error {
	if err := validateLLMOptions(opts.BaseURL, opts.APIKey, opts.Model, opts.Timeout); err != nil {
		return err
	}
	if opts.Temperature != nil && *opts.Temperature < 0 {
		return fmt.Errorf("%w: Temperature must be >= 0", ErrInvalidLLMConfig)
	}
	if opts.MaxOutputTokens != nil && *opts.MaxOutputTokens <= 0 {
		return fmt.Errorf("%w: MaxOutputTokens must be > 0", ErrInvalidLLMConfig)
	}
	return nil
}
