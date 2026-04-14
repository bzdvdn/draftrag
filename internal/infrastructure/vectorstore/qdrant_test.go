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

// @sk-test T2.2: TestSearchHybrid с Query API Prefetch и Fusion.RRF (AC-002, AC-003)
func TestQdrantStore_SearchHybrid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/points/query", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		// Проверка тела запроса
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))

		// Проверка Prefetch структуры
		prefetch, ok := req["prefetch"].([]interface{})
		require.True(t, ok)
		require.Len(t, prefetch, 2)

		// Проверка Fusion.RRF
		query, ok := req["query"].(map[string]interface{})
		require.True(t, ok)
		fusion, ok := query["fusion"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "rrf", fusion["type"])

		// Ответ Query API
		resp := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"id":    "chunk1",
					"score": 0.92,
					"payload": map[string]interface{}{
						"id":        "chunk1",
						"content":   "hybrid result 1",
						"parent_id": "doc1",
						"position":  0,
					},
				},
				{
					"id":    "chunk2",
					"score": 0.85,
					"payload": map[string]interface{}{
						"id":        "chunk2",
						"content":   "hybrid result 2",
						"parent_id": "doc1",
						"position":  1,
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

	// Гибридный поиск
	query := "test query"
	embedding := []float64{0.1, 0.2, 0.3}
	config := domain.DefaultHybridConfig()
	result, err := store.SearchHybrid(context.Background(), query, embedding, 2, config)

	require.NoError(t, err)
	assert.Len(t, result.Chunks, 2)
	assert.Equal(t, "test query", result.QueryText)
	assert.Equal(t, 2, result.TotalFound)

	// Проверка первого результата
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "hybrid result 1", result.Chunks[0].Chunk.Content)
	assert.InDelta(t, 0.92, result.Chunks[0].Score, 0.001)
}

// @sk-test T2.2: TestSearchHybrid валидация HybridConfig (AC-005)
func TestQdrantStore_SearchHybrid_Validation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": []map[string]interface{}{},
			"status": "ok",
		})
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	t.Run("пустой query", func(t *testing.T) {
		embedding := []float64{0.1, 0.2, 0.3}
		config := domain.DefaultHybridConfig()
		_, err := store.SearchHybrid(context.Background(), "", embedding, 10, config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "query is empty")
	})

	t.Run("невалидная HybridConfig", func(t *testing.T) {
		embedding := []float64{0.1, 0.2, 0.3}
		config := domain.HybridConfig{
			SemanticWeight: 1.5, // Неверное значение (> 1.0)
			UseRRF:         true,
		}
		_, err := store.SearchHybrid(context.Background(), "test", embedding, 10, config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hybrid config")
	})
}

// @sk-test T2.2: TestSearchHybrid обработка ошибок Query API (AC-006)
func TestQdrantStore_SearchHybrid_APIErrors(t *testing.T) {
	t.Run("Query API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{
					"error": "Query execution failed",
				},
			})
		}))
		defer server.Close()

		store := NewQdrantStore(server.URL, "test_collection", 3)
		embedding := []float64{0.1, 0.2, 0.3}
		config := domain.DefaultHybridConfig()

		_, err := store.SearchHybrid(context.Background(), "test", embedding, 10, config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status=500")
	})

	t.Run("context timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		store := NewQdrantStore(server.URL, "test_collection", 3)
		embedding := []float64{0.1, 0.2, 0.3}
		config := domain.DefaultHybridConfig()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := store.SearchHybrid(ctx, "test", embedding, 10, config)
		require.Error(t, err)
	})
}

// @sk-test T4.1: TestSearchHybridWithParentIDFilter с фильтрацией по ParentID (AC-004)
func TestQdrantStore_SearchHybridWithParentIDFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/points/query", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))

		// Проверка фильтра в Prefetch
		prefetch, ok := req["prefetch"].([]interface{})
		require.True(t, ok)

		for _, p := range prefetch {
			pMap := p.(map[string]interface{})
			filter, ok := pMap["filter"].(map[string]interface{})
			require.True(t, ok)

			should, ok := filter["should"].([]interface{})
			require.True(t, ok)
			require.Len(t, should, 2)
		}

		// Ответ
		resp := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"id":    "chunk1",
					"score": 0.91,
					"payload": map[string]interface{}{
						"id":        "chunk1",
						"content":   "filtered result 1",
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

	query := "test query"
	embedding := []float64{0.1, 0.2, 0.3}
	config := domain.DefaultHybridConfig()
	result, err := store.SearchHybridWithParentIDFilter(context.Background(), query, embedding, 10, config, filter)

	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "doc1", result.Chunks[0].Chunk.ParentID)
}

// @sk-test T4.1: TestSearchHybridWithMetadataFilter с фильтрацией по метаданным (AC-004)
func TestQdrantStore_SearchHybridWithMetadataFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/collections/test_collection/points/query", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))

		// Проверка фильтра в Prefetch
		prefetch, ok := req["prefetch"].([]interface{})
		require.True(t, ok)

		for _, p := range prefetch {
			pMap := p.(map[string]interface{})
			filter, ok := pMap["filter"].(map[string]interface{})
			require.True(t, ok)

			must, ok := filter["must"].([]interface{})
			require.True(t, ok)
			require.Len(t, must, 2)
		}

		// Ответ
		resp := map[string]interface{}{
			"result": []map[string]interface{}{
				{
					"id":    "chunk1",
					"score": 0.89,
					"payload": map[string]interface{}{
						"id":              "chunk1",
						"content":         "metadata filtered result",
						"parent_id":       "doc1",
						"position":        0,
						"metadata.author": "Jane",
						"metadata.tag":    "research",
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
			"author": "Jane",
			"tag":    "research",
		},
	}

	query := "test query"
	embedding := []float64{0.1, 0.2, 0.3}
	config := domain.DefaultHybridConfig()
	result, err := store.SearchHybridWithMetadataFilter(context.Background(), query, embedding, 10, config, filter)

	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "Jane", result.Chunks[0].Chunk.Metadata["author"])
	assert.Equal(t, "research", result.Chunks[0].Chunk.Metadata["tag"])
}

// @sk-test T4.1: TestSearchHybridWithParentIDFilter пустой фильтр (AC-004)
func TestQdrantStore_SearchHybridWithParentIDFilter_EmptyFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// При пустом фильтре должен вызываться обычный SearchHybrid
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": []map[string]interface{}{},
			"status": "ok",
		})
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	filter := domain.ParentIDFilter{
		ParentIDs: []string{}, // Пустой фильтр
	}

	query := "test query"
	embedding := []float64{0.1, 0.2, 0.3}
	config := domain.DefaultHybridConfig()
	result, err := store.SearchHybridWithParentIDFilter(context.Background(), query, embedding, 10, config, filter)

	require.NoError(t, err)
	assert.Empty(t, result.Chunks)
}

// @sk-test T4.1: TestSearchHybridWithMetadataFilter пустой фильтр (AC-004)
func TestQdrantStore_SearchHybridWithMetadataFilter_EmptyFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": []map[string]interface{}{},
			"status": "ok",
		})
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)

	filter := domain.MetadataFilter{
		Fields: map[string]string{}, // Пустой фильтр
	}

	query := "test query"
	embedding := []float64{0.1, 0.2, 0.3}
	config := domain.DefaultHybridConfig()
	result, err := store.SearchHybridWithMetadataFilter(context.Background(), query, embedding, 10, config, filter)

	require.NoError(t, err)
	assert.Empty(t, result.Chunks)
}
