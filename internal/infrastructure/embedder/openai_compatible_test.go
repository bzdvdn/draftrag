package embedder

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAICompatibleEmbedder_Embed_Success(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("expected /v1/embeddings, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Fatalf("unexpected Authorization header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"embedding": []float64{1, 2, 3},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	emb := NewOpenAICompatibleEmbedder(srv.Client(), srv.URL, apiKey, "test-model")
	vec, err := emb.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected len=3, got %d", len(vec))
	}
}

func TestOpenAICompatibleEmbedder_Embed_Non200_RedactsAPIKey(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	emb := NewOpenAICompatibleEmbedder(srv.Client(), srv.URL, apiKey, "test-model")
	_, err := emb.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("expected APIKey to be redacted from error, got: %v", err)
	}
}

func TestOpenAICompatibleEmbedder_Embed_InvalidJSON(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not-json"))
	}))
	t.Cleanup(srv.Close)

	emb := NewOpenAICompatibleEmbedder(srv.Client(), srv.URL, apiKey, "test-model")
	_, err := emb.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOpenAICompatibleEmbedder_Embed_ContextCancel(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	emb := NewOpenAICompatibleEmbedder(srv.Client(), srv.URL, apiKey, "test-model")

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	_, err := emb.Embed(ctx, "hello")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}
