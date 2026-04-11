package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

const (
	openAIEmbeddingsPath = "/v1/embeddings"
	maxErrorBodyBytes    = 4 * 1024
)

type embeddingsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// OpenAICompatibleEmbedder реализует запрос к embeddings endpoint в формате OpenAI-compatible.
//
// Реализация использует минимальный контракт:
// - POST {BaseURL}/v1/embeddings
// - Authorization: Bearer {APIKey}
// - request: {model, input}
// - response: data[0].embedding
//
// Важно: эта реализация не читает env vars и не хранит persisted состояние.
type OpenAICompatibleEmbedder struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	model      string
}

// NewOpenAICompatibleEmbedder создаёт embedder поверх OpenAI-compatible embeddings endpoint.
func NewOpenAICompatibleEmbedder(httpClient *http.Client, baseURL, apiKey, model string) *OpenAICompatibleEmbedder {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &OpenAICompatibleEmbedder{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     apiKey,
		model:      model,
	}
}

// Embed вычисляет embedding для текста.
func (e *OpenAICompatibleEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}

	endpoint, err := buildEmbeddingsURL(e.baseURL)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(embeddingsRequest{
		Model: e.model,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := readBodySnippet(resp.Body, maxErrorBodyBytes)
		snippet = redactSecret(snippet, e.apiKey)
		return nil, fmt.Errorf("embeddings request failed: status=%d body=%q", resp.StatusCode, snippet)
	}

	var decoded embeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if len(decoded.Data) == 0 {
		return nil, errors.New("invalid embeddings response: missing data")
	}
	embedding := decoded.Data[0].Embedding
	if len(embedding) == 0 {
		return nil, errors.New("invalid embeddings response: empty embedding")
	}
	for i, v := range embedding {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid embeddings response: non-finite value at index %d", i)
		}
	}

	return embedding, nil
}

func buildEmbeddingsURL(base string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid BaseURL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid BaseURL: scheme and host are required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint := parsed.ResolveReference(&url.URL{Path: openAIEmbeddingsPath})
	return endpoint.String(), nil
}

func readBodySnippet(r io.Reader, limit int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func redactSecret(text, secret string) string {
	if secret == "" {
		return text
	}
	return strings.ReplaceAll(text, secret, "<redacted>")
}
