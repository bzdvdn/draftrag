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
	openAIResponsesPath = "/v1/responses"
	maxErrorBodyBytes   = 4 * 1024
	maxSSEBufferBytes   = 64 * 1024
)

type responsesRequest struct {
	Model           string                  `json:"model"`
	Input           []responsesInputMessage `json:"input"`
	Temperature     *float64                `json:"temperature,omitempty"`
	MaxOutputTokens *int                    `json:"max_output_tokens,omitempty"`
}

type responsesInputMessage struct {
	Role    string                  `json:"role"`
	Content []responsesInputContent `json:"content"`
}

type responsesInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

type responsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Usage *responsesUsage `json:"usage,omitempty"`
}

// responsesStreamRequest — запрос с включенным streaming.
type responsesStreamRequest struct {
	Model           string                  `json:"model"`
	Input           []responsesInputMessage `json:"input"`
	Temperature     *float64                `json:"temperature,omitempty"`
	MaxOutputTokens *int                    `json:"max_output_tokens,omitempty"`
	Stream          bool                    `json:"stream"`
}

// streamEvent — структура SSE события от OpenAI streaming API.
type streamEvent struct {
	Type   string `json:"type"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage *responsesUsage `json:"usage,omitempty"`
}

// OpenAICompatibleResponsesLLM реализует минимальный OpenAI-compatible Responses API клиент.
type OpenAICompatibleResponsesLLM struct {
	httpClient      *http.Client
	baseURL         string
	apiKey          string
	model           string
	temperature     *float64
	maxOutputTokens *int

	streamUsageMu sync.Mutex
	streamUsage   domain.TokenUsage
}

// NewOpenAICompatibleResponsesLLM создаёт LLM-клиент для `POST /v1/responses`.
func NewOpenAICompatibleResponsesLLM(
	httpClient *http.Client,
	baseURL string,
	apiKey string,
	model string,
	temperature *float64,
	maxOutputTokens *int,
) *OpenAICompatibleResponsesLLM {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OpenAICompatibleResponsesLLM{
		httpClient:      httpClient,
		baseURL:         baseURL,
		apiKey:          apiKey,
		model:           model,
		temperature:     temperature,
		maxOutputTokens: maxOutputTokens,
	}
}

// @sk-task cost-tracking: shared generate helper с возвратом usage (AC-001, RQ-001, T2.1)
//
//nolint:gocyclo // В одном методе: валидация, HTTP, редакция и парсинг ответа.
func (l *OpenAICompatibleResponsesLLM) generateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", domain.TokenUsage{}, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", domain.TokenUsage{}, errors.New("userMessage is empty")
	}

	endpoint, err := buildResponsesURL(l.baseURL)
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	reqBody, err := json.Marshal(responsesRequest{
		Model: l.model,
		Input: []responsesInputMessage{
			{
				Role: "system",
				Content: []responsesInputContent{
					{Type: "input_text", Text: systemPrompt},
				},
			},
			{
				Role: "user",
				Content: []responsesInputContent{
					{Type: "input_text", Text: userMessage},
				},
			},
		},
		Temperature:     l.temperature,
		MaxOutputTokens: l.maxOutputTokens,
	})
	if err != nil {
		return "", domain.TokenUsage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", domain.TokenUsage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", domain.TokenUsage{}, ctxErr
		}
		return "", domain.TokenUsage{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = domain.RedactSecrets(snippet, l.apiKey, "Bearer "+l.apiKey)
		return "", domain.TokenUsage{}, fmt.Errorf("responses request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded responsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", domain.TokenUsage{}, err
	}

	var text string
	if t := strings.TrimSpace(decoded.OutputText); t != "" {
		text = t
	} else {
		for _, out := range decoded.Output {
			if out.Type != "message" {
				continue
			}
			for _, c := range out.Content {
				if c.Type == "output_text" && strings.TrimSpace(c.Text) != "" {
					text = c.Text
					break
				}
			}
			if text != "" {
				break
			}
		}
	}
	if text == "" {
		return "", domain.TokenUsage{}, errors.New("invalid responses response: missing output text")
	}

	usage := domain.TokenUsage{}
	if decoded.Usage != nil {
		usage.PromptTokens = decoded.Usage.InputTokens
		usage.CompletionTokens = decoded.Usage.OutputTokens
		usage.TotalTokens = decoded.Usage.TotalTokens
	}

	return text, usage, nil
}

// Generate генерирует текстовый ответ на основе system и user сообщений.
func (l *OpenAICompatibleResponsesLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	text, _, err := l.generateWithUsage(ctx, systemPrompt, userMessage)
	return text, err
}

// @sk-task cost-tracking: GenerateWithUsage — возвращает token usage (AC-001, RQ-001, T2.1)
func (l *OpenAICompatibleResponsesLLM) GenerateWithUsage(ctx context.Context, systemPrompt, userMessage string) (string, domain.TokenUsage, error) {
	return l.generateWithUsage(ctx, systemPrompt, userMessage)
}

// @sk-task cost-tracking: ModelName — имя модели (AC-002, RQ-002, T2.1)
func (l *OpenAICompatibleResponsesLLM) ModelName() string {
	return l.model
}

// @sk-task health-check-interface#T3.3: Health на OpenAICompatibleResponsesLLM (RQ-006)
func (l *OpenAICompatibleResponsesLLM) Health(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, l.baseURL, nil)
	if err != nil {
		return fmt.Errorf("openai-compatible responses health: create request: %w", err)
	}
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openai-compatible responses health: %w", err)
	}
	resp.Body.Close()
	return nil
}

func buildResponsesURL(base string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid BaseURL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid BaseURL: scheme and host are required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint := parsed.ResolveReference(&url.URL{Path: openAIResponsesPath})
	return endpoint.String(), nil
}

func readBodySnippet(r io.Reader, limit int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateStream генерирует ответ токен за токеном через SSE streaming.
// Возвращает канал для чтения текстовых чанков; канал закрывается при завершении или ошибке.
//
// @ds-task T2.1: Реализовать GenerateStream с SSE парсингом (AC-001, AC-003, AC-005, DEC-002)
// @ds-task T2.2: Обработка SSE edge cases (AC-005)
//
//nolint:gocyclo // SSE-парсинг и обработка ошибок/контекста держим вместе.
func (l *OpenAICompatibleResponsesLLM) GenerateStream(ctx context.Context, systemPrompt, userMessage string) (<-chan string, error) {
	if ctx == nil {
		panic("nil context")
	}
	l.streamUsageMu.Lock()
	l.streamUsage = domain.TokenUsage{}
	l.streamUsageMu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("userMessage is empty")
	}

	endpoint, err := buildResponsesURL(l.baseURL)
	if err != nil {
		return nil, err
	}

	reqBody, err := json.Marshal(responsesStreamRequest{
		Model: l.model,
		Input: []responsesInputMessage{
			{
				Role: "system",
				Content: []responsesInputContent{
					{Type: "input_text", Text: systemPrompt},
				},
			},
			{
				Role: "user",
				Content: []responsesInputContent{
					{Type: "input_text", Text: userMessage},
				},
			},
		},
		Temperature:     l.temperature,
		MaxOutputTokens: l.maxOutputTokens,
		Stream:          true,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := l.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer func() { _ = resp.Body.Close() }()
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = domain.RedactSecrets(snippet, l.apiKey, "Bearer "+l.apiKey)
		return nil, fmt.Errorf("responses stream request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	// Канал для передачи токенов потребителю
	tokenChan := make(chan string, 10)

	// Горутина-производитель читает SSE и пишет в канал
	go func() {
		defer func() { _ = resp.Body.Close() }()
		defer close(tokenChan)

		reader := io.LimitReader(resp.Body, maxSSEBufferBytes)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 4096), maxSSEBufferBytes)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				// Контекст отменён — выходим
				return
			default:
			}

			line := scanner.Text()

			// Игнорируем пустые линии и комментарии (ping)
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// SSE формат: "data: <json>"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// [DONE] — конец streaming'а
			if data == "[DONE]" {
				return
			}

			var event streamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				// Пропускаем невалидные события
				continue
			}

			if event.Usage != nil {
				l.streamUsageMu.Lock()
				l.streamUsage = domain.TokenUsage{
					PromptTokens:     event.Usage.InputTokens,
					CompletionTokens: event.Usage.OutputTokens,
					TotalTokens:      event.Usage.TotalTokens,
				}
				l.streamUsageMu.Unlock()
			}

			// Извлекаем текст из события
			var text string
			if event.Delta.Text != "" {
				text = event.Delta.Text
			} else if len(event.Output) > 0 {
				for _, out := range event.Output {
					if out.Type == "message" {
						for _, c := range out.Content {
							if c.Type == "output_text" {
								text += c.Text
							}
						}
					}
				}
			}

			if text != "" {
				select {
				case tokenChan <- text:
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
func (l *OpenAICompatibleResponsesLLM) StreamUsage() (domain.TokenUsage, bool) {
	l.streamUsageMu.Lock()
	defer l.streamUsageMu.Unlock()
	if l.streamUsage.TotalTokens == 0 && l.streamUsage.PromptTokens == 0 && l.streamUsage.CompletionTokens == 0 {
		return domain.TokenUsage{}, false
	}
	return l.streamUsage, true
}
