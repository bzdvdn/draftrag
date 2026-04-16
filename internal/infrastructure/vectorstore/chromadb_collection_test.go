package vectorstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChromaCreateCollection_Idempotent проверяет, что CreateCollection возвращает nil
// при 200 и 201 (idempotent через get_or_create=true).
// @sk-task T3.1: тест AC-001
func TestChromaCreateCollection_Idempotent(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, "/api/v1/collections") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(status)
			}))
			defer srv.Close()

			store := NewChromaStore(srv.URL, "docs", 3)
			if err := store.CreateCollection(context.Background()); err != nil {
				t.Errorf("CreateCollection() status=%d: unexpected error: %v", status, err)
			}
		})
	}
}

// TestChromaCreateCollection_HTTPError проверяет, что CreateCollection возвращает ошибку при HTTP 5xx.
// @sk-task T3.1: тест AC-001 (error path)
func TestChromaCreateCollection_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "internal error")
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	err := store.CreateCollection(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка при HTTP 500, получен nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("ошибка должна содержать статус, got: %v", err)
	}
}

// TestChromaDeleteCollection_HappyPath проверяет, что DeleteCollection отправляет DELETE
// и возвращает nil при HTTP 200.
// @sk-task T3.1: тест AC-002
func TestChromaDeleteCollection_HappyPath(t *testing.T) {
	var capturedMethod, capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	if err := store.DeleteCollection(context.Background()); err != nil {
		t.Fatalf("DeleteCollection() error: %v", err)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("метод = %s, want DELETE", capturedMethod)
	}
	if capturedPath != "/api/v1/collections/docs" {
		t.Errorf("путь = %s, want /api/v1/collections/docs", capturedPath)
	}
}

// TestChromaDeleteCollection_Idempotent404 проверяет, что DeleteCollection возвращает nil при 404.
// @sk-task T3.1: тест AC-003 (404 idempotent)
func TestChromaDeleteCollection_Idempotent404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	if err := store.DeleteCollection(context.Background()); err != nil {
		t.Errorf("DeleteCollection() при 404: ожидался nil, got: %v", err)
	}
}

// TestChromaDeleteCollection_HTTP5xx проверяет, что DeleteCollection возвращает ошибку при 5xx.
// @sk-task T3.1: тест AC-003 (5xx error)
func TestChromaDeleteCollection_HTTP5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "server error")
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	err := store.DeleteCollection(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка при HTTP 500, получен nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("ошибка должна содержать статус 500, got: %v", err)
	}
}

// TestChromaCollectionExists_True проверяет, что CollectionExists возвращает (true, nil) при HTTP 200.
// @sk-task T3.1: тест AC-004
func TestChromaCollectionExists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/collections/docs" {
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	exists, err := store.CollectionExists(context.Background())
	if err != nil {
		t.Fatalf("CollectionExists() error: %v", err)
	}
	if !exists {
		t.Error("CollectionExists() = false, want true")
	}
}

// TestChromaCollectionExists_False проверяет, что CollectionExists возвращает (false, nil) при 404.
// @sk-task T3.1: тест AC-005
func TestChromaCollectionExists_False(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	exists, err := store.CollectionExists(context.Background())
	if err != nil {
		t.Errorf("CollectionExists() при 404: ожидался nil error, got: %v", err)
	}
	if exists {
		t.Error("CollectionExists() = true, want false при 404")
	}
}

// TestChromaCollectionExists_ServerError проверяет, что CollectionExists возвращает (false, error) при 5xx.
// @sk-task T3.1: тест AC-005 (error case)
func TestChromaCollectionExists_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "server error")
	}))
	defer srv.Close()

	store := NewChromaStore(srv.URL, "docs", 3)
	exists, err := store.CollectionExists(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка при HTTP 500, получен nil")
	}
	if exists {
		t.Error("CollectionExists() = true при ошибке, want false")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("ошибка должна содержать статус 500, got: %v", err)
	}
}
