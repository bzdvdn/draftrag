package draftrag

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// @sk-test llm-providers-mistral-deepseek#T2.1: TestNewMistralLLM_Interface (AC-001)
func TestNewMistralLLM_Interface(t *testing.T) {
	provider := NewMistralLLM(MistralLLMOptions{
		APIKey: "sk-test",
		Model:  "test-model",
	})
	if provider == nil {
		t.Fatal("NewMistralLLM returned nil")
	}

	if _, ok := provider.(StreamingLLMProvider); !ok {
		t.Error("expected StreamingLLMProvider interface")
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.1: TestMistralLLM_Defaults (AC-006)
func TestMistralLLM_Defaults(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["model"] != "mistral-large-latest" {
			t.Errorf("model = %v, want mistral-large-latest", req["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	provider := NewMistralLLM(MistralLLMOptions{
		BaseURL: server.URL,
		APIKey:  "sk-test",
	})
	_, err := provider.Generate(context.Background(), "", "hello")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(capturedURL, "/v1/chat/completions") {
		t.Errorf("expected /v1/chat/completions in URL, got %s", capturedURL)
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.1: TestMistralLLM_InvalidConfig (AC-005)
func TestMistralLLM_InvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		opts MistralLLMOptions
	}{
		{"empty APIKey", MistralLLMOptions{BaseURL: "http://localhost", APIKey: "", Model: "m"}},
		{"invalid URL", MistralLLMOptions{BaseURL: "no-scheme", APIKey: "sk-test", Model: "m"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMistralLLM(tt.opts)
			_, err := provider.Generate(context.Background(), "", "hello")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), ErrInvalidLLMConfig.Error()) {
				t.Errorf("expected ErrInvalidLLMConfig, got %v", err)
			}
		})
	}
}

// @sk-test llm-providers-mistral-deepseek#T2.1: TestMistralLLM_GenerateStream_Success (AC-004)
func TestMistralLLM_GenerateStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	sp, ok := NewMistralLLM(MistralLLMOptions{BaseURL: server.URL, APIKey: "sk-test", Model: "m"}).(StreamingLLMProvider)
	if !ok {
		t.Fatal("not a StreamingLLMProvider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := sp.GenerateStream(ctx, "", "hello")
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	var result string
	for chunk := range ch {
		result += chunk
	}
	if result != "Hi" {
		t.Errorf("expected 'Hi', got %q", result)
	}
}
