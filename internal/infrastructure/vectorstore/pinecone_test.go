// @sk-test prod-issues#T3.2: Pinecone VectorStore unit tests (AC-007)

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

func newPineconeTestServer(handler http.HandlerFunc) (*httptest.Server, *PineconeStore) {
	srv := httptest.NewServer(handler)

	store := &PineconeStore{
		apiKey:    "test-key",
		indexName: "test-index",
		dimension: 3,
		host:      "http://" + srv.Listener.Addr().String(),
		client:    srv.Client(),
	}
	store.client.Timeout = 5 * time.Second
	return srv, store
}

func pineconeHandler(t *testing.T, expectedMethod, expectedPath string, responseCode int, responseBody interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedMethod, r.Method)
		assert.Contains(t, r.URL.Path, expectedPath)
		assert.Equal(t, "test-key", r.Header.Get("Api-Key"))

		if responseBody != nil {
			w.WriteHeader(responseCode)
			err := json.NewEncoder(w).Encode(responseBody)
			assert.NoError(t, err)
		} else {
			w.WriteHeader(responseCode)
		}
	}
}

func TestPineconeStore_Upsert(t *testing.T) {
	srv, store := newPineconeTestServer(
		pineconeHandler(t, http.MethodPost, "/vectors/upsert", http.StatusOK, nil),
	)
	defer srv.Close()

	err := store.Upsert(context.Background(), domain.Chunk{
		ID:      "chunk1",
		Content: "test content",
		Embedding: []float64{0.1, 0.2, 0.3},
		Metadata: map[string]string{"key": "val"},
	})
	assert.NoError(t, err)
}

func TestPineconeStore_Delete(t *testing.T) {
	srv, store := newPineconeTestServer(
		pineconeHandler(t, http.MethodPost, "/vectors/delete", http.StatusOK, nil),
	)
	defer srv.Close()

	err := store.Delete(context.Background(), "chunk1")
	assert.NoError(t, err)
}

func TestPineconeStore_DeleteEmptyID(t *testing.T) {
	_, store := newPineconeTestServer(nil)

	err := store.Delete(context.Background(), "")
	assert.Error(t, err)
}

func TestPineconeStore_Search(t *testing.T) {
	resp := pineconeQueryResponse{
		Matches: []pineconeMatch{
			{
				ID:    "chunk1",
				Score: 0.95,
				Metadata: map[string]interface{}{
					"content":   "search result",
					"parent_id": "doc1",
					"position":  float64(1),
				},
			},
		},
	}

	srv, store := newPineconeTestServer(
		pineconeHandler(t, http.MethodPost, "/vectors/query", http.StatusOK, resp),
	)
	defer srv.Close()

	result, err := store.Search(context.Background(), []float64{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, result.Chunks, 1)
	assert.Equal(t, "chunk1", result.Chunks[0].Chunk.ID)
	assert.Equal(t, "search result", result.Chunks[0].Chunk.Content)
	assert.Equal(t, "doc1", result.Chunks[0].Chunk.ParentID)
	assert.Equal(t, 1, result.Chunks[0].Chunk.Position)
	assert.Equal(t, 0.95, result.Chunks[0].Score)
}

func TestPineconeStore_SearchEmptyEmbedding(t *testing.T) {
	_, store := newPineconeTestServer(nil)

	_, err := store.Search(context.Background(), []float64{}, 5)
	assert.Error(t, err)
}

func TestPineconeStore_Health(t *testing.T) {
	resp := pineconeStatsResponse{
		Dimension:        3,
		TotalVectorCount: 100,
	}

	srv, store := newPineconeTestServer(
		pineconeHandler(t, http.MethodPost, "/describe_index_stats", http.StatusOK, resp),
	)
	defer srv.Close()

	err := store.Health(context.Background())
	assert.NoError(t, err)
}

func TestPineconeStore_Close(t *testing.T) {
	_, store := newPineconeTestServer(nil)
	err := store.Close()
	assert.NoError(t, err)
}

func TestPineconeStore_NewPineconeStore_Validation(t *testing.T) {
	_, err := NewPineconeStore(PineconeOptions{
		APIKey:    "",
		IndexName: "test",
		Dimension: 3,
	})
	assert.Error(t, err)

	_, err = NewPineconeStore(PineconeOptions{
		APIKey:    "key",
		IndexName: "",
		Dimension: 3,
	})
	assert.Error(t, err)

	_, err = NewPineconeStore(PineconeOptions{
		APIKey:    "key",
		IndexName: "test",
		Dimension: 0,
	})
	assert.Error(t, err)

	store, err := NewPineconeStore(PineconeOptions{
		APIKey:    "key",
		IndexName: "test",
		Dimension: 3,
	})
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestPineconeStore_UpsertEmptyID(t *testing.T) {
	_, store := newPineconeTestServer(nil)

	err := store.Upsert(context.Background(), domain.Chunk{
		ID: "",
	})
	assert.Error(t, err)
}

func TestPineconeStore_DeleteByParentID(t *testing.T) {
	_, store := newPineconeTestServer(nil)

	err := store.DeleteByParentID(context.Background(), "parent1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
