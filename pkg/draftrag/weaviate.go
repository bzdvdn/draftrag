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

// WeaviateOptions задаёт параметры подключения к Weaviate.
//
// @ds-task T3.1: Публичный API WeaviateOptions (AC-005, RQ-001)
type WeaviateOptions struct {
	// Host — адрес Weaviate, например "localhost:8080" (обязательно).
	Host string
	// Scheme — протокол: "http" или "https". По умолчанию "http".
	Scheme string
	// Collection — имя коллекции (Weaviate class, обязательно).
	Collection string
	// APIKey — API ключ для Weaviate Cloud (опционально).
	APIKey string
	// Timeout — HTTP таймаут (по умолчанию 10s).
	Timeout time.Duration
}

// Validate проверяет корректность опций.
func (o WeaviateOptions) Validate() error {
	if o.Host == "" {
		return fmt.Errorf("host is required")
	}
	if o.Collection == "" {
		return fmt.Errorf("collection is required")
	}
	return nil
}

// scheme возвращает scheme с дефолтом "http".
func (o WeaviateOptions) scheme() string {
	if o.Scheme == "" {
		return "http"
	}
	return o.Scheme
}

// baseURL возвращает базовый URL вида scheme://host.
func (o WeaviateOptions) baseURL() string {
	return fmt.Sprintf("%s://%s", o.scheme(), o.Host)
}

// NewWeaviateStore создаёт Weaviate-backed реализацию VectorStore.
// При пустом opts.Host возвращает ErrInvalidVectorStoreConfig.
//
// @ds-task T3.1: NewWeaviateStore с валидацией и ErrInvalidVectorStoreConfig (AC-005, RQ-001)
func NewWeaviateStore(opts WeaviateOptions) (VectorStore, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidVectorStoreConfig, err)
	}
	return vectorstore.NewWeaviateStore(opts.scheme(), opts.Host, opts.Collection, opts.APIKey), nil
}

// CreateWeaviateCollection создаёт коллекцию (Weaviate class) со схемой для хранения чанков.
// Если коллекция уже существует — не возвращает ошибку (идемпотентно через 422).
//
// @ds-task T3.1: CreateWeaviateCollection → POST /v1/schema (RQ-007)
func CreateWeaviateCollection(ctx context.Context, opts WeaviateOptions) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	body := map[string]interface{}{
		"class":      opts.Collection,
		"vectorizer": "none",
		"properties": []map[string]interface{}{
			{"name": "chunkId", "dataType": []string{"text"}},
			{"name": "content", "dataType": []string{"text"}},
			{"name": "parentId", "dataType": []string{"text"}},
			{"name": "position", "dataType": []string{"int"}},
			{"name": "chunkMetadata", "dataType": []string{"text"}},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/v1/schema", opts.baseURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if opts.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("weaviate request: %w", err)
	}
	defer resp.Body.Close()

	// 200 — создана; 422 — коллекция уже существует (идемпотентно)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnprocessableEntity {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
}

// DeleteWeaviateCollection удаляет коллекцию из Weaviate.
// 404 считается успехом (идемпотентность).
//
// @ds-task T3.1: DeleteWeaviateCollection → DELETE /v1/schema/{class}
func DeleteWeaviateCollection(ctx context.Context, opts WeaviateOptions) error {
	if opts.Collection == "" {
		return fmt.Errorf("collection is required")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/v1/schema/%s", opts.baseURL(), opts.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if opts.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("weaviate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
}

// WeaviateCollectionExists проверяет существование коллекции в Weaviate.
// Возвращает true при статусе 200, false при 404.
//
// @ds-task T3.1: WeaviateCollectionExists → GET /v1/schema/{class}
func WeaviateCollectionExists(ctx context.Context, opts WeaviateOptions) (bool, error) {
	if opts.Collection == "" {
		return false, fmt.Errorf("collection is required")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	reqURL := fmt.Sprintf("%s/v1/schema/%s", opts.baseURL(), opts.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	if opts.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("weaviate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	b, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
}
