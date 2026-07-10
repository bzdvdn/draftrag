package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/bzdvdn/draftrag/internal/domain"
)

const (
	anthropicMessagesPath   = "/v1/messages"
	defaultAnthropicVersion = "2023-06-01"
	defaultAnthropicModel   = "claude-3-haiku-20240307"
	defaultMaxTokens        = 1024
)

// @ds-task T1.1: Структуры данных для Anthropic Messages API (AC-002)
type messagesRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	System      string           `json:"system,omitempty"`
	Messages    []messageContent `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
}

type messageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type messagesResponse struct {
	Content []contentBlock  `json:"content"`
	Role    string          `json:"role"`
	Usage   *anthropicUsage `json:"usage,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// @ds-task T2.1: Структуры для streaming (AC-004)
type messagesStreamRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	System      string           `json:"system,omitempty"`
	Messages    []messageContent `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
	Stream      bool             `json:"stream"`
}

type anthropicStreamUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage *anthropicStreamUsage `json:"usage,omitempty"`
}

// ClaudeLLM реализует нативный клиент для Anthropic Messages API.
// @ds-task T1.2: Структура клиента и конструктор (DEC-001)
type ClaudeLLM struct {
	httpClient       *http.Client
	baseURL          string
	apiKey           string
	model            string
	anthropicVersion string
	temperature      *float64
	maxTokens        *int

	streamUsageMu sync.Mutex
	streamUsage   domain.TokenUsage
}

// NewClaudeLLM создаёт клиент для Anthropic Messages API.
// Если httpClient == nil, используется http.DefaultClient.
// Если model == "", используется defaultAnthropicModel.
// Если anthropicVersion == "", используется defaultAnthropicVersion.
func NewClaudeLLM(
	httpClient *http.Client,
	baseURL string,
	apiKey string,
	model string,
	anthropicVersion string,
	temperature *float64,
	maxTokens *int,
) *ClaudeLLM {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if model == "" {
		model = defaultAnthropicModel
	}
	if anthropicVersion == "" {
		anthropicVersion = defaultAnthropicVersion
	}
	return &ClaudeLLM{
		httpClient:       httpClient,
		baseURL:          baseURL,
		apiKey:           apiKey,
		model:            model,
		anthropicVersion: anthropicVersion,
		temperature:      temperature,
		maxTokens:        maxTokens,
	}
}

// @sk-task cost-tracking: shared generate helper с возвратом usage (AC-001, RQ-001, T3.1)
//
//nolint:gocyclo // Валидация, HTTP, редакция и парсинг ответа в одном методе для читабельности.
func (c *ClaudeLLM) generateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.TokenUsage{}, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", domain.TokenUsage{}, errors.New("userMessage is empty")
	}

	endpoint, err := buildAnthropicURL(c.baseURL)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	maxTokens := defaultMaxTokens
	if c.maxTokens != nil {
		maxTokens = *c.maxTokens
	}

	reqBody, err := json.Marshal(messagesRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []messageContent{
			{Role: "user", Content: userMessage},
		},
		Temperature: c.temperature,
	})
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("anthropic-version", c.anthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", domain.TokenUsage{}, ctxErr
		}
		return "", domain.TokenUsage{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = domain.RedactSecrets(snippet, c.apiKey, "Bearer "+c.apiKey)
		return "", domain.TokenUsage{}, fmt.Errorf("anthropic request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", domain.TokenUsage{}, err
	}

	var text string
	for _, block := range decoded.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			text = block.Text
			break
		}
	}
	if text == "" {
		return "", domain.TokenUsage{}, errors.New("invalid anthropic response: missing content text")
	}

	usage := domain.TokenUsage{}
	if decoded.Usage != nil {
		usage.PromptTokens = decoded.Usage.InputTokens
		usage.CompletionTokens = decoded.Usage.OutputTokens
		usage.TotalTokens = decoded.Usage.InputTokens + decoded.Usage.OutputTokens
	}

	return text, usage, nil
}

// Generate генерирует текстовый ответ на основе system и user сообщений.
func (c *ClaudeLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	text, _, err := c.generateWithUsage(ctx, systemPrompt, userMessage)
	return text, err
}

// @sk-task cost-tracking: GenerateWithUsage — возвращает token usage (AC-001, RQ-001, T3.1)
func (c *ClaudeLLM) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	return c.generateWithUsage(ctx, systemPrompt, userMessage)
}

// @sk-task cost-tracking: ModelName — имя модели (AC-002, RQ-002, T3.1)
func (c *ClaudeLLM) ModelName() string {
	return c.model
}

// GenerateStream генерирует ответ токен за токеном через SSE streaming.
// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
// @ds-task T2.2: GenerateStream реализация (AC-004)
//
//nolint:gocyclo // SSE-парсинг и обработка ошибок/контекста держим вместе.
func (c *ClaudeLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	c.streamUsageMu.Lock()
	c.streamUsage = domain.TokenUsage{}
	c.streamUsageMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("userMessage is empty")
	}

	endpoint, err := buildAnthropicURL(c.baseURL)
	if err != nil {
		return nil, err
	}

	maxTokens := defaultMaxTokens
	if c.maxTokens != nil {
		maxTokens = *c.maxTokens
	}

	reqBody, err := json.Marshal(messagesStreamRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []messageContent{
			{Role: "user", Content: userMessage},
		},
		Temperature: c.temperature,
		Stream:      true,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("anthropic-version", c.anthropicVersion)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = domain.RedactSecrets(snippet, c.apiKey, "Bearer "+c.apiKey)
		return nil, fmt.Errorf("anthropic stream request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	tokenChan := make(chan string, 10)

	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(tokenChan)

		reader := io.LimitReader(resp.Body, maxSSEBufferBytes)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 4096), maxSSEBufferBytes)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()

			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				return
			}

			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Usage != nil {
				c.streamUsageMu.Lock()
				if event.Usage.InputTokens > 0 {
					c.streamUsage.PromptTokens = event.Usage.InputTokens
				}
				if event.Usage.OutputTokens > 0 {
					c.streamUsage.CompletionTokens = event.Usage.OutputTokens
				}
				c.streamUsage.TotalTokens = c.streamUsage.PromptTokens + c.streamUsage.CompletionTokens
				c.streamUsageMu.Unlock()
			}

			if event.Delta.Text != "" {
				select {
				case tokenChan <- event.Delta.Text:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return tokenChan, nil
}

// @sk-task cost-tracking: StreamUsage — возвращает usage из streaming (AC-005, RQ-006, T3.4)
// StreamUsage возвращает token usage последнего streaming-вызова.
// Должен вызываться после полного чтения канала GenerateStream.
func (c *ClaudeLLM) StreamUsage() (domain.TokenUsage, bool) {
	c.streamUsageMu.Lock()
	defer c.streamUsageMu.Unlock()
	if c.streamUsage.TotalTokens == 0 && c.streamUsage.PromptTokens == 0 && c.streamUsage.CompletionTokens == 0 {
		return domain.TokenUsage{}, false
	}
	return c.streamUsage, true
}

// @sk-task health-check-interface#T3.3: Health на ClaudeLLM (RQ-006)
func (c *ClaudeLLM) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL, nil)
	if err != nil {
		return fmt.Errorf("claude health: create request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.anthropicVersion)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("claude health: %w", err)
	}
	resp.Body.Close()
	return nil
}

func buildAnthropicURL(base string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid BaseURL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid BaseURL: scheme and host are required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint := parsed.ResolveReference(&url.URL{Path: anthropicMessagesPath})
	return endpoint.String(), nil
}
