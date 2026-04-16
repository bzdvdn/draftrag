package vectorstore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChromaStore_DeleteByParentID_SendsWhereFilter(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/collections/test_col/delete" {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &capturedBody)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test_col", 3)
	err := store.DeleteByParentID(context.Background(), "parent-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	where, ok := capturedBody["where"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected where in body, got: %v", capturedBody)
	}
	if where["parent_id"] != "parent-42" {
		t.Fatalf("expected parent_id=parent-42 in where, got: %v", where)
	}
}

func TestChromaStore_DeleteByParentID_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer server.Close()

	store := NewChromaStore(server.URL, "test_col", 3)
	err := store.DeleteByParentID(context.Background(), "doc-1")
	if err == nil {
		t.Fatal("expected error on 400")
	}
}

func TestChromaStore_DeleteByParentID_ContextCancelled(t *testing.T) {
	store := NewChromaStore("http://localhost:19998", "c", 3)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := store.DeleteByParentID(ctx, "doc-1")
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestChromaStore_DeleteByParentID_NilContextPanics(t *testing.T) {
	store := NewChromaStore("http://localhost:19998", "c", 3)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil context")
		}
	}()
	//nolint:staticcheck
	_ = store.DeleteByParentID(nil, "doc-1")
}
