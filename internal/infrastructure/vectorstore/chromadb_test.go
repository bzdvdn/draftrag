package vectorstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// @sk-task T3.1: Тест Upsert — сохранение чанка с embedding и metadata (AC-001)
func TestChromaStore_Upsert(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/collections/test-collection/upsert", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	chunk := domain.Chunk{
		ID:        "chunk-1",
		Content:   "test content",
		ParentID:  "doc-1",
		Position:  0,
		Embedding: []float64{0.1, 0.2, 0.3},
		Metadata: map[string]string{
			"source": "file1.txt",
		},
	}

	err := store.Upsert(context.Background(), chunk)
	require.NoError(t, err)

	// Проверяем структуру запроса
	ids, ok := receivedRequest["ids"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "chunk-1", ids[0])

	metadatas, ok := receivedRequest["metadatas"].([]interface{})
	require.True(t, ok)
	meta := metadatas[0].(map[string]interface{})
	assert.Equal(t, "doc-1", meta["parent_id"])
	assert.Equal(t, "file1.txt", meta["source"])
	assert.Equal(t, "test content", meta["content"])
}

// @sk-task T3.1: Тест Search — поиск с возвратом результатов и score (AC-002)
func TestChromaStore_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/collections/test-collection/query", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		response := map[string]interface{}{
			"ids":        [][]string{{"chunk-1", "chunk-2"}},
			"distances":  [][]float64{{0.1, 0.3}},
			"metadatas":  [][]map[string]string{{{"parent_id": "doc-1", "content": "content 1"}, {"parent_id": "doc-1", "content": "content 2"}}},
			"documents":  [][]string{{"content 1", "content 2"}},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	embedding := []float64{0.1, 0.2, 0.3}
	result, err := store.Search(context.Background(), embedding, 2)
	require.NoError(t, err)

	assert.Len(t, result.Chunks, 2)
	assert.Equal(t, 2, result.TotalFound)

	// Проверяем score: 1 - distance
	assert.InDelta(t, 0.9, result.Chunks[0].Score, 0.001) // 1 - 0.1
	assert.InDelta(t, 0.7, result.Chunks[1].Score, 0.001) // 1 - 0.3

	assert.Equal(t, "chunk-1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "content 1", result.Chunks[0].Chunk.Content)
}

// @sk-task T3.1: Тест SearchWithMetadataFilter — фильтрация по metadata (AC-003)
func TestChromaStore_SearchWithMetadataFilter(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		require.NoError(t, err)

		response := map[string]interface{}{
			"ids":        [][]string{{"chunk-1"}},
			"distances":  [][]float64{{0.2}},
			"metadatas":  [][]map[string]string{{{"parent_id": "doc-1", "source": "file1.txt"}}},
			"documents":  [][]string{{"filtered content"}},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	embedding := []float64{0.1, 0.2, 0.3}
	filter := domain.MetadataFilter{
		Fields: map[string]string{"source": "file1.txt"},
	}

	result, err := store.SearchWithMetadataFilter(context.Background(), embedding, 5, filter)
	require.NoError(t, err)

	assert.Len(t, result.Chunks, 1)

	// Проверяем where-фильтр в запросе
	where, ok := receivedRequest["where"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "file1.txt", where["source"])
}

// @sk-task T3.2: Тест Delete — удаление чанка по ID (AC-004)
func TestChromaStore_Delete(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/collections/test-collection/delete", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	err := store.Delete(context.Background(), "chunk-to-delete")
	require.NoError(t, err)

	ids, ok := receivedRequest["ids"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, "chunk-to-delete", ids[0])
}

// @sk-task T3.2: Тест валидации размерности эмбеддинга (AC-005)
func TestChromaStore_DimensionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	// Upsert с неверной размерностью
	chunk := domain.Chunk{
		ID:        "chunk-1",
		Content:   "content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2}, // размерность 2 вместо 3
	}

	err := store.Upsert(context.Background(), chunk)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmbeddingDimensionMismatch)

	// Search с неверной размерностью
	_, err = store.Search(context.Background(), []float64{0.1, 0.2}, 5)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmbeddingDimensionMismatch)
}

// @sk-task T3.2: Тест автосоздания коллекции при отсутствии (AC-007)
func TestChromaStore_AutocreateCollection(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		switch r.URL.Path {
		case "/api/v1/collections/test-collection/query":
			if callCount == 1 {
				// Первый вызов — 404
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Collection not found",
				})
				return
			}
			// Второй вызов после создания — успех
			response := map[string]interface{}{
				"ids":       [][]string{{"chunk-1"}},
				"distances": [][]float64{{0.1}},
				"metadatas": [][]map[string]string{{{"parent_id": "doc-1"}}},
				"documents": [][]string{{"content"}},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)

		case "/api/v1/collections":
			// Создание коллекции
			assert.Equal(t, http.MethodPost, r.Method)
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "test-collection", body["name"])

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"status": "created"})

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	embedding := []float64{0.1, 0.2, 0.3}
	filter := domain.MetadataFilter{Fields: map[string]string{"key": "value"}}

	result, err := store.SearchWithMetadataFilter(context.Background(), embedding, 5, filter)
	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)

	// Должно быть 3 вызова: query (404), create, query (retry)
	assert.Equal(t, 3, callCount)
}

// @sk-task T3.3: Тест cancellation через context (AC-006)
func TestChromaStore_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Имитируем медленный ответ
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	// Контекст с очень коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Ждём чтобы таймаут точно истёк
	time.Sleep(5 * time.Millisecond)

	chunk := domain.Chunk{
		ID:        "chunk-1",
		Content:   "content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
	}

	err := store.Upsert(ctx, chunk)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// @sk-task T3.3: Тест nil context panic (consistent с другими реализациями)
func TestChromaStore_NilContext(t *testing.T) {
	store := NewChromaStore("http://localhost:8000", "test", 3)

	assert.Panics(t, func() {
		store.Upsert(nil, domain.Chunk{})
	})

	assert.Panics(t, func() {
		store.Search(nil, []float64{0.1}, 5)
	})

	assert.Panics(t, func() {
		store.Delete(nil, "id")
	})
}

// @sk-task T3.1: Тест Search с пустой коллекцией
func TestChromaStore_SearchEmptyCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ids":       [][]string{{}},
			"distances": [][]float64{{}},
			"metadatas": [][]map[string]string{{}},
			"documents": [][]string{{}},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	result, err := store.Search(context.Background(), []float64{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Empty(t, result.Chunks)
	assert.Equal(t, 0, result.TotalFound)
}

// @sk-task T3.2: Тест SearchWithFilter с пустым ParentIDFilter
func TestChromaStore_SearchWithFilter_EmptyFilter(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Должен вызвать обычный Search без where-фильтра

		response := map[string]interface{}{
			"ids":       [][]string{{"chunk-1"}},
			"distances": [][]float64{{0.1}},
			"metadatas": [][]map[string]string{{{"parent_id": "doc-1"}}},
			"documents": [][]string{{"content"}},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	// Пустой filter — должен вызвать Search
	emptyFilter := domain.ParentIDFilter{ParentIDs: []string{}}
	result, err := store.SearchWithFilter(context.Background(), []float64{0.1, 0.2, 0.3}, 5, emptyFilter)
	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)
}

// @sk-task T3.2: Тест SearchWithMetadataFilter с пустым фильтром
func TestChromaStore_SearchWithMetadataFilter_EmptyFilter(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		// Не должен содержать where при пустом фильтре
		_, hasWhere := req["where"]
		assert.False(t, hasWhere, "where should not be present with empty filter")

		response := map[string]interface{}{
			"ids":       [][]string{{"chunk-1"}},
			"distances": [][]float64{{0.1}},
			"metadatas": [][]map[string]string{{{"parent_id": "doc-1"}}},
			"documents": [][]string{{"content"}},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	// Пустой filter.Fields — должен вызвать Search
	emptyFilter := domain.MetadataFilter{Fields: nil}
	result, err := store.SearchWithMetadataFilter(context.Background(), []float64{0.1, 0.2, 0.3}, 5, emptyFilter)
	require.NoError(t, err)
	assert.Len(t, result.Chunks, 1)
}

// @sk-task T3.2: Тест Delete с пустым ID
func TestChromaStore_DeleteEmptyID(t *testing.T) {
	store := NewChromaStore("http://localhost:8000", "test-collection", 3)

	err := store.Delete(context.Background(), "")
	assert.ErrorIs(t, err, domain.ErrEmptyChunkID)
}

// @sk-task T3.2: Тест SearchWithFilter с OR по нескольким ParentID
func TestChromaStore_SearchWithFilter_MultipleParentIDs(t *testing.T) {
	var receivedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		require.NoError(t, err)

		response := map[string]interface{}{
			"ids":       [][]string{{"chunk-1", "chunk-2"}},
			"distances": [][]float64{{0.1, 0.2}},
			"metadatas": [][]map[string]string{{{"parent_id": "doc-1"}, {"parent_id": "doc-2"}}},
			"documents": [][]string{{"content 1", "content 2"}},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	filter := domain.ParentIDFilter{ParentIDs: []string{"doc-1", "doc-2"}}
	result, err := store.SearchWithFilter(context.Background(), []float64{0.1, 0.2, 0.3}, 5, filter)
	require.NoError(t, err)
	assert.Len(t, result.Chunks, 2)

	// Проверяем where-фильтр с $or
	where, ok := receivedRequest["where"].(map[string]interface{})
	require.True(t, ok)
	orConditions, ok := where["$or"].([]interface{})
	require.True(t, ok)
	assert.Len(t, orConditions, 2)
}

// @sk-task T3.2: Тест Search с невалидным topK
func TestChromaStore_InvalidTopK(t *testing.T) {
	store := NewChromaStore("http://localhost:8000", "test", 3)

	_, err := store.Search(context.Background(), []float64{0.1, 0.2, 0.3}, 0)
	assert.ErrorIs(t, err, domain.ErrInvalidQueryTopK)

	_, err = store.Search(context.Background(), []float64{0.1, 0.2, 0.3}, -1)
	assert.ErrorIs(t, err, domain.ErrInvalidQueryTopK)
}

// @sk-task T3.3: Тест обработки HTTP ошибок
func TestChromaStore_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "internal error",
		})
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test-collection", 3)

	err := store.Upsert(context.Background(), domain.Chunk{
		ID:        "chunk-1",
		Content:   "content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status=500")
}

// @sk-task T3.3: Тест валидации чанка перед upsert
func TestChromaStore_UpsertInvalidChunk(t *testing.T) {
	store := NewChromaStore("http://localhost:8000", "test-collection", 3)

	// Чанк без ID
	err := store.Upsert(context.Background(), domain.Chunk{
		ID:        "",
		Content:   "content",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmptyChunkID)

	// Чанк без Content
	err = store.Upsert(context.Background(), domain.Chunk{
		ID:        "chunk-1",
		Content:   "",
		ParentID:  "doc-1",
		Embedding: []float64{0.1, 0.2, 0.3},
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmptyChunkContent)

	// Чанк без ParentID
	err = store.Upsert(context.Background(), domain.Chunk{
		ID:        "chunk-1",
		Content:   "content",
		ParentID:  "",
		Embedding: []float64{0.1, 0.2, 0.3},
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmptyChunkParentID)
}

// @sk-task T3.3: Тест NewChromaStore с defaults
func TestNewChromaStore_Defaults(t *testing.T) {
	store := NewChromaStore("", "my-collection", 384)

	assert.Equal(t, "http://localhost:8000", store.baseURL)
	assert.Equal(t, "my-collection", store.collection)
	assert.Equal(t, 384, store.dimension)
	assert.NotNil(t, store.client)
}
