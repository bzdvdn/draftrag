package draftrag

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/llm"
)

// OllamaLLMOptions задаёт параметры для Ollama LLMProvider.
type OllamaLLMOptions struct {
	// BaseURL — базовый URL Ollama API. Если пустая строка, используется http://localhost:11434.
	BaseURL string
	// Model — имя модели (обязательно).
	Model string
	// APIKey — опциональный ключ доступа (для кастомных инстансов с авторизацией).
	APIKey string

	// Temperature — параметр генерации; если nil, параметр не передаётся в запросе.
	Temperature *float64
	// MaxTokens — лимит выходных токенов; если nil, параметр не передаётся в запросе.
	MaxTokens *int

	// HTTPClient — опциональный клиент; если nil, используется http.DefaultClient.
	HTTPClient *http.Client
	// Timeout — опциональный таймаут на один вызов Generate.
	Timeout time.Duration
}

type ollamaLLM struct {
	opts OllamaLLMOptions
	impl *llm.OllamaLLM
}

// NewOllamaLLM создаёт Ollama реализацию LLMProvider.
//
// Ошибки конфигурации возвращаются из Generate и сопоставимы через errors.Is с ErrInvalidLLMConfig.
func NewOllamaLLM(opts OllamaLLMOptions) LLMProvider {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &ollamaLLM{
		opts: opts,
		impl: llm.NewOllamaLLM(
			client,
			opts.BaseURL,
			opts.APIKey,
			opts.Model,
			opts.Temperature,
			opts.MaxTokens,
		),
	}
}

func (p *ollamaLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return generateWithValidation(
		ctx,
		systemPrompt,
		userMessage,
		p.opts.Timeout,
		func() error { return validateOllamaLLMOptions(p.opts) },
		p.impl.Generate,
	)
}

func validateOllamaLLMOptions(opts OllamaLLMOptions) error {
	if strings.TrimSpace(opts.Model) == "" {
		return fmt.Errorf("%w: Model is empty", ErrInvalidLLMConfig)
	}
	if opts.Timeout < 0 {
		return fmt.Errorf("%w: Timeout must be >= 0", ErrInvalidLLMConfig)
	}
	if opts.Temperature != nil && *opts.Temperature < 0 {
		return fmt.Errorf("%w: Temperature must be >= 0", ErrInvalidLLMConfig)
	}
	if opts.MaxTokens != nil && *opts.MaxTokens <= 0 {
		return fmt.Errorf("%w: MaxTokens must be > 0", ErrInvalidLLMConfig)
	}
	return nil
}
