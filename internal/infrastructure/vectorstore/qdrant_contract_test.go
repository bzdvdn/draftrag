package vectorstore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// qdrantMock имитирует Qdrant REST API для contract-тестов.
type qdrantMock struct {
	mu     sync.Mutex
	points map[string]qdrantPoint
}

func newQdrantMock() *qdrantMock {
	return &qdrantMock{points: make(map[string]qdrantPoint)}
}

func (m *qdrantMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := r.URL.Path
	collection := extractQdrantCollection(path)
	if collection == "" {
		http.Error(w, `{"status":{"error":"invalid collection"}}`, http.StatusBadRequest)
		return
	}

	switch {
	case strings.HasSuffix(path, "/points") && r.Method == http.MethodPut:
		m.handleUpsert(w, r)
	case strings.HasSuffix(path, "/points/delete") && r.Method == http.MethodPost:
		m.handleDelete(w, r)
	case strings.HasSuffix(path, "/points/search") && r.Method == http.MethodPost:
		m.handleSearch(w, r)
	default:
		http.Error(w, `{"status":{"error":"not found"}}`, http.StatusNotFound)
	}
}

func (m *qdrantMock) handleUpsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Points []struct {
			ID      string                 `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"status":{"error":"%s"}}`, err.Error()), http.StatusBadRequest)
		return
	}
	for _, p := range req.Points {
		m.points[p.ID] = qdrantPoint{
			ID:      p.ID,
			Score:   0,
			Payload: p.Payload,
		}
	}
	writeJSON(w, map[string]interface{}{
		"result": map[string]interface{}{
			"operation_id": 1,
			"status":       "completed",
		},
	})
}

func (m *qdrantMock) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Points []string `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"status":{"error":"%s"}}`, err.Error()), http.StatusBadRequest)
		return
	}
	for _, id := range req.Points {
		delete(m.points, id)
	}
	writeJSON(w, map[string]interface{}{
		"result": map[string]interface{}{
			"operation_id": 2,
			"status":       "completed",
		},
	})
}

func (m *qdrantMock) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Vector      []float64 `json:"vector"`
		Limit       int       `json:"limit"`
		WithPayload bool      `json:"with_payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"status":{"error":"%s"}}`, err.Error()), http.StatusBadRequest)
		return
	}

	var results []qdrantPoint
	for _, p := range m.points {
		results = append(results, qdrantPoint{
			ID:      p.ID,
			Score:   0.9,
			Payload: p.Payload,
		})
	}
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}
	writeJSON(w, qdrantSearchResponse{Result: results, Status: "ok"})
}

func extractQdrantCollection(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "collections" {
		return parts[1]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// @sk-test contract-tests-stores#T4.1: QdrantStore HTTP mock prototype
func TestContract_QdrantMock(t *testing.T) {
	mock := newQdrantMock()
	srv := httptest.NewServer(mock)
	defer srv.Close()

	t.Run("VectorStore/qdrant", func(t *testing.T) {
		runVectorStoreContract(t, func() domain.VectorStore {
			mock.mu.Lock()
			mock.points = make(map[string]qdrantPoint)
			mock.mu.Unlock()
			return NewQdrantStore(srv.URL, "test-collection", 3)
		})
	})
}
