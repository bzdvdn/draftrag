package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

const (
	defaultCohereModel   = "rerank-english-v3.0"
	defaultCohereBaseURL = "https://api.cohere.com/v2"
	defaultMaxRetries    = 2
	defaultMaxTokensDoc  = 4096
)

// CohereRerankOptions задаёт параметры для Cohere Rerank API v2.
type CohereRerankOptions struct {
	APIKey          string
	Model           string
	BaseURL         string
	Timeout         time.Duration
	MaxRetries      int
	MaxTokensPerDoc int
	HTTPClient      *http.Client
}

// CohereReranker реализует Reranker и BatchReranker через Cohere Rerank API v2.
type CohereReranker struct {
	opts CohereRerankOptions
}

// NewCohereRerank создаёт CohereReranker. APIKey обязателен.
//
// @sk-task reranker-cross-encoder#T2.1: конструктор CohereReranker (AC-001, AC-003)
// @sk-task reranker-cross-encoder#T3.1: error handling 401/429/5xx/таймаут (AC-006)
func NewCohereRerank(opts CohereRerankOptions) (*CohereReranker, error) {
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, fmt.Errorf("%w: APIKey is required", ErrInvalidRerankerConfig)
	}
	if opts.Model == "" {
		opts.Model = defaultCohereModel
	}
	if opts.BaseURL == "" {
		opts.BaseURL = defaultCohereBaseURL
	}
	if _, err := url.Parse(opts.BaseURL); err != nil {
		return nil, fmt.Errorf("%w: invalid BaseURL: %w", ErrInvalidRerankerConfig, err)
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = defaultMaxRetries
	}
	if opts.MaxTokensPerDoc <= 0 {
		opts.MaxTokensPerDoc = defaultMaxTokensDoc
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	return &CohereReranker{opts: opts}, nil
}

// Rerank отправляет один запрос к Cohere Rerank API и возвращает переранжированные чанки.
func (c *CohereReranker) Rerank(ctx context.Context, query string, chunks []domain.RetrievedChunk) ([]domain.RetrievedChunk, error) {
	if len(chunks) == 0 {
		return chunks, nil
	}
	documents := make([]string, len(chunks))
	for i, ch := range chunks {
		documents[i] = ch.Chunk.Content
	}
	body := cohereRerankRequest{
		Model:           c.opts.Model,
		Query:           query,
		Documents:       documents,
		MaxTokensPerDoc: &c.opts.MaxTokensPerDoc,
	}
	resp, err := c.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	return c.mapResults(resp.Results, chunks), nil
}

// RerankBatch выполняет N запросов к Cohere Rerank API конкурентно (fan-out).
//
// @sk-task reranker-cross-encoder#T3.2: BatchReranker.RerankBatch implementation (AC-008)
func (c *CohereReranker) RerankBatch(ctx context.Context, queries []string, chunks []domain.RetrievedChunk) ([][]domain.RetrievedChunk, error) {
	if len(queries) == 0 {
		return nil, nil
	}
	if len(chunks) == 0 {
		out := make([][]domain.RetrievedChunk, len(queries))
		return out, nil
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([][]domain.RetrievedChunk, len(queries))
	var firstErr error

	for i, q := range queries {
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			r, err := c.Rerank(ctx, q, chunks)
			mu.Lock()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			results[i] = r
			mu.Unlock()
		}(i, q)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (c *CohereReranker) doRequest(ctx context.Context, body cohereRerankRequest) (*cohereRerankResponse, error) {
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("reranker: marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.opts.BaseURL+"/rerank", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("reranker: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.opts.APIKey)
		req.Header.Set("Content-Type", "application/json")

		client := c.opts.HTTPClient
		if c.opts.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.opts.Timeout)
			defer cancel()
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("reranker: request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reranker: read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var result cohereRerankResponse
			if err := json.Unmarshal(respBody, &result); err != nil {
				return nil, fmt.Errorf("reranker: unmarshal response: %w", err)
			}
			return &result, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("reranker: cohere API %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		return nil, fmt.Errorf("reranker: cohere API %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, lastErr
}

func (c *CohereReranker) mapResults(results []cohereRerankResult, chunks []domain.RetrievedChunk) []domain.RetrievedChunk {
	if len(results) == 0 {
		return chunks
	}

	type scored struct {
		chunk domain.RetrievedChunk
		score float64
	}
	scoredChunks := make([]scored, 0, len(results))
	for _, r := range results {
		if r.Index >= 0 && r.Index < len(chunks) {
			scoredChunks = append(scoredChunks, scored{
				chunk: chunks[r.Index],
				score: r.RelevanceScore,
			})
		}
	}

	if len(scoredChunks) < len(chunks) {
		seen := make(map[int]bool)
		for _, r := range results {
			if r.Index >= 0 && r.Index < len(chunks) {
				seen[r.Index] = true
			}
		}
		for i, ch := range chunks {
			if !seen[i] {
				scoredChunks = append(scoredChunks, scored{
					chunk: ch,
					score: 0,
				})
			}
		}
	}

	sort.SliceStable(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].score > scoredChunks[j].score
	})

	out := make([]domain.RetrievedChunk, len(scoredChunks))
	for i, s := range scoredChunks {
		s.chunk.Score = s.score
		out[i] = s.chunk
	}
	return out
}

type cohereRerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	MaxTokensPerDoc *int     `json:"max_tokens_per_doc,omitempty"`
}

type cohereRerankResponse struct {
	ID      string               `json:"id"`
	Results []cohereRerankResult `json:"results"`
}

type cohereRerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}
