package draftrag

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWeaviateStore_InvalidConfig проверяет, что пустой Host возвращает ErrInvalidVectorStoreConfig.
// @sk-task T4.1: TestWeaviateNewStore_InvalidConfig (AC-005)
func TestNewWeaviateStore_InvalidConfig(t *testing.T) {
	// Пустой Host → ErrInvalidVectorStoreConfig (AC-005)
	_, err := NewWeaviateStore(WeaviateOptions{
		Host:       "",
		Collection: "chunks",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidVectorStoreConfig), "ожидался ErrInvalidVectorStoreConfig, получен: %v", err)

	// Пустой Collection → тоже ошибка конфигурации
	_, err = NewWeaviateStore(WeaviateOptions{
		Host:       "localhost:8080",
		Collection: "",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidVectorStoreConfig))

	// Корректные опции → нет ошибки (go build ./... проходит — AC-005)
	store, err := NewWeaviateStore(WeaviateOptions{
		Host:       "localhost:8080",
		Collection: "chunks",
	})
	require.NoError(t, err)
	assert.NotNil(t, store)
}

// TestCreateWeaviateCollection проверяет создание коллекции через mock-сервер.
// @sk-task T4.1: CreateWeaviateCollection (RQ-007)
func TestCreateWeaviateCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/schema", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "chunks", req["class"])
		assert.Equal(t, "none", req["vectorizer"])

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"class": "chunks"})
	}))
	defer server.Close()

	opts := WeaviateOptions{
		Host:       server.URL[len("http://"):],
		Collection: "chunks",
	}
	err := CreateWeaviateCollection(context.Background(), opts)
	require.NoError(t, err)
}

// TestWeaviateCollectionExists проверяет проверку существования коллекции.
// @sk-task T4.1: WeaviateCollectionExists
func TestWeaviateCollectionExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		if r.URL.Path == "/v1/schema/chunks" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"class": "chunks"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	exists, err := WeaviateCollectionExists(context.Background(), WeaviateOptions{Host: host, Collection: "chunks"})
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = WeaviateCollectionExists(context.Background(), WeaviateOptions{Host: host, Collection: "missing"})
	require.NoError(t, err)
	assert.False(t, exists)
}
