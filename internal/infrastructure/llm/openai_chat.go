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

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

// @sk-task llm-providers-mistral-deepseek#T1.1: Структура SSE-события для streaming (AC-004)
type chatStreamEvent struct {
	Choices []chatStreamChoice `json:"choices"`
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

// Generate sends a chat completion request and returns the generated text.
//
// @sk-task llm-providers-mistral-deepseek#T1.1: Generate реализация (AC-003)
//
//nolint:gocyclo
func (c *OpenAIChatLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", errors.New("userMessage is empty")
	}

	endpoint, err := buildChatURL(c.baseURL)
	if err != nil {
		return "", err
	}

	messages := buildMessages(systemPrompt, userMessage)

	reqBody, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = domain.RedactSecrets(snippet, c.apiKey, "Bearer "+c.apiKey)
		return "", fmt.Errorf("chat request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}

	if len(decoded.Choices) > 0 {
		if text := strings.TrimSpace(decoded.Choices[0].Message.Content); text != "" {
			return text, nil
		}
	}

	return "", errors.New("invalid chat response: missing choices[0].message.content")
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
