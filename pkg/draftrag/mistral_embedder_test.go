package draftrag

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test llm-providers-mistral-deepseek#T2.3: TestNewMistralEmbedder_Interface (AC-008)
func TestNewMistralEmbedder_Interface(t *testing.T) {
	emb := NewMistralEmbedder(MistralEmbedderOptions{
		APIKey: "sk-test",
	})
	if emb == nil {
		t.Fatal("NewMistralEmbedder returned nil")
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.3: TestMistralEmbedder_Defaults (AC-011)
func TestMistralEmbedder_Defaults(t *testing.T) {
	var capturedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		capturedModel, _ = req["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"embedding": []float64{0.1, 0.2}}},
		})
	}))
	defer server.Close()

	emb := NewMistralEmbedder(MistralEmbedderOptions{
		BaseURL: server.URL,
		APIKey:  "sk-test",
	})
	_, err := emb.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if capturedModel != "mistral-embed" {
		t.Errorf("model = %q, want mistral-embed", capturedModel)
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.3: TestMistralEmbedder_InvalidConfig (AC-010)
func TestMistralEmbedder_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		opts MistralEmbedderOptions
	}{
		{"empty APIKey", MistralEmbedderOptions{BaseURL: "http://localhost", APIKey: "", Model: "m"}},
		{"invalid URL", MistralEmbedderOptions{BaseURL: "no-scheme", APIKey: "sk-test", Model: "m"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb := NewMistralEmbedder(tt.opts)
			_, err := emb.Embed(context.Background(), "hello")
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidEmbedderConfig) {
				t.Errorf("expected ErrInvalidEmbedderConfig, got %v", err)
			}
		})
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.3: TestMistralEmbedder_PipelineFullCycle (AC-008)
func TestMistralEmbedder_PipelineFullCycle(t *testing.T) {
	apiKey := "test-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"embedding": []float64{1, 0},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	emb := NewMistralEmbedder(MistralEmbedderOptions{
		BaseURL: srv.URL,
		APIKey:  apiKey,
	})

	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, testLLMProvider{}, emb)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := p.Index(ctx, []Document{{ID: "doc-1", Content: "cat"}}); err != nil {
		t.Fatalf("index: %v", err)
	}

	result, err := p.Search("cat").TopK(5).Retrieve(ctx)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(result.Chunks) == 0 {
		t.Fatalf("expected non-empty results")
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.3: TestMistralEmbedder_RedactsAPIKey
func TestMistralEmbedder_RedactsAPIKey(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	emb := NewMistralEmbedder(MistralEmbedderOptions{
		BaseURL: srv.URL,
		APIKey:  apiKey,
	})

	_, err := emb.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("expected APIKey to be redacted from error, got: %v", err)
	}
}
