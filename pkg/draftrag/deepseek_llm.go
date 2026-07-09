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
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
	defaultDeepSeekModel   = "deepseek-chat"
)

// DeepSeekLLMOptions задаёт параметры для DeepSeek LLMProvider (Chat Completions API).
// @sk-task llm-providers-mistral-deepseek#T2.2: DeepSeekLLMOptions (AC-002, AC-006)
type DeepSeekLLMOptions struct {
	// BaseURL — базовый URL DeepSeek API. Если пустая строка, используется https://api.deepseek.com.
	BaseURL string
	// APIKey — ключ доступа. Передаётся в заголовке Authorization: Bearer.
	APIKey string
	// Model — имя модели. Если пустая строка, используется deepseek-chat.
	Model string

	Temperature *float64
	MaxTokens   *int

	HTTPClient *http.Client
	Timeout    time.Duration
}

// @sk-task llm-providers-mistral-deepseek#T2.2: deepseekLLM структура и конструктор (AC-002)
type deepseekLLM struct {
	opts DeepSeekLLMOptions
	impl *llm.OpenAIChatLLM
}

// NewDeepSeekLLM создаёт DeepSeek реализацию LLMProvider (Chat Completions API).
//
// Возвращаемый тип реализует также StreamingLLMProvider — используйте type assertion для streaming.
// Ошибки конфигурации возвращаются из Generate/GenerateStream и сопоставимы через errors.Is с ErrInvalidLLMConfig.
func NewDeepSeekLLM(opts DeepSeekLLMOptions) LLMProvider {
	if opts.BaseURL == "" {
		opts.BaseURL = defaultDeepSeekBaseURL
	}
	if opts.Model == "" {
		opts.Model = defaultDeepSeekModel
	}

	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &deepseekLLM{
		opts: opts,
		impl: llm.NewOpenAIChatLLM(client, opts.BaseURL, opts.APIKey, opts.Model, opts.Temperature, opts.MaxTokens),
	}
}

// @sk-task llm-providers-mistral-deepseek#T2.2: Generate (AC-002, AC-003)
func (p *deepseekLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return generateWithValidation(
		ctx,
		systemPrompt,
		userMessage,
		p.opts.Timeout,
		func() error { return validateDeepSeekLLMOptions(p.opts) },
		p.impl.Generate,
	)
}

// @sk-task cost-tracking: GenerateWithUsage — возвращает token usage (AC-001, RQ-001, T3.3)
func (p *deepseekLLM) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	return generateWithUsageValidation(
		ctx,
		systemPrompt,
		userMessage,
		p.opts.Timeout,
		func() error { return validateDeepSeekLLMOptions(p.opts) },
		p.impl.GenerateWithUsage,
	)
}

// @sk-task cost-tracking: ModelName — имя модели (AC-002, RQ-002, T3.3)
func (p *deepseekLLM) ModelName() string {
	return p.impl.ModelName()
}

// @sk-task llm-providers-mistral-deepseek#T2.2: GenerateStream (AC-002, AC-004)
// @sk-task arch-generics#T4.1: nil context guard вместо panic (AC-002)
func (p *deepseekLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("userMessage is empty")
	}
	if err := validateDeepSeekLLMOptions(p.opts); err != nil {
		return nil, err
	}
	return p.impl.GenerateStream(ctx, systemPrompt, userMessage)
}

// @sk-task cost-tracking: StreamUsage — делегирует impl (AC-005, RQ-006, T3.4)
func (p *deepseekLLM) StreamUsage() (domain.TokenUsage, bool) {
	return p.impl.StreamUsage()
}

// @sk-task health-check-interface#T3.5: Health на deepseekLLM (RQ-006)
func (p *deepseekLLM) Health(ctx context.Context) error {
	return p.impl.Health(ctx)
}

// @sk-task llm-providers-mistral-deepseek#T2.2: validateDeepSeekLLMOptions (AC-005)
func validateDeepSeekLLMOptions(opts DeepSeekLLMOptions) error {
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
