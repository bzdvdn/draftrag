package vectorstore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// @sk-task T1.2: Базовый mock HTTP server для тестирования (DEC-001)
func setupMockQdrantServer(t *testing.T) (*httptest.Server, *QdrantStore) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Базовый handler, будет переопределён в тестах
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}))

	store := NewQdrantStore(server.URL, "test_collection", 3)
	return server, store
}

// @sk-task T3.1: Тест Search с mock server (AC-001)
func TestQdrantStore_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/points/search", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		// Проверка тела запроса
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, float64(2), req["limit"])
		assert.Equal(t, true, req["with_payload"])

		// Ответ Qdrant
		resp := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"id":    "chunk1",
					"score": 0.95,
					"payload": map[string]interface{}{
						"id":              "chunk1",
						"content":         "test content 1",
						"parent_id":       "doc1",
						"position":        0,
						"metadata.author": "John",
					},
				},
				{
					"id":    "chunk2",
					"score": 0.85,
					"payload": map[string]interface{}{
						"id":        "chunk2",
						"content":   "test content 2",
						"parent_id": "doc1",
						"position":  1,
					},
				},
			},
			"status": "ok",
			"time":   0.001,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	// Поиск
	embedding := []float64{0.1, 0.2, 0.3}
	result, err := store.Search(context.Background(), embedding, 2)

	require.NoError(t, err)
	assert.Len(t, result.Chunks, 2)
	assert.Equal(t, 2, result.TotalFound)

	// Проверка первого результата
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "test content 1", result.Chunks[0].Chunk.Content)
	assert.Equal(t, "doc1", result.Chunks[0].Chunk.ParentID)
	assert.Equal(t, 0, result.Chunks[0].Chunk.Position)
	assert.InDelta(t, 0.95, result.Chunks[0].Score, 0.001)
	assert.Equal(t, "John", result.Chunks[0].Chunk.Metadata["author"])

	// Проверка второго результата
	assert.Equal(t, "chunk2", result.Chunks[1].Chunk.ID)
	assert.Equal(t, 1, result.Chunks[1].Chunk.Position)
}

// @sk-task T3.2: Тест Upsert и Delete (AC-004)
func TestQdrantStore_UpsertDelete(t *testing.T) {
	var upsertCalled, deleteCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/test_collection/points":
			if r.Method == http.MethodPut {
				upsertCalled = true

				// Проверка тела запроса
				body, _ := io.ReadAll(r.Body)
				var req map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &req))

				points, ok := req["points"].([]interface{})
				require.True(t, ok)
				require.Len(t, points, 1)

				point := points[0].(map[string]interface{})
				assert.Equal(t, "chunk1", point["id"])

				payload := point["payload"].(map[string]interface{})
				assert.Equal(t, "test content", payload["content"])
				assert.Equal(t, "doc1", payload["parent_id"])
				assert.Equal(t, float64(0), payload["position"])
				assert.Equal(t, "value1", payload["metadata.key1"])

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
			}

		case "/collections/test_collection/points/delete":
			if r.Method == http.MethodPost {
				deleteCalled = true

				body, _ := io.ReadAll(r.Body)
				var req map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &req))

				points, ok := req["points"].([]interface{})
				require.True(t, ok)
				assert.Contains(t, points, "chunk1")

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
			}
		}
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	// Upsert
	chunk := domain.Chunk{
		ID:        "chunk1",
		Content:   "test content",
		ParentID:  "doc1",
		Embedding: []float64{0.1, 0.2, 0.3},
		Position:  0,
		Metadata: map[string]string{
			"key1": "value1",
		},
	}

	err := store.Upsert(context.Background(), chunk)
	require.NoError(t, err)
	assert.True(t, upsertCalled, "Upsert должен был вызвать API")

	// Delete
	err = store.Delete(context.Background(), "chunk1")
	require.NoError(t, err)
	assert.True(t, deleteCalled, "Delete должен был вызвать API")
}

// @sk-task T3.3: Тест SearchWithFilter (ParentID) (AC-002)
func TestQdrantStore_SearchWithParentIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/points/search", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))

		// Проверка фильтра
		filter, ok := req["filter"].(map[string]interface{})
		require.True(t, ok)

		should, ok := filter["should"].([]interface{})
		require.True(t, ok)
		require.Len(t, should, 2)

		// Проверка первого условия
		cond1 := should[0].(map[string]interface{})
		assert.Equal(t, "parent_id", cond1["key"])
		match1 := cond1["match"].(map[string]interface{})
		assert.Equal(t, "doc1", match1["value"])

		// Ответ
		resp := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"id":    "chunk1",
					"score": 0.9,
					"payload": map[string]interface{}{
						"id":        "chunk1",
						"content":   "content 1",
						"parent_id": "doc1",
						"position":  0,
					},
				},
			},
			"status": "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	filter := domain.ParentIDFilter{
		ParentIDs: []string{"doc1", "doc2"},
	}

	embedding := []float64{0.1, 0.2, 0.3}
	result, err := store.SearchWithFilter(context.Background(), embedding, 10, filter)

	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "doc1", result.Chunks[0].Chunk.ParentID)
}

// @sk-task T3.3: Тест SearchWithMetadataFilter (AC-003)
func TestQdrantStore_SearchWithMetadataFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/points/search", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))

		// Проверка фильтра
		filter, ok := req["filter"].(map[string]interface{})
		require.True(t, ok)

		must, ok := filter["must"].([]interface{})
		require.True(t, ok)
		require.Len(t, must, 2)

		// Проверка условий (порядок может отличаться)
		keys := make(map[string]string)
		for _, c := range must {
			cond := c.(map[string]interface{})
			key := cond["key"].(string)
			match := cond["match"].(map[string]interface{})
			keys[key] = match["value"].(string)
		}

		assert.Equal(t, "John", keys["metadata.author"])
		assert.Equal(t, "important", keys["metadata.tag"])

		// Ответ
		resp := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"id":    "chunk1",
					"score": 0.88,
					"payload": map[string]interface{}{
						"id":              "chunk1",
						"content":         "content with metadata",
						"parent_id":       "doc1",
						"position":        0,
						"metadata.author": "John",
						"metadata.tag":    "important",
					},
				},
			},
			"status": "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	filter := domain.MetadataFilter{
		Fields: map[string]string{
			"author": "John",
			"tag":    "important",
		},
	}

	embedding := []float64{0.1, 0.2, 0.3}
	result, err := store.SearchWithMetadataFilter(context.Background(), embedding, 10, filter)

	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "John", result.Chunks[0].Chunk.Metadata["author"])
	assert.Equal(t, "important", result.Chunks[0].Chunk.Metadata["tag"])
}

// @sk-task T3.3: Тест обработки ошибок API (AC-006)
func TestQdrantStore_APIErrors(t *testing.T) {
	t.Run("collection not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"error": "Collection test_collection not found",
				},
			})
		}))
		defer server.Close()

		store := NewQdrantStore(server.URL, "test_collection", 3)
		embedding := []float64{0.1, 0.2, 0.3}

		_, err := store.Search(context.Background(), embedding, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status=404")
	})

	t.Run("bad request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"error": "Wrong input: Vector dimension 5 does not match",
				},
			})
		}))
		defer server.Close()

		store := NewQdrantStore(server.URL, "test_collection", 3)
		// Неправильная размерность вектора
		embedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

		_, err := store.Search(context.Background(), embedding, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dimension mismatch")
	})
}

func TestQdrantStore_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"result": []map[string]interface{}{},
			"status": "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)
	embedding := []float64{0.1, 0.2, 0.3}

	result, err := store.Search(context.Background(), embedding, 10)
	require.NoError(t, err)
	assert.Empty(t, result.Chunks)
	assert.Equal(t, 0, result.TotalFound)
}

func TestQdrantStore_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	embedding := []float64{0.1, 0.2, 0.3}
	_, err := store.Search(ctx, embedding, 10)
	require.Error(t, err)
}
