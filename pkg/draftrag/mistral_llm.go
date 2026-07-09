package draftrag //nolint:dupl // OpenAI-compatible LLM providers have similar structure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/llm"
)

const (
	defaultMistralBaseURL = "https://api.mistral.ai"
	defaultMistralModel   = "mistral-large-latest"
)

// MistralLLMOptions задаёт параметры для Mistral LLMProvider (Chat Completions API).
// @sk-task llm-providers-mistral-deepseek#T2.1: MistralLLMOptions (AC-001, AC-006)
type MistralLLMOptions struct {
	// BaseURL — базовый URL Mistral API. Если пустая строка, используется https://api.mistral.ai.
	BaseURL string
	// APIKey — ключ доступа. Передаётся в заголовке Authorization: Bearer.
	APIKey string
	// Model — имя модели. Если пустая строка, используется mistral-large-latest.
	Model string

	Temperature *float64
	MaxTokens   *int

	HTTPClient *http.Client
	Timeout    time.Duration
}

// @sk-task llm-providers-mistral-deepseek#T2.1: mistralLLM структура и конструктор (AC-001)
type mistralLLM struct {
	opts MistralLLMOptions
	impl *llm.OpenAIChatLLM
}

// NewMistralLLM создаёт Mistral реализацию LLMProvider (Chat Completions API).
//
// Возвращаемый тип реализует также StreamingLLMProvider — используйте type assertion для streaming.
// Ошибки конфигурации возвращаются из Generate/GenerateStream и сопоставимы через errors.Is с ErrInvalidLLMConfig.
func NewMistralLLM(opts MistralLLMOptions) LLMProvider {
	if opts.BaseURL == "" {
		opts.BaseURL = defaultMistralBaseURL
	}
	if opts.Model == "" {
		opts.Model = defaultMistralModel
	}

	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &mistralLLM{
		opts: opts,
		impl: llm.NewOpenAIChatLLM(client, opts.BaseURL, opts.APIKey, opts.Model, opts.Temperature, opts.MaxTokens),
	}
}

// @sk-task llm-providers-mistral-deepseek#T2.1: Generate (AC-001, AC-003)
func (p *mistralLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return generateWithValidation(
		ctx,
		systemPrompt,
		userMessage,
		p.opts.Timeout,
		func() error { return validateMistralLLMOptions(p.opts) },
		p.impl.Generate,
	)
}

// @sk-task cost-tracking: GenerateWithUsage — возвращает token usage (AC-001, RQ-001, T3.3)
func (p *mistralLLM) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	return generateWithUsageValidation(
		ctx,
		systemPrompt,
		userMessage,
		p.opts.Timeout,
		func() error { return validateMistralLLMOptions(p.opts) },
		p.impl.GenerateWithUsage,
	)
}

// @sk-task cost-tracking: ModelName — имя модели (AC-002, RQ-002, T3.3)
func (p *mistralLLM) ModelName() string {
	return p.impl.ModelName()
}

// @sk-task llm-providers-mistral-deepseek#T2.1: GenerateStream (AC-001, AC-004)
// @sk-task arch-generics#T4.1: nil context guard вместо panic (AC-002)
func (p *mistralLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("userMessage is empty")
	}
	if err := validateMistralLLMOptions(p.opts); err != nil {
		return nil, err
	}
	return p.impl.GenerateStream(ctx, systemPrompt, userMessage)
}

// @sk-task cost-tracking: StreamUsage — делегирует impl (AC-005, RQ-006, T3.4)
func (p *mistralLLM) StreamUsage() (domain.TokenUsage, bool) {
	return p.impl.StreamUsage()
}

// @sk-task health-check-interface#T3.5: Health на mistralLLM (RQ-006)
func (p *mistralLLM) Health(ctx context.Context) error {
	return p.impl.Health(ctx)
}

// @sk-task llm-providers-mistral-deepseek#T2.1: validateMistralLLMOptions (AC-005)
func validateMistralLLMOptions(opts MistralLLMOptions) error {
	if err := validateLLMOptions(opts.BaseURL, opts.APIKey, opts.Model, opts.Timeout); err != nil {
		return err
	}
	if opts.Temperature != nil && *opts.Temperature < 0 {
		return fmt.Errorf("%w: Temperature must be >= 0", ErrInvalidLLMConfig)
	}
	if opts.MaxTokens != nil && *opts.MaxTokens <= 0 {
		return fmt.Errorf("%w: MaxTokens must be > 0", ErrInvalidLLMConfig)
	}
	return nil
}
