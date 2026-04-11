package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	ollamaChatPath         = "/api/chat"
	ollamaDefaultBaseURL   = "http://localhost:11434"
	ollamaDefaultMaxTokens = 1024
)

// @ds-task T1.1: Структуры запроса и ответа для Ollama Chat API (AC-001, DEC-001)
type ollamaChatRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// @ds-task T1.1: Структура ответа от Ollama Chat API (AC-001)
type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// @ds-task T1.1: Структура клиента и конструктор (AC-001, DEC-001, DEC-003)
// OllamaLLM реализует LLMProvider для локального Ollama API.
type OllamaLLM struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	model       string
	temperature *float64
	maxTokens   *int
}

// NewOllamaLLM создаёт клиент для Ollama Chat API.
// Если httpClient == nil, используется http.DefaultClient.
// Если baseURL == "", используется ollamaDefaultBaseURL (http://localhost:11434).
// Если model == "", используется пустая строка (должна быть задана явно).
func NewOllamaLLM(
	httpClient *http.Client,
	baseURL string,
	apiKey string,
	model string,
	temperature *float64,
	maxTokens *int,
) *OllamaLLM {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = ollamaDefaultBaseURL
	}
	return &OllamaLLM{
		httpClient:  httpClient,
		baseURL:     baseURL,
		apiKey:      apiKey,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

// @ds-task T2.1: Реализация Generate для Ollama Chat API (AC-001, AC-003, AC-004, AC-005, RQ-001, RQ-002, RQ-003, RQ-006, RQ-007)
// Generate генерирует текстовый ответ на основе system и user сообщений.
func (o *OllamaLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", errors.New("userMessage is empty")
	}

	endpoint, err := buildOllamaChatURL(o.baseURL)
	if err != nil {
		return "", err
	}

	messages := []ollamaMessage{}
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, ollamaMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, ollamaMessage{Role: "user", Content: userMessage})

	reqBody, err := json.Marshal(ollamaChatRequest{
		Model:       o.model,
		Messages:    messages,
		Stream:      false,
		Temperature: o.temperature,
		MaxTokens:   o.maxTokens,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		return "", fmt.Errorf("ollama chat request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}

	content := strings.TrimSpace(decoded.Message.Content)
	if content == "" {
		return "", errors.New("invalid ollama response: empty message content")
	}

	return content, nil
}

func buildOllamaChatURL(base string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid BaseURL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid BaseURL: scheme and host are required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint := parsed.ResolveReference(&url.URL{Path: ollamaChatPath})
	return endpoint.String(), nil
}
