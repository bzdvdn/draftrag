package vectorstore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQdrantStore_DeleteByParentID_SendsFilter(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/collections/test_collection/points/delete" {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &capturedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","result":{"operation_id":1,"status":"completed"}}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)
	err := store.DeleteByParentID(context.Background(), "my-doc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Проверяем что отправлен фильтр по parent_id
	filter, ok := capturedBody["filter"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected filter in body, got: %v", capturedBody)
	}
	must, ok := filter["must"].([]interface{})
	if !ok || len(must) == 0 {
		t.Fatalf("expected must conditions, got: %v", filter)
	}
	cond := must[0].(map[string]interface{})
	if cond["key"] != "parent_id" {
		t.Fatalf("expected key=parent_id, got: %v", cond["key"])
	}
	match := cond["match"].(map[string]interface{})
	if match["value"] != "my-doc" {
		t.Fatalf("expected value=my-doc, got: %v", match["value"])
	}
}

func TestQdrantStore_DeleteByParentID_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "test_collection", 3)
	err := store.DeleteByParentID(context.Background(), "doc-1")
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestQdrantStore_DeleteByParentID_ContextCancelled(t *testing.T) {
	store := NewQdrantStore("http://localhost:19999", "c", 3)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.DeleteByParentID(ctx, "doc-1")
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestQdrantStore_DeleteByParentID_NilContextPanics(t *testing.T) {
	store := NewQdrantStore("http://localhost:19999", "c", 3)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	//nolint:staticcheck
	_ = store.DeleteByParentID(nil, "doc-1")
}
