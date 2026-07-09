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
	chatCompletionsPath = "/v1/chat/completions"
)

// @sk-task llm-providers-mistral-deepseek#T1.1: Структуры запроса/ответа Chat Completions API (AC-003, AC-004)
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type chatStreamRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   *chatUsage   `json:"usage,omitempty"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

// @sk-task llm-providers-mistral-deepseek#T1.1: Структура SSE-события для streaming (AC-004)
type chatStreamEvent struct {
	Choices []chatStreamChoice `json:"choices"`
	Usage   *chatUsage         `json:"usage,omitempty"`
}

type chatStreamChoice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}

// OpenAIChatLLM реализует клиент для OpenAI-совместимого Chat Completions API (/v1/chat/completions).
// Используется провайдерами Mistral, DeepSeek и другими OpenAI-совместимыми API.
// @sk-task llm-providers-mistral-deepseek#T1.1: Структура клиента и конструктор (DEC-001)
type OpenAIChatLLM struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	model       string
	temperature *float64
	maxTokens   *int

	streamUsageMu sync.Mutex
	streamUsage   domain.TokenUsage
}

// NewOpenAIChatLLM создаёт клиент для OpenAI-совместимого Chat Completions API.
// Если httpClient == nil, используется http.DefaultClient.
func NewOpenAIChatLLM(
	httpClient *http.Client,
	baseURL string,
	apiKey string,
	model string,
	temperature *float64,
	maxTokens *int,
) *OpenAIChatLLM {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OpenAIChatLLM{
		httpClient:  httpClient,
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

// @sk-task cost-tracking: shared generate helper с возвратом usage (AC-001, RQ-001, T3.2)
//
//nolint:gocyclo
func (c *OpenAIChatLLM) generateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.TokenUsage{}, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", domain.TokenUsage{}, errors.New("userMessage is empty")
	}

	endpoint, err := buildChatURL(c.baseURL)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	messages := buildMessages(systemPrompt, userMessage)

	reqBody, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	})
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return "", domain.TokenUsage{}, fmt.Errorf("chat request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", domain.TokenUsage{}, err
	}

	if len(decoded.Choices) == 0 {
		return "", domain.TokenUsage{}, errors.New("invalid chat response: missing choices[0].message.content")
	}
	text := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if text == "" {
		return "", domain.TokenUsage{}, errors.New("invalid chat response: missing choices[0].message.content")
	}

	usage := domain.TokenUsage{}
	if decoded.Usage != nil {
		usage.PromptTokens = decoded.Usage.PromptTokens
		usage.CompletionTokens = decoded.Usage.CompletionTokens
		usage.TotalTokens = decoded.Usage.TotalTokens
	}

	return text, usage, nil
}

// Generate sends a chat completion request and returns the generated text.
func (c *OpenAIChatLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	text, _, err := c.generateWithUsage(ctx, systemPrompt, userMessage)
	return text, err
}

// @sk-task cost-tracking: GenerateWithUsage — возвращает token usage (AC-001, RQ-001, T3.2)
func (c *OpenAIChatLLM) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	return c.generateWithUsage(ctx, systemPrompt, userMessage)
}

// @sk-task cost-tracking: ModelName — имя модели (AC-002, RQ-002, T3.2)
func (c *OpenAIChatLLM) ModelName() string {
	return c.model
}

// GenerateStream sends a streaming chat completion request and returns a channel of text tokens.
//
// @sk-task llm-providers-mistral-deepseek#T1.1: GenerateStream реализация (AC-004)
//
//nolint:gocyclo
func (c *OpenAIChatLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
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

	endpoint, err := buildChatURL(c.baseURL)
	if err != nil {
		return nil, err
	}

	messages := buildMessages(systemPrompt, userMessage)

	reqBody, err := json.Marshal(chatStreamRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
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
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
		return nil, fmt.Errorf("chat stream request failed: status=%d body=%q", resp.StatusCode, snippet)
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

			var event chatStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if event.Usage != nil {
				c.streamUsageMu.Lock()
				c.streamUsage = domain.TokenUsage{
					PromptTokens:     event.Usage.PromptTokens,
					CompletionTokens: event.Usage.CompletionTokens,
					TotalTokens:      event.Usage.TotalTokens,
				}
				c.streamUsageMu.Unlock()
			}

			if len(event.Choices) > 0 {
				if text := event.Choices[0].Delta.Content; text != "" {
					select {
					case tokenChan <- text:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return tokenChan, nil
}

// @sk-task cost-tracking: StreamUsage — возвращает usage из streaming (AC-005, RQ-006, T3.4)
// StreamUsage возвращает token usage последнего streaming-вызова.
// Должен вызываться после полного чтения канала GenerateStream.
func (c *OpenAIChatLLM) StreamUsage() (domain.TokenUsage, bool) {
	c.streamUsageMu.Lock()
	defer c.streamUsageMu.Unlock()
	if c.streamUsage.TotalTokens == 0 && c.streamUsage.PromptTokens == 0 && c.streamUsage.CompletionTokens == 0 {
		return domain.TokenUsage{}, false
	}
	return c.streamUsage, true
}

func buildMessages(systemPrompt, userMessage string) []chatMessage {
	msgs := make([]chatMessage, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: systemPrompt})
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: userMessage})
	return msgs
}

// @sk-task health-check-interface#T3.3: Health на OpenAIChatLLM (RQ-006)
func (c *OpenAIChatLLM) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL, nil)
	if err != nil {
		return fmt.Errorf("openai chat health: create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openai chat health: %w", err)
	}
	resp.Body.Close()
	return nil
}

func buildChatURL(base string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid BaseURL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid BaseURL: scheme and host are required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint := parsed.ResolveReference(&url.URL{Path: chatCompletionsPath})
	return endpoint.String(), nil
}
