package draftrag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// @sk-task T3.4: Тест CreateCollection и DeleteCollection (AC-005)
func TestCreateCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection", r.URL.Path)
		require.Equal(t, http.MethodPut, r.Method)

		// Проверка тела запроса
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		vectors, ok := req["vectors"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(768), vectors["size"])
		assert.Equal(t, "Cosine", vectors["distance"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": true,
			"status": "ok",
			"time":   0.1,
		})
	}))
	defer server.Close()

	opts := QdrantOptions{
		URL:        server.URL,
		Collection: "test_collection",
		Dimension:  768,
	}

	err := CreateCollection(context.Background(), opts)
	require.NoError(t, err)
}

func TestDeleteCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection", r.URL.Path)
		require.Equal(t, http.MethodDelete, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": true,
			"status": "ok",
		})
	}))
	defer server.Close()

	opts := QdrantOptions{
		URL:        server.URL,
		Collection: "test_collection",
	}

	err := DeleteCollection(context.Background(), opts)
	require.NoError(t, err)
}

func TestDeleteCollection_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": map[string]interface{}{
				"error": "Collection test_collection not found",
			},
		})
	}))
	defer server.Close()

	opts := QdrantOptions{
		URL:        server.URL,
		Collection: "test_collection",
	}

	// 404 не должен возвращать ошибку — коллекция уже удалена
	err := DeleteCollection(context.Background(), opts)
	require.NoError(t, err)
}

func TestCollectionExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/exists", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": true,
			"status": "ok",
		})
	}))
	defer server.Close()

	opts := QdrantOptions{
		URL:        server.URL,
		Collection: "test_collection",
	}

	exists, err := CollectionExists(context.Background(), opts)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCollectionExists_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": false,
			"status": "ok",
		})
	}))
	defer server.Close()

	opts := QdrantOptions{
		URL:        server.URL,
		Collection: "nonexistent",
	}

	exists, err := CollectionExists(context.Background(), opts)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestQdrantStore_Validation(t *testing.T) {
	// Пустое имя коллекции
	_, err := NewQdrantStore(QdrantOptions{
		Collection: "",
		Dimension:  768,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name is required")

	// Неправильная размерность
	_, err = NewQdrantStore(QdrantOptions{
		Collection: "test",
		Dimension:  0,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dimension must be > 0")
}

func TestCreateCollection_Validation(t *testing.T) {
	err := CreateCollection(context.Background(), QdrantOptions{
		Collection: "",
		Dimension:  768,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection name is required")
}
