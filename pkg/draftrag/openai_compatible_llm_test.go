package draftrag

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var _ LLMProvider = NewOpenAICompatibleLLM(OpenAICompatibleLLMOptions{})

func TestOpenAICompatibleLLM_ConfigValidation(t *testing.T) {
	llm := NewOpenAICompatibleLLM(OpenAICompatibleLLMOptions{
		BaseURL: "",
		APIKey:  "k",
		Model:   "m",
	})

	_, err := llm.Generate(context.Background(), "sys", "user")
	if !errors.Is(err, ErrInvalidLLMConfig) {
		t.Fatalf("expected ErrInvalidLLMConfig, got %v", err)
	}
}

func TestOpenAICompatibleLLM_RedactsAPIKey(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleLLM(OpenAICompatibleLLMOptions{
		BaseURL: srv.URL,
		APIKey:  apiKey,
		Model:   "test-model",
	})

	_, err := llm.Generate(context.Background(), "sys", "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("expected APIKey to be redacted from error, got: %v", err)
	}
}
