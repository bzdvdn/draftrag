package draftrag

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestWeaviateAuthInvalidKey проверяет, что неверный API key возвращает ошибку.
func TestWeaviateAuthInvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer invalid-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid API key"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
		APIKey:     "invalid-key",
		Timeout:    10 * time.Second,
	}
	err := CreateWeaviateCollection(context.Background(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

// TestWeaviateAuthMissingHeader проверяет, что отсутствие auth header возвращает ошибку.
func TestWeaviateAuthMissingHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "missing auth header"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
		APIKey:     "", // APIKey не установлен, auth header не будет отправлен
		Timeout:    10 * time.Second,
	}
	err := CreateWeaviateCollection(context.Background(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

// TestWeaviateError404 проверяет, что 404 обрабатывается корректно для WeaviateCollectionExists.
func TestWeaviateError404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "collection not found"})
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	exists, err := WeaviateCollectionExists(context.Background(), WeaviateOptions{Host: host, Collection: "missing"})
	require.NoError(t, err)
	assert.False(t, exists) // 404 для WeaviateCollectionExists считается нормой (false, не error)
}

// TestWeaviateError500 проверяет, что 500 возвращается как явная ошибка.
func TestWeaviateError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "internal server error"})
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	err := CreateWeaviateCollection(context.Background(), WeaviateOptions{Host: host, Collection: "chunks"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestWeaviateNetworkError проверяет, что network errors возвращаются явно.
func TestWeaviateNetworkError(t *testing.T) {
	// Используем несуществующий адрес для имитации network error
	opts := WeaviateOptions{
		Host:       "invalid-host-that-does-not-exist:9999",
		Collection: "chunks",
		Timeout:    1 * time.Second, // короткий timeout для быстрого теста
	}
	err := CreateWeaviateCollection(context.Background(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "weaviate request") // network error wrapped в weaviate request error
}

// TestWeaviateContextCancellation проверяет context cancellation.
func TestWeaviateContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Задержка для проверки таймаута
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
		Timeout:    10 * time.Second,
	}
	err := CreateWeaviateCollection(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestWeaviateContextCancellationDelete проверяет context cancellation для DeleteWeaviateCollection.
func TestWeaviateContextCancellationDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
		Timeout:    10 * time.Second,
	}
	err := DeleteWeaviateCollection(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestWeaviateContextCancellationExists проверяет context cancellation для WeaviateCollectionExists.
func TestWeaviateContextCancellationExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
		Timeout:    10 * time.Second,
	}
	_, err := WeaviateCollectionExists(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestDeleteWeaviateCollection проверяет удаление коллекции через mock-сервер.
func TestDeleteWeaviateCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
	}
	err := DeleteWeaviateCollection(context.Background(), opts)
	require.NoError(t, err)
}

// TestWeaviateCollectionExistsSuccess проверяет success case для WeaviateCollectionExists.
func TestWeaviateCollectionExistsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"class": "chunks"})
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	exists, err := WeaviateCollectionExists(context.Background(), WeaviateOptions{Host: host, Collection: "chunks"})
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestWeaviateCollectionExistsError проверяет error case для WeaviateCollectionExists.
func TestWeaviateCollectionExistsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	_, err := WeaviateCollectionExists(context.Background(), WeaviateOptions{Host: host, Collection: "chunks"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestDeleteWeaviateCollectionError проверяет error case для DeleteWeaviateCollection.
func TestDeleteWeaviateCollectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	opts := WeaviateOptions{
		Host:       host,
		Collection: "chunks",
	}
	err := DeleteWeaviateCollection(context.Background(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
