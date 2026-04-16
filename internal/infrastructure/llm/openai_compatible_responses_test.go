package llm

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

const testAPIKey = "secret-key"

func TestOpenAICompatibleResponsesLLM_Generate_Success_OutputText(t *testing.T) {
	apiKey := testAPIKey
	model := "test-model"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Fatalf("unexpected Authorization header: %q", got)
		}

		var req responsesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != model {
			t.Fatalf("expected model %q, got %q", model, req.Model)
		}
		if len(req.Input) != 2 {
			t.Fatalf("expected 2 input messages, got %d", len(req.Input))
		}
		if req.Input[0].Role != "system" || req.Input[1].Role != "user" {
			t.Fatalf("unexpected roles: %q, %q", req.Input[0].Role, req.Input[1].Role)
		}
		if len(req.Input[0].Content) != 1 || req.Input[0].Content[0].Type != "input_text" {
			t.Fatalf("unexpected system content: %#v", req.Input[0].Content)
		}
		if len(req.Input[1].Content) != 1 || req.Input[1].Content[0].Type != "input_text" {
			t.Fatalf("unexpected user content: %#v", req.Input[1].Content)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output_text": "answer",
		})
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, model, nil, nil)
	text, err := llm.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if text != "answer" {
		t.Fatalf("expected %q, got %q", "answer", text)
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_Success_FallbackOutput(t *testing.T) {
	apiKey := testAPIKey

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []any{
				map[string]any{
					"type": "message",
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "fallback",
						},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)
	text, err := llm.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if text != "fallback" {
		t.Fatalf("expected %q, got %q", "fallback", text)
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_Non200_RedactsAPIKey(t *testing.T) {
	apiKey := testAPIKey

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)
	_, err := llm.Generate(context.Background(), "sys", "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("expected APIKey to be redacted from error, got: %v", err)
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_InvalidJSON(t *testing.T) {
	apiKey := testAPIKey

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not-json"))
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)
	_, err := llm.Generate(context.Background(), "sys", "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_MissingText(t *testing.T) {
	apiKey := testAPIKey

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)
	_, err := llm.Generate(context.Background(), "sys", "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_ContextCancel(t *testing.T) {
	apiKey := testAPIKey

	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-unblock
	}))
	t.Cleanup(func() {
		close(unblock)
		srv.Close()
	})

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancel()

	_, err := llm.Generate(ctx, "sys", "user")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected cancellation within 100ms, took %v", time.Since(start))
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_ContextDeadline(t *testing.T) {
	apiKey := testAPIKey

	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-unblock
	}))
	t.Cleanup(func() {
		close(unblock)
		srv.Close()
	})

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	t.Cleanup(cancel)

	start := time.Now()
	_, err := llm.Generate(ctx, "sys", "user")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("expected deadline within 100ms, took %v", time.Since(start))
	}
}

func TestOpenAICompatibleResponsesLLM_Generate_IncludesOptions(t *testing.T) {
	apiKey := testAPIKey
	temp := 0.7
	maxTokens := 123

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var decoded map[string]any
		if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := decoded["temperature"]; !ok {
			t.Fatalf("expected temperature to be present, got: %#v", decoded)
		}
		if _, ok := decoded["max_output_tokens"]; !ok {
			t.Fatalf("expected max_output_tokens to be present, got: %#v", decoded)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"output_text": "ok"})
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", &temp, &maxTokens)
	text, err := llm.Generate(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if text != "ok" {
		t.Fatalf("expected %q, got %q", "ok", text)
	}
}

// TestGenerateStream_Success проверяет успешное SSE streaming.
// @sk-task T3.1: Тест GenerateStream с мок HTTP server (SSE) (AC-001, AC-005)
func TestGenerateStream_Success(t *testing.T) {
	apiKey := testAPIKey
	model := "test-model"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Fatalf("expected Accept: text/event-stream, got %q", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// SSE события
		events := []string{
			": ping\n",
			"\n",
			`data: {"type":"content","delta":{"type":"output_text","text":"Hello"}}` + "\n",
			`data: {"type":"content","delta":{"type":"output_text","text":" world"}}` + "\n",
			"data: [DONE]\n",
		}

		for _, event := range events {
			_, _ = w.Write([]byte(event))
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, model, nil, nil)
	ch, err := llm.GenerateStream(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	var result strings.Builder
	for token := range ch {
		result.WriteString(token)
	}

	if result.String() != "Hello world" {
		t.Fatalf("expected %q, got %q", "Hello world", result.String())
	}
}

// TestGenerateStream_ContextCancellation проверяет обработку отмены контекста.
// @sk-task T3.1: Тест context cancellation без утечек (AC-003, RQ-005)
func TestGenerateStream_ContextCancellation(t *testing.T) {
	apiKey := testAPIKey

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Отправляем события с задержкой
		for i := 0; i < 10; i++ {
			_, _ = w.Write([]byte(`data: {"delta":{"text":"token"}}` + "\n\n"))
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	ch, err := llm.GenerateStream(ctx, "sys", "user")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Читаем немного токенов, но контекст отменится
	count := 0
	for range ch {
		count++
	}

	// Проверяем, что канал закрылся (не заблокировался)
	if count == 0 {
		t.Log("no tokens received (context cancelled before first token)")
	}
}

// TestGenerateStream_Non200 проверяет обработку ошибок HTTP.
// @sk-task T3.1: Тест обработки ошибок streaming'а (AC-005, RQ-006)
func TestGenerateStream_Non200(t *testing.T) {
	apiKey := testAPIKey

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	t.Cleanup(srv.Close)

	llm := NewOpenAICompatibleResponsesLLM(srv.Client(), srv.URL, apiKey, "test-model", nil, nil)
	ch, err := llm.GenerateStream(context.Background(), "sys", "user")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ch != nil {
		t.Fatal("expected nil channel on error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected error with 401, got %v", err)
	}
}
