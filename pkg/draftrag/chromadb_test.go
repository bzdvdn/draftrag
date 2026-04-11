package draftrag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T3.1 Тест валидации ChromaDBOptions — AC-005
func TestChromaDBOptions_Validate(t *testing.T) {
	// Пустое имя коллекции
	err := ChromaDBOptions{
		Collection: "",
		Dimension:  768,
	}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name is required")

	// Неправильная размерность
	err = ChromaDBOptions{
		Collection: "test",
		Dimension:  0,
	}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dimension must be > 0")

	// Корректные опции
	err = ChromaDBOptions{
		Collection: "test",
		Dimension:  768,
	}.Validate()
	require.NoError(t, err)
}

// T3.1 Тест валидации в NewChromaDBStore — AC-005
func TestNewChromaDBStore_Validation(t *testing.T) {
	// Пустое имя коллекции
	_, err := NewChromaDBStore(ChromaDBOptions{
		Collection: "",
		Dimension:  768,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name is required")

	// Неправильная размерность
	_, err = NewChromaDBStore(ChromaDBOptions{
		Collection: "test",
		Dimension:  0,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dimension must be > 0")
}

// T3.2 Тест CreateCollection — AC-001
func TestChromaDBCreateCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/collections", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		// Проверка тела запроса
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		assert.Equal(t, "test_collection", req["name"])
		metadata, ok := req["metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(768), metadata["dimension"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "test-id",
			"name": "test_collection",
		})
	}))
	defer server.Close()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
		Dimension:  768,
	}

	err := CreateChromaDBCollection(context.Background(), opts)
	require.NoError(t, err)
}

// T3.2 Тест CreateCollection с валидацией
func TestChromaDBCreateCollection_Validation(t *testing.T) {
	err := CreateChromaDBCollection(context.Background(), ChromaDBOptions{
		Collection: "",
		Dimension:  768,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name is required")
}

// T3.3 Тест DeleteCollection — AC-002
func TestChromaDBDeleteCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/collections/test_collection", r.URL.Path)
		require.Equal(t, http.MethodDelete, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "test-id",
			"name": "test_collection",
		})
	}))
	defer server.Close()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
	}

	err := DeleteChromaDBCollection(context.Background(), opts)
	require.NoError(t, err)
}

// T3.3 Тест DeleteCollection с 404 (идемпотентность) — AC-002
func TestChromaDBDeleteCollection_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Collection test_collection not found",
		})
	}))
	defer server.Close()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
	}

	// 404 не должен возвращать ошибку — коллекция уже удалена
	err := DeleteChromaDBCollection(context.Background(), opts)
	require.NoError(t, err)
}

// T3.4 Тест CollectionExists (коллекция есть) — AC-003
func TestChromaDBCollectionExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/collections/test_collection", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "test-id",
			"name":     "test_collection",
			"metadata": map[string]interface{}{},
		})
	}))
	defer server.Close()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
	}

	exists, err := ChromaDBCollectionExists(context.Background(), opts)
	require.NoError(t, err)
	assert.True(t, exists)
}

// T3.4 Тест CollectionExists (коллекции нет) — AC-004
func TestChromaDBCollectionExists_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/collections/nonexistent", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Collection nonexistent not found",
		})
	}))
	defer server.Close()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "nonexistent",
	}

	exists, err := ChromaDBCollectionExists(context.Background(), opts)
	require.NoError(t, err)
	assert.False(t, exists)
}

// T3.5 Тест контекстной отмены — AC-006
func TestChromaDBCreateCollection_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Задержка для проверки таймаута
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
		Dimension:  768,
	}

	err := CreateChromaDBCollection(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// T3.5 Тест контекстной отмены для DeleteCollection — AC-006
func TestChromaDBDeleteCollection_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
	}

	err := DeleteChromaDBCollection(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// T3.5 Тест контекстной отмены для CollectionExists — AC-006
func TestChromaDBCollectionExists_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	opts := ChromaDBOptions{
		BaseURL:    server.URL,
		Collection: "test_collection",
	}

	_, err := ChromaDBCollectionExists(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
