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

// QdrantOptions задаёт опции для подключения к Qdrant.
//
// @ds-task T2.4: Опции подключения к Qdrant (RQ-003)
type QdrantOptions struct {
	// URL адрес Qdrant сервера (по умолчанию: http://localhost:6333)
	URL string
	// Имя коллекции для хранения чанков
	Collection string
	// Размерность векторов (обязательно)
	Dimension int
	// HTTP таймаут (по умолчанию: 10s)
	Timeout time.Duration
}

// Validate проверяет корректность опций.
func (o QdrantOptions) Validate() error {
	if o.Collection == "" {
		return fmt.Errorf("collection name is required")
	}
	if o.Dimension <= 0 {
		return fmt.Errorf("dimension must be > 0")
	}
	return nil
}

// NewQdrantStore создаёт новый VectorStore на базе Qdrant.
//
// @ds-task T2.4: Фабрика NewQdrantStore (RQ-003)
func NewQdrantStore(opts QdrantOptions) (VectorStore, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	url := opts.URL
	if url == "" {
		url = "http://localhost:6333"
	}

	return vectorstore.NewQdrantStore(url, opts.Collection, opts.Dimension), nil
}

// CreateCollection создаёт коллекцию в Qdrant с указанной размерностью.
//
// @ds-task T2.4: Миграция CreateCollection (AC-005)
func CreateCollection(ctx context.Context, opts QdrantOptions) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	url := opts.URL
	if url == "" {
		url = "http://localhost:6333"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	// Формирование тела запроса для создания коллекции
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     opts.Dimension,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/collections/%s", url, opts.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteCollection удаляет коллекцию из Qdrant.
//
// @ds-task T2.4: Миграция DeleteCollection (AC-005)
func DeleteCollection(ctx context.Context, opts QdrantOptions) error {
	if opts.Collection == "" {
		return fmt.Errorf("collection name is required")
	}

	url := opts.URL
	if url == "" {
		url = "http://localhost:6333"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/collections/%s", url, opts.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 404 допустим — коллекция уже не существует
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// CollectionExists проверяет существование коллекции в Qdrant.
//
// @ds-task T2.4: Проверка существования коллекции
func CollectionExists(ctx context.Context, opts QdrantOptions) (bool, error) {
	if opts.Collection == "" {
		return false, fmt.Errorf("collection name is required")
	}

	url := opts.URL
	if url == "" {
		url = "http://localhost:6333"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/collections/%s/exists", url, opts.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result bool `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decode response: %w", err)
	}

	return result.Result, nil
}
