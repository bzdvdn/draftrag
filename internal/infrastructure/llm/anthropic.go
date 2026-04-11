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

type messagesResponse struct {
	Content []contentBlock `json:"content"`
	Role    string         `json:"role"`
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

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// @ds-task T1.2: Структура клиента и конструктор (DEC-001)
// ClaudeLLM реализует нативный клиент для Anthropic Messages API.
type ClaudeLLM struct {
	httpClient       *http.Client
	baseURL          string
	apiKey           string
	model            string
	anthropicVersion string
	temperature      *float64
	maxTokens        *int
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

// @ds-task T1.3: Generate реализация (AC-001, AC-002, AC-003)
// Generate генерирует текстовый ответ на основе system и user сообщений.
func (c *ClaudeLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", errors.New("userMessage is empty")
	}

	endpoint, err := buildAnthropicURL(c.baseURL)
	if err != nil {
		return "", err
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
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("anthropic-version", c.anthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = redactSecret(snippet, c.apiKey)
		return "", fmt.Errorf("anthropic request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}

	for _, block := range decoded.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return block.Text, nil
		}
	}

	return "", errors.New("invalid anthropic response: missing content text")
}

// @ds-task T2.2: GenerateStream реализация (AC-004)
// GenerateStream генерирует ответ токен за токеном через SSE streaming.
// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
func (c *ClaudeLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
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
		defer resp.Body.Close()
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = redactSecret(snippet, c.apiKey)
		return nil, fmt.Errorf("anthropic stream request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	tokenChan := make(chan string, 10)

	go func() {
		defer resp.Body.Close()
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
