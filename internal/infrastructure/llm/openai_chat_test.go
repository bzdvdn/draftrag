package llm

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

const testChatAPIKey = "sk-chat-test-key"

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_Generate_Success (AC-003)
func TestOpenAIChat_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+testChatAPIKey {
			t.Errorf("expected Authorization=Bearer %s, got %s", testChatAPIKey, auth)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", contentType)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req chatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		if req.Model == "" {
			t.Error("model field is empty")
		}
		if len(req.Messages) == 0 {
			t.Error("messages array is empty")
		}

		resp := chatResponse{
			Choices: []chatChoice{
				{Message: chatMessage{Role: "assistant", Content: "Hello from Chat!"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIChatLLM(nil, server.URL, testChatAPIKey, "test-model", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Generate(ctx, "System prompt", "User message")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "Hello from Chat!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_Generate_EmptyUserMessage (AC-003)
func TestOpenAIChat_Generate_EmptyUserMessage(t *testing.T) {
	client := NewOpenAIChatLLM(nil, "http://localhost", "key", "model", nil, nil)

	_, err := client.Generate(context.Background(), "System", "   ")
	if err == nil {
		t.Error("expected error for empty userMessage")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error containing 'empty', got %v", err)
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_Generate_NilContext (AC-003)
func TestOpenAIChat_Generate_NilContext(t *testing.T) {
	client := NewOpenAIChatLLM(nil, "http://localhost", "key", "model", nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context")
		}
	}()

	//nolint:staticcheck
	_, _ = client.Generate(nil, "System", "User")
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_Generate_HTTPError (AC-003)
func TestOpenAIChat_Generate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	client := NewOpenAIChatLLM(nil, server.URL, testChatAPIKey, "model", nil, nil)

	_, err := client.Generate(context.Background(), "System", "User")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "status=401") {
		t.Errorf("expected status=401 in error, got %v", err)
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_Generate_EmptyResponse (AC-003)
func TestOpenAIChat_Generate_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatResponse{Choices: []chatChoice{}}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIChatLLM(nil, server.URL, "key", "model", nil, nil)

	_, err := client.Generate(context.Background(), "System", "User")
	if err == nil {
		t.Error("expected error for empty response")
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_GenerateStream_Success (AC-004)
func TestOpenAIChat_GenerateStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "text/event-stream" {
			t.Errorf("expected Accept=text/event-stream, got %s", accept)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}

		events := []string{
			`{"choices":[{"delta":{"content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" "}}]}`,
			`{"choices":[{"delta":{"content":"world!"}}]}`,
		}

		for _, event := range events {
			_, _ = w.Write([]byte("data: " + event + "\n\n"))
			flusher.Flush()
		}

		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewOpenAIChatLLM(nil, server.URL, "key", "model", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := client.GenerateStream(ctx, "System", "User message")
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	var chunks []string
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	result := strings.Join(chunks, "")
	expected := "Hello world!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_GenerateStream_EmptyUserMessage (AC-004)
func TestOpenAIChat_GenerateStream_EmptyUserMessage(t *testing.T) {
	client := NewOpenAIChatLLM(nil, "http://localhost", "key", "model", nil, nil)

	_, err := client.GenerateStream(context.Background(), "System", "   ")
	if err == nil {
		t.Error("expected error for empty userMessage")
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_GenerateStream_NilContext (AC-004)
func TestOpenAIChat_GenerateStream_NilContext(t *testing.T) {
	client := NewOpenAIChatLLM(nil, "http://localhost", "key", "model", nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context")
		}
	}()

	//nolint:staticcheck
	_, _ = client.GenerateStream(nil, "System", "User")
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestOpenAIChat_GenerateStream_HTTPError (AC-004)
func TestOpenAIChat_GenerateStream_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client := NewOpenAIChatLLM(nil, server.URL, "key", "model", nil, nil)

	_, err := client.GenerateStream(context.Background(), "System", "User")
	if err == nil {
		t.Fatal("expected error for bad request")
	}
}

// @sk-test llm-providers-mistral-deepseek#T1.2: TestBuildChatURL (AC-003)
func TestBuildChatURL(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		want    string
		wantErr bool
	}{
		{
			name: "valid URL",
			base: "https://api.mistral.ai",
			want: "https://api.mistral.ai/v1/chat/completions",
		},
		{
			name: "URL with trailing slash",
			base: "https://api.deepseek.com/",
			want: "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name:    "invalid URL - no scheme",
			base:    "api.mistral.ai",
			wantErr: true,
		},
		{
			name:    "invalid URL - no host",
			base:    "http://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildChatURL(tt.base)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildChatURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildChatURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
