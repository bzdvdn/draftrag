package embedder

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
	ollamaEmbeddingsPath = "/api/embeddings"
	ollamaDefaultBaseURL = "http://localhost:11434"
)

// @ds-task T1.2: Структуры запроса и ответа для Ollama Embeddings API (AC-002, DEC-001)
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// @ds-task T1.2: Структура ответа от Ollama Embeddings API (AC-002)
type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// OllamaEmbedder реализует Embedder для локального Ollama API.
// @ds-task T1.2: Структура клиента и конструктор (AC-002, DEC-001, DEC-003)
type OllamaEmbedder struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

// NewOllamaEmbedder создаёт embedder для Ollama Embeddings API.
// Если httpClient == nil, используется http.DefaultClient.
// Если baseURL == "", используется ollamaDefaultBaseURL (http://localhost:11434).
// Если model == "", используется пустая строка (должна быть задана явно).
func NewOllamaEmbedder(httpClient *http.Client, baseURL, apiKey, model string) *OllamaEmbedder {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = ollamaDefaultBaseURL
	}
	return &OllamaEmbedder{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
	}
}

// Embed вычисляет embedding для текста.
// @ds-task T2.2: Реализация Embed для Ollama Embeddings API (AC-002, AC-003, AC-004, AC-005, RQ-004, RQ-005, RQ-006, RQ-007)
func (o *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}

	endpoint, err := buildOllamaEmbeddingsURL(o.baseURL)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(ollamaEmbedRequest{
		Model:  o.model,
		Prompt: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	return parseOllamaEmbeddingResponse(resp)
}

func buildOllamaEmbeddingsURL(base string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid BaseURL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid BaseURL: scheme and host are required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint := parsed.ResolveReference(&url.URL{Path: ollamaEmbeddingsPath})
	return endpoint.String(), nil
}

func parseOllamaEmbeddingResponse(resp *http.Response) ([]float64, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		return nil, fmt.Errorf("ollama embeddings request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	embedding := decoded.Embedding
	if len(embedding) == 0 {
		return nil, errors.New("invalid ollama embeddings response: empty embedding")
	}

	if err := validateFiniteVector(embedding, "invalid ollama embeddings response"); err != nil {
		return nil, err
	}
	return embedding, nil
}
