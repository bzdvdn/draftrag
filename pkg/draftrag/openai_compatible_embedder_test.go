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

var _ Embedder = NewOpenAICompatibleEmbedder(OpenAICompatibleEmbedderOptions{})

type testLLMProvider struct{}

func (testLLMProvider) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "ok", nil
}

func TestOpenAICompatibleEmbedder_ConfigValidation(t *testing.T) {
	emb := NewOpenAICompatibleEmbedder(OpenAICompatibleEmbedderOptions{
		BaseURL: "",
		APIKey:  "k",
		Model:   "m",
	})

	_, err := emb.Embed(context.Background(), "hello")
	if !errors.Is(err, ErrInvalidEmbedderConfig) {
		t.Fatalf("expected ErrInvalidEmbedderConfig, got %v", err)
	}
}

func TestOpenAICompatibleEmbedder_PipelineFullCycle(t *testing.T) {
	apiKey := "test-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	emb := NewOpenAICompatibleEmbedder(OpenAICompatibleEmbedderOptions{
		BaseURL: srv.URL,
		APIKey:  apiKey,
		Model:   "test-model",
	})

	store := vectorstore.NewInMemoryStore()
	p := NewPipeline(store, testLLMProvider{}, emb)

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

func TestOpenAICompatibleEmbedder_RedactsAPIKey(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	emb := NewOpenAICompatibleEmbedder(OpenAICompatibleEmbedderOptions{
		BaseURL: srv.URL,
		APIKey:  apiKey,
		Model:   "test-model",
	})

	_, err := emb.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("expected APIKey to be redacted from error, got: %v", err)
	}
}
