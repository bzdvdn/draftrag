// @sk-task prod-issues#T3.1: Pinecone VectorStore (AC-007, RQ-023–RQ-028)

package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// PineconeStore реализует domain.VectorStore через Pinecone REST API.
type PineconeStore struct {
	apiKey      string
	environment string
	projectID   string
	indexName   string
	dimension   int
	cloud       string
	region      string
	host        string
	client      *http.Client
}

// PineconeOptions задаёт параметры подключения к Pinecone.
type PineconeOptions struct {
	APIKey      string
	Environment string
	ProjectID   string
	IndexName   string
	Dimension   int
	Cloud       string
	Region      string
	Timeout     time.Duration
}

type pineconeUpsertRequest struct {
	Vectors []pineconeVector `json:"vectors"`
}

type pineconeVector struct {
	ID       string                 `json:"id"`
	Values   []float64              `json:"values"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type pineconeQueryRequest struct {
	Vector          []float64 `json:"vector"`
	TopK            int       `json:"topK"`
	IncludeMetadata bool      `json:"includeMetadata"`
}

type pineconeQueryResponse struct {
	Matches []pineconeMatch `json:"matches"`
}

type pineconeMatch struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

type pineconeDeleteRequest struct {
	IDs []string `json:"ids"`
}

type pineconeStatsResponse struct {
	Dimension         int               `json:"dimension"`
	IndexFullness     float64           `json:"indexFullness"`
	TotalVectorCount  int               `json:"totalVectorCount"`
	Namespaces        map[string]interface{} `json:"namespaces"`
}

type pineconeIndexResponse struct {
	Name      string `json:"name"`
	Dimension int    `json:"dimension"`
	Metric    string `json:"metric"`
	Host      string `json:"host"`
	Status    struct {
		Ready bool   `json:"ready"`
		State string `json:"state"`
	} `json:"status"`
}

func NewPineconeStore(opts PineconeOptions) (*PineconeStore, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}
	if opts.IndexName == "" {
		return nil, fmt.Errorf("IndexName is required")
	}
	if opts.Dimension <= 0 {
		return nil, fmt.Errorf("Dimension must be > 0")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cloud := opts.Cloud
	if cloud == "" {
		cloud = "aws"
	}
	region := opts.Region
	if region == "" {
		region = "us-west-2"
	}

	store := &PineconeStore{
		apiKey:      opts.APIKey,
		environment: opts.Environment,
		projectID:   opts.ProjectID,
		indexName:   opts.IndexName,
		dimension:   opts.Dimension,
		cloud:       cloud,
		region:      region,
		client:      &http.Client{Timeout: timeout},
	}

	// Если host не указан через describe, пробуем построить из компонентов
	if opts.Environment != "" && opts.ProjectID != "" {
		store.host = fmt.Sprintf("%s-%s.svc.%s.pinecone.io", opts.IndexName, opts.ProjectID, opts.Environment)
	}

	return store, nil
}

func (s *PineconeStore) dataPlaneURL(path string) string {
	if strings.HasPrefix(s.host, "http://") || strings.HasPrefix(s.host, "https://") {
		return s.host + path
	}
	return fmt.Sprintf("https://%s%s", s.host, path)
}

func (s *PineconeStore) controlPlaneURL(path string) string {
	return fmt.Sprintf("https://api.pinecone.io%s", path)
}

func (s *PineconeStore) authHeader() (string, string) {
	return "Api-Key", s.apiKey
}

func (s *PineconeStore) doJSON(ctx context.Context, url, method string, body, resp interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(s.authHeader())

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("pinecone request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("pinecone error: status=%d, body=%s", res.StatusCode, string(raw))
	}

	if resp != nil {
		if err := json.Unmarshal(raw, resp); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (s *PineconeStore) resolveHost(ctx context.Context) error {
	if s.host != "" {
		return nil
	}
	var index pineconeIndexResponse
	url := s.controlPlaneURL("/indexes/" + s.indexName)
	if err := s.doJSON(ctx, url, http.MethodGet, nil, &index); err != nil {
		return fmt.Errorf("resolve pinecone host: %w", err)
	}
	if index.Host == "" {
		return fmt.Errorf("index %q not ready or not found", s.indexName)
	}
	s.host = index.Host
	return nil
}

// Upsert реализует domain.VectorStore.Upsert.
func (s *PineconeStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if chunk.ID == "" {
		return fmt.Errorf("chunk ID must not be empty")
	}

	if err := s.resolveHost(ctx); err != nil {
		return err
	}

	metadata := map[string]interface{}{
		"content":   chunk.Content,
		"parent_id": chunk.ParentID,
		"position":  chunk.Position,
	}
	for k, v := range chunk.Metadata {
		metadata["metadata."+k] = v
	}

	req := pineconeUpsertRequest{
		Vectors: []pineconeVector{{
			ID:       chunk.ID,
			Values:   chunk.Embedding,
			Metadata: metadata,
		}},
	}

	return s.doJSON(ctx, s.dataPlaneURL("/vectors/upsert"), http.MethodPost, req, nil)
}

// Delete реализует domain.VectorStore.Delete.
func (s *PineconeStore) Delete(ctx context.Context, id string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("id must not be empty")
	}

	if err := s.resolveHost(ctx); err != nil {
		return err
	}

	req := pineconeDeleteRequest{IDs: []string{id}}
	return s.doJSON(ctx, s.dataPlaneURL("/vectors/delete"), http.MethodPost, req, nil)
}

// Search реализует domain.VectorStore.Search.
func (s *PineconeStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if len(embedding) == 0 {
		return domain.RetrievalResult{}, fmt.Errorf("embedding must not be empty")
	}

	if err := s.resolveHost(ctx); err != nil {
		return domain.RetrievalResult{}, err
	}

	req := pineconeQueryRequest{
		Vector:          embedding,
		TopK:            topK,
		IncludeMetadata: true,
	}

	var resp pineconeQueryResponse
	if err := s.doJSON(ctx, s.dataPlaneURL("/vectors/query"), http.MethodPost, req, &resp); err != nil {
		return domain.RetrievalResult{}, err
	}

	chunks := make([]domain.RetrievedChunk, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		chunk := domain.Chunk{ID: m.ID}
		if content, ok := m.Metadata["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := m.Metadata["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := m.Metadata["position"].(float64); ok {
			chunk.Position = int(pos)
		}
		chunk.Metadata = make(map[string]string)
		for k, v := range m.Metadata {
			if strings.HasPrefix(k, "metadata.") {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[strings.TrimPrefix(k, "metadata.")] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: m.Score,
		})
	}

	return domain.RetrievalResult{Chunks: chunks}, nil
}

// Health проверяет доступность Pinecone индекса.
func (s *PineconeStore) Health(ctx context.Context) error {
	if err := s.resolveHost(ctx); err != nil {
		return err
	}

	var stats pineconeStatsResponse
	return s.doJSON(ctx, s.dataPlaneURL("/describe_index_stats"), http.MethodPost, struct{}{}, &stats)
}

// Close освобождает HTTP-клиент (реализует domain.Closer).
func (s *PineconeStore) Close() error {
	if t, ok := s.client.Transport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
	return nil
}

// CreateCollection создаёт индекс в Pinecone (реализует domain.CollectionManager).
func (s *PineconeStore) CreateCollection(ctx context.Context) error {
	body := map[string]interface{}{
		"name":      s.indexName,
		"dimension": s.dimension,
		"metric":    "cosine",
		"spec": map[string]interface{}{
			"serverless": map[string]interface{}{
				"cloud":  s.cloud,
				"region": s.region,
			},
		},
	}
	return s.doJSON(ctx, s.controlPlaneURL("/indexes"), http.MethodPost, body, nil)
}

// DeleteCollection удаляет индекс из Pinecone.
func (s *PineconeStore) DeleteCollection(ctx context.Context) error {
	url := s.controlPlaneURL("/indexes/" + s.indexName)
	return s.doJSON(ctx, url, http.MethodDelete, nil, nil)
}

// CollectionExists проверяет существование индекса.
func (s *PineconeStore) CollectionExists(ctx context.Context) (bool, error) {
	url := s.controlPlaneURL("/indexes/" + s.indexName)

	var resp pineconeIndexResponse
	err := s.doJSON(ctx, url, http.MethodGet, nil, &resp)
	if err != nil {
		if strings.Contains(err.Error(), "status=404") {
			return false, nil
		}
		return false, err
	}
	return resp.Name != "", nil
}

// DeleteByParentID не поддерживается Pinecone REST API (только удаление по ID).
// Используйте Delete с конкретными ID чанков.
func (s *PineconeStore) DeleteByParentID(ctx context.Context, parentID string) error {
	if parentID == "" {
		return fmt.Errorf("parentID must not be empty")
	}
	return errors.New("Pinecone: DeleteByParentID not supported via REST API, use Delete with specific IDs")
}
