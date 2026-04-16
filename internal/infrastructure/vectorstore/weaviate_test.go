package vectorstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testWeaviateCollection = "TestChunks"

// weaviateSearchResponse формирует тестовый GraphQL-ответ Weaviate с одним чанком.
func weaviateSearchResponse(collection string, chunkID, content, parentID string, position int, metadata map[string]string, certainty float64) map[string]interface{} {
	metaJSON, _ := json.Marshal(metadata)
	return map[string]interface{}{
		"data": map[string]interface{}{
			"Get": map[string]interface{}{
				collection: []map[string]interface{}{
					{
						"chunkId":       chunkID,
						"content":       content,
						"parentId":      parentID,
						"position":      position,
						"chunkMetadata": string(metaJSON),
						"_additional": map[string]interface{}{
							"id":        uuidFromID(chunkID),
							"certainty": certainty,
						},
					},
				},
			},
		},
	}
}

// TestWeaviateUpsertSearch проверяет round-trip Upsert → Search (AC-001).
// @sk-task T4.1: TestWeaviateUpsertSearch (AC-001)
func TestWeaviateUpsertSearch(t *testing.T) {
	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "Go concurrency",
		ParentID:  "doc-1",
		Embedding: []float64{1, 0, 0},
		Position:  2,
		Metadata:  map[string]string{"category": "go"},
	}

	putCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v1/objects/"):
			// Первый PUT → 404, чтобы спровоцировать POST (DEC-005)
			if !putCalled {
				putCalled = true
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}

		case r.Method == http.MethodPost && r.URL.Path == "/v1/objects":
			// POST create
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": uuidFromID(chunk.ID)})

		case r.Method == http.MethodPost && r.URL.Path == "/v1/graphql":
			// GraphQL near-vector search
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateSearchResponse(
				testWeaviateCollection, chunk.ID, chunk.Content, chunk.ParentID,
				chunk.Position, chunk.Metadata, 0.95,
			))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	// Upsert
	err := store.Upsert(context.Background(), chunk)
	require.NoError(t, err)

	// Search
	result, err := store.Search(context.Background(), chunk.Embedding, 1)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	got := result.Chunks[0]
	assert.Equal(t, chunk.ID, got.Chunk.ID)
	assert.Equal(t, chunk.Content, got.Chunk.Content)
	assert.Equal(t, chunk.ParentID, got.Chunk.ParentID)
	assert.Equal(t, chunk.Position, got.Chunk.Position)
	assert.Equal(t, chunk.Metadata, got.Chunk.Metadata)
	assert.Greater(t, got.Score, 0.0, "Score должен быть > 0")
}

// TestWeaviateSearchWithFilter проверяет фильтрацию по parentId (AC-002).
// Тест убеждается, что WHERE-блок в GraphQL-запросе содержит поле parentId.
//
// @sk-task T4.1: TestWeaviateSearchWithFilter (AC-002)
func TestWeaviateSearchWithFilter(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateSearchResponse(
				testWeaviateCollection, "c1", "Go concurrency", "doc-1", 0, nil, 0.9,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	filter := domain.ParentIDFilter{ParentIDs: []string{"doc-1"}}
	result, err := store.SearchWithFilter(context.Background(), []float64{1, 0, 0}, 5, filter)
	require.NoError(t, err)
	require.NotEmpty(t, result.Chunks)

	// AC-002: WHERE-блок должен содержать parentId (AC-002)
	assert.Contains(t, capturedQuery, "parentId", "GraphQL запрос должен содержать WHERE по parentId")
	assert.Contains(t, capturedQuery, "doc-1", "GraphQL запрос должен содержать значение parentId")
}

// TestWeaviateSearchWithMetadataFilter проверяет фильтрацию по meta_* (AC-003).
// Тест убеждается, что WHERE-блок в GraphQL-запросе использует meta_-префикс.
//
// @sk-task T4.1: TestWeaviateSearchWithMetadataFilter (AC-003)
func TestWeaviateSearchWithMetadataFilter(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateSearchResponse(
				testWeaviateCollection, "c1", "Go channels", "doc-1", 0,
				map[string]string{"category": "go"}, 0.88,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	filter := domain.MetadataFilter{Fields: map[string]string{"category": "go"}}
	result, err := store.SearchWithMetadataFilter(context.Background(), []float64{1, 0, 0}, 5, filter)
	require.NoError(t, err)
	require.NotEmpty(t, result.Chunks)

	// AC-003: путь в WHERE должен содержать prefix meta_ (DEC-003)
	assert.Contains(t, capturedQuery, "meta_category", "GraphQL запрос должен использовать meta_-префикс для metadata filter")
	assert.Contains(t, capturedQuery, `"go"`, "GraphQL запрос должен содержать значение фильтра")
}

// TestWeaviateDeleteIdempotent проверяет идемпотентность Delete (AC-004).
// Оба вызова — для несуществующего и существующего ID — должны возвращать nil.
//
// @sk-task T4.1: TestWeaviateDeleteIdempotent (AC-004)
func TestWeaviateDeleteIdempotent(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/objects/") {
			callCount++
			if callCount == 1 {
				// Первый вызов: объект не существует → 404
				w.WriteHeader(http.StatusNotFound)
			} else {
				// Второй вызов: объект удалён → 204
				w.WriteHeader(http.StatusNoContent)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	// Первый Delete: объект не существует — должен вернуть nil (AC-004)
	err := store.Delete(context.Background(), "nonexistent-id")
	require.NoError(t, err, "Delete несуществующего объекта должен возвращать nil")

	// Второй Delete: объект существует — должен вернуть nil (AC-004)
	err = store.Delete(context.Background(), "existing-id")
	require.NoError(t, err, "Delete существующего объекта должен возвращать nil")
}

// TestWeaviateSearchEmpty проверяет поведение при пустой коллекции.
// @sk-task T4.1: Edge case — пустая коллекция возвращает пустой результат без ошибки
func TestWeaviateSearchEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"Get": map[string]interface{}{
						testWeaviateCollection: []interface{}{},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	result, err := store.Search(context.Background(), []float64{1, 0, 0}, 5)
	require.NoError(t, err)
	assert.Empty(t, result.Chunks, "пустая коллекция должна возвращать пустой результат")
}

// TestWeaviateUuidFromID проверяет детерминированность UUID v5 (DEC-002).
// @sk-task T4.1: Детерминированность UUID v5
func TestWeaviateUuidFromID(t *testing.T) {
	id := "my-chunk-id"
	uuid1 := uuidFromID(id)
	uuid2 := uuidFromID(id)
	assert.Equal(t, uuid1, uuid2, "uuidFromID должна возвращать одинаковый UUID для одного id")

	// Разные ID → разные UUID
	uuidOther := uuidFromID("other-id")
	assert.NotEqual(t, uuid1, uuidOther, "разные id должны давать разные UUID")

	// Формат UUID: 8-4-4-4-12
	parts := strings.Split(uuid1, "-")
	require.Len(t, parts, 5)
	assert.Len(t, parts[0], 8)
	assert.Len(t, parts[1], 4)
	assert.Len(t, parts[2], 4)
	assert.Len(t, parts[3], 4)
	assert.Len(t, parts[4], 12)
}

// weaviateHybridSearchResponse формирует тестовый GraphQL-ответ Weaviate hybrid search с одним чанком.
func weaviateHybridSearchResponse(collection string, chunkID, content, parentID string, position int, metadata map[string]string, score float64) map[string]interface{} {
	metaJSON, _ := json.Marshal(metadata)
	return map[string]interface{}{
		"data": map[string]interface{}{
			"Get": map[string]interface{}{
				collection: []map[string]interface{}{
					{
						"chunkId":       chunkID,
						"content":       content,
						"parentId":      parentID,
						"position":      position,
						"chunkMetadata": string(metaJSON),
						"_additional": map[string]interface{}{
							"id":    uuidFromID(chunkID),
							"score": score,
						},
					},
				},
			},
		},
	}
}

// weaviateTestHost извлекает host:port из URL httptest.Server для передачи в NewWeaviateStore.
func weaviateTestHost(rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") {
		return rawURL[len("http://"):]
	}
	if strings.HasPrefix(rawURL, "https://") {
		return rawURL[len("https://"):]
	}
	return rawURL
}

// TestWeaviateSearchHybridRRF проверяет SearchHybrid с RRF fusion (AC-002, AC-003).
// @sk-test T3.1: TestWeaviateSearchHybridRRF (AC-002, AC-003)
func TestWeaviateSearchHybridRRF(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateHybridSearchResponse(
				testWeaviateCollection, "c1", "Go concurrency patterns", "doc-1", 0,
				map[string]string{"category": "go"}, 0.95,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.HybridConfig{
		SemanticWeight: 0.7,
		UseRRF:         true,
		RRFK:           60,
		BMFinalK:       0,
	}

	result, err := store.SearchHybrid(context.Background(), "concurrency", []float64{1, 0, 0}, 5, config)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	// AC-002: GraphQL запрос должен содержать bm25 (через query) и nearVector (через vector)
	assert.Contains(t, capturedQuery, "concurrency", "GraphQL запрос должен содержать query для BM25")
	assert.Contains(t, capturedQuery, "vector", "GraphQL запрос должен содержать vector для nearVector")

	// AC-003: GraphQL запрос должен содержать fusionType: "RankedFusion" для RRF
	assert.Contains(t, capturedQuery, "fusionType", "GraphQL запрос должен содержать fusionType")
	assert.Contains(t, capturedQuery, "RankedFusion", "GraphQL запрос должен использовать RankedFusion для RRF")

	got := result.Chunks[0]
	assert.Equal(t, "c1", got.Chunk.ID)
	assert.Equal(t, "Go concurrency patterns", got.Chunk.Content)
	assert.Equal(t, "doc-1", got.Chunk.ParentID)
	assert.Equal(t, 0.95, got.Score, "Score должен быть fusion score от Weaviate")
}

// TestWeaviateSearchHybridWeighted проверяет SearchHybrid с weighted fusion (AC-002, AC-003).
// @sk-test T3.1: TestWeaviateSearchHybridWeighted (AC-002, AC-003)
func TestWeaviateSearchHybridWeighted(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateHybridSearchResponse(
				testWeaviateCollection, "c2", "Go channels", "doc-2", 1,
				map[string]string{"category": "go"}, 0.88,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.HybridConfig{
		SemanticWeight: 0.7,
		UseRRF:         false,
		RRFK:           60,
		BMFinalK:       0,
	}

	result, err := store.SearchHybrid(context.Background(), "channels", []float64{0, 1, 0}, 5, config)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	// AC-002: GraphQL запрос должен содержать bm25 (через query) и nearVector (через vector)
	assert.Contains(t, capturedQuery, "channels", "GraphQL запрос должен содержать query для BM25")
	assert.Contains(t, capturedQuery, "vector", "GraphQL запрос должен содержать vector для nearVector")

	// AC-003: GraphQL запрос должен содержать alpha для weighted fusion
	assert.Contains(t, capturedQuery, "alpha", "GraphQL запрос должен содержать alpha для weighted fusion")
	assert.Contains(t, capturedQuery, "0.7", "GraphQL запрос должен содержать SemanticWeight как alpha")

	got := result.Chunks[0]
	assert.Equal(t, "c2", got.Chunk.ID)
	assert.Equal(t, "Go channels", got.Chunk.Content)
	assert.Equal(t, 0.88, got.Score, "Score должен быть fusion score от Weaviate")
}

// TestWeaviateSearchHybridInvalidConfig проверяет валидацию HybridConfig (AC-005).
// @sk-test T3.1: TestWeaviateSearchHybridInvalidConfig (AC-005)
func TestWeaviateSearchHybridInvalidConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	// SemanticWeight вне диапазона [0,1]
	config := domain.HybridConfig{
		SemanticWeight: 1.5,
		UseRRF:         true,
		RRFK:           60,
		BMFinalK:       0,
	}

	_, err := store.SearchHybrid(context.Background(), "test", []float64{1, 0, 0}, 5, config)
	require.Error(t, err, "SearchHybrid должен возвращать ошибку при невалидной HybridConfig")
	assert.Contains(t, err.Error(), "SemanticWeight", "Ошибка должна указывать на SemanticWeight")

	// RRFK < 1
	config = domain.HybridConfig{
		SemanticWeight: 0.7,
		UseRRF:         true,
		RRFK:           0,
		BMFinalK:       0,
	}

	_, err = store.SearchHybrid(context.Background(), "test", []float64{1, 0, 0}, 5, config)
	require.Error(t, err, "SearchHybrid должен возвращать ошибку при невалидной HybridConfig")
	assert.Contains(t, err.Error(), "RRFK", "Ошибка должна указывать на RRFK")
}

// TestWeaviateSearchHybridGraphQLError проверяет обработку ошибок GraphQL API (AC-006).
// @sk-test T3.1: TestWeaviateSearchHybridGraphQLError (AC-006)
func TestWeaviateSearchHybridGraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			// Возвращаем GraphQL ошибку
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []map[string]string{
					{"message": "invalid query syntax"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	_, err := store.SearchHybrid(context.Background(), "test", []float64{1, 0, 0}, 5, config)
	require.Error(t, err, "SearchHybrid должен возвращать ошибку при GraphQL ошибке")
	assert.Contains(t, err.Error(), "graphql error", "Ошибка должна указывать на GraphQL ошибку")
	assert.Contains(t, err.Error(), "invalid query syntax", "Ошибка должна содержать сообщение от GraphQL")
}

// TestWeaviateSearchHybridHTTPError проверяет обработку HTTP ошибок (AC-006).
// @sk-test T3.1: TestWeaviateSearchHybridHTTPError (AC-006)
func TestWeaviateSearchHybridHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	_, err := store.SearchHybrid(context.Background(), "test", []float64{1, 0, 0}, 5, config)
	require.Error(t, err, "SearchHybrid должен возвращать ошибку при HTTP ошибке")
	assert.Contains(t, err.Error(), "weaviate error", "Ошибка должна указывать на Weaviate ошибку")
	assert.Contains(t, err.Error(), "500", "Ошибка должна содержать HTTP статус код")
}

// TestWeaviateSearchHybridEmptyQuery проверяет поведение при пустом query (AC-006).
// @sk-test T3.1: TestWeaviateSearchHybridEmptyQuery (AC-006)
func TestWeaviateSearchHybridEmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	_, err := store.SearchHybrid(context.Background(), "", []float64{1, 0, 0}, 5, config)
	require.Error(t, err, "SearchHybrid должен возвращать ошибку при пустом query")
	assert.Equal(t, domain.ErrEmptyQueryText, err, "Ошибка должна быть ErrEmptyQueryText")
}

// TestWeaviateSearchHybridEmpty проверяет поведение при пустых результатах.
// @sk-test T3.1: TestWeaviateSearchHybridEmpty (AC-002)
func TestWeaviateSearchHybridEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"Get": map[string]interface{}{
						testWeaviateCollection: []interface{}{},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	result, err := store.SearchHybrid(context.Background(), "test", []float64{1, 0, 0}, 5, config)
	require.NoError(t, err)
	assert.Empty(t, result.Chunks, "пустая коллекция должна возвращать пустой результат")
}

// TestWeaviateSearchHybridWithParentIDFilter проверяет фильтрацию по ParentID в hybrid search (AC-004).
// @sk-test T5.1: TestWeaviateSearchHybridWithParentIDFilter (AC-004)
func TestWeaviateSearchHybridWithParentIDFilter(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateHybridSearchResponse(
				testWeaviateCollection, "c1", "Go concurrency", "doc-1", 0,
				map[string]string{"category": "go"}, 0.92,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	filter := domain.ParentIDFilter{ParentIDs: []string{"doc-1"}}

	result, err := store.SearchHybridWithParentIDFilter(context.Background(), "concurrency", []float64{1, 0, 0}, 5, config, filter)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	// AC-004: GraphQL запрос должен содержать WHERE по parentId
	assert.Contains(t, capturedQuery, "parentId", "GraphQL запрос должен содержать WHERE по parentId")
	assert.Contains(t, capturedQuery, "doc-1", "GraphQL запрос должен содержать значение parentId")

	got := result.Chunks[0]
	assert.Equal(t, "c1", got.Chunk.ID)
	assert.Equal(t, "doc-1", got.Chunk.ParentID)
}

// TestWeaviateSearchHybridWithParentIDFilterEmpty проверяет делегирование в SearchHybrid при пустом фильтре (AC-004).
// @sk-test T5.1: TestWeaviateSearchHybridWithParentIDFilterEmpty (AC-004)
func TestWeaviateSearchHybridWithParentIDFilterEmpty(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateHybridSearchResponse(
				testWeaviateCollection, "c1", "Go concurrency", "doc-1", 0,
				map[string]string{"category": "go"}, 0.92,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	filter := domain.ParentIDFilter{ParentIDs: []string{}}

	result, err := store.SearchHybridWithParentIDFilter(context.Background(), "concurrency", []float64{1, 0, 0}, 5, config, filter)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	// AC-004: при пустом фильтре должен делегировать в SearchHybrid без WHERE
	assert.NotContains(t, capturedQuery, "where", "GraphQL запрос не должен содержать WHERE при пустом фильтре")
}

// TestWeaviateSearchHybridWithMetadataFilter проверяет фильтрацию по метаданным в hybrid search (AC-004).
// @sk-test T5.1: TestWeaviateSearchHybridWithMetadataFilter (AC-004)
func TestWeaviateSearchHybridWithMetadataFilter(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateHybridSearchResponse(
				testWeaviateCollection, "c2", "Go channels", "doc-2", 1,
				map[string]string{"category": "go"}, 0.89,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	filter := domain.MetadataFilter{Fields: map[string]string{"category": "go"}}

	result, err := store.SearchHybridWithMetadataFilter(context.Background(), "channels", []float64{0, 1, 0}, 5, config, filter)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	// AC-004: GraphQL запрос должен содержать WHERE по meta_* свойствам
	assert.Contains(t, capturedQuery, "meta_category", "GraphQL запрос должен использовать meta_-префикс для metadata filter")
	assert.Contains(t, capturedQuery, `"go"`, "GraphQL запрос должен содержать значение фильтра")

	got := result.Chunks[0]
	assert.Equal(t, "c2", got.Chunk.ID)
	assert.Equal(t, "go", got.Chunk.Metadata["category"])
}

// TestWeaviateSearchHybridWithMetadataFilterEmpty проверяет делегирование в SearchHybrid при пустом фильтре (AC-004).
// @sk-test T5.1: TestWeaviateSearchHybridWithMetadataFilterEmpty (AC-004)
func TestWeaviateSearchHybridWithMetadataFilterEmpty(t *testing.T) {
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/graphql" {
			var body struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedQuery = body.Query

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(weaviateHybridSearchResponse(
				testWeaviateCollection, "c2", "Go channels", "doc-2", 1,
				map[string]string{"category": "go"}, 0.89,
			))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	store := NewWeaviateStore("http", weaviateTestHost(server.URL), testWeaviateCollection, "")

	config := domain.DefaultHybridConfig()
	filter := domain.MetadataFilter{Fields: map[string]string{}}

	result, err := store.SearchHybridWithMetadataFilter(context.Background(), "channels", []float64{0, 1, 0}, 5, config, filter)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)

	// AC-004: при пустом фильтре должен делегировать в SearchHybrid без WHERE
	assert.NotContains(t, capturedQuery, "where", "GraphQL запрос не должен содержать WHERE при пустом фильтре")
}
