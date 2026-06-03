package draftrag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// ChromaDBOptions задаёт параметры для ChromaDB VectorStore.
type ChromaDBOptions struct {
	// BaseURL — базовый URL ChromaDB HTTP API. Если пустая строка, используется http://localhost:8000.
	BaseURL string
	// Collection — имя коллекции (обязательно).
	Collection string
	// CollectionUUID — UUID коллекции (заполняется после CreateChromaCollection,
	// т.к. ChromaDB 0.5+ требует UUID в URL).
	CollectionUUID string
	// Dimension — фиксированная размерность embedding-векторов (обязательно > 0).
	Dimension int
	// HTTP таймаут (по умолчанию: 10s)
	Timeout time.Duration
}

// Validate проверяет корректность опций.
func (o ChromaDBOptions) Validate() error {
	if o.Collection == "" {
		return fmt.Errorf("collection name is required")
	}
	if o.Dimension <= 0 {
		return fmt.Errorf("dimension must be > 0")
	}
	return nil
}

// NewChromaDBStore создаёт ChromaDB-backed реализацию VectorStore.
//
// Коллекция должна быть создана заранее. Для управления коллекциями используйте CreateChromaCollection.
func NewChromaDBStore(opts ChromaDBOptions) (VectorStore, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidVectorStoreConfig, err)
	}
	collID := opts.CollectionUUID
	if collID == "" {
		collID = opts.Collection
	}
	return vectorstore.NewChromaStore(opts.BaseURL, collID, opts.Dimension), nil
}

// CreateChromaCollection создаёт коллекцию в ChromaDB и возвращает её UUID.
//
// Использует POST /api/v1/collections с указанным именем и размерностью.
func CreateChromaCollection(ctx context.Context, opts ChromaDBOptions) (string, error) {
	if err := opts.Validate(); err != nil {
		return "", fmt.Errorf("invalid options: %w", err)
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	// Формирование тела запроса для создания коллекции
	body := map[string]interface{}{
		"name": opts.Collection,
		"metadata": map[string]interface{}{
			"dimension": opts.Dimension,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/api/v1/collections", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("chromadb: decode response: %w", err)
	}

	return result.ID, nil
}

// DeleteChromaCollection удаляет коллекцию из ChromaDB.
//
// Использует DELETE /api/v1/collections/{name}. 404 считается успехом (идемпотентность).
func DeleteChromaCollection(ctx context.Context, opts ChromaDBOptions) error {
	if opts.Collection == "" {
		return fmt.Errorf("collection name is required")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/api/v1/collections/%s", baseURL, opts.Collection)
	return deleteCollectionHTTP(ctx, client, reqURL, "chromadb")
}

// ChromaCollectionExists проверяет существование коллекции в ChromaDB.
//
// Использует GET /api/v1/collections/{name}. Возвращает true при статусе 200, false при 404.
func ChromaCollectionExists(ctx context.Context, opts ChromaDBOptions) (bool, error) {
	if opts.Collection == "" {
		return false, fmt.Errorf("collection name is required")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/api/v1/collections/%s", baseURL, opts.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	// ChromaDB возвращает 400 с "does not exist" вместо 404
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(body))
}

// GetChromaCollectionUUID возвращает UUID коллекции по её имени через GET /api/v1/collections.
func GetChromaCollectionUUID(ctx context.Context, opts ChromaDBOptions) (string, error) {
	if opts.Collection == "" {
		return "", fmt.Errorf("collection name is required")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/api/v1/collections", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var collections []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return "", fmt.Errorf("chromadb: decode response: %w", err)
	}

	for _, c := range collections {
		if c.Name == opts.Collection {
			return c.ID, nil
		}
	}

	return "", fmt.Errorf("chromadb: collection %q not found", opts.Collection)
}
