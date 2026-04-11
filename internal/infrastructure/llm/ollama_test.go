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

// @sk-task T3.1: Тест на успешную генерацию (AC-001)
func TestOllamaLLM_Generate_Success(t *testing.T) {
	// Мок-сервер, возвращающий валидный Ollama-ответ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Проверка пути
		if r.URL.Path != ollamaChatPath {
			t.Errorf("expected path %s, got %s", ollamaChatPath, r.URL.Path)
		}

		// Проверка Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", contentType)
		}

		// Чтение и проверка тела запроса (AC-001)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var req ollamaChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if req.Model == "" {
			t.Error("model field is empty")
		}
		if req.Stream {
			t.Error("stream should be false for synchronous call")
		}
		if len(req.Messages) == 0 {
			t.Error("messages array is empty")
		}
		if req.Messages[len(req.Messages)-1].Role != "user" {
			t.Errorf("expected last role=user, got %s", req.Messages[len(req.Messages)-1].Role)
		}

		// Ответ в формате Ollama
		resp := ollamaChatResponse{
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "Hello from Ollama!",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaLLM(nil, server.URL, "", "llama3.2", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Generate(ctx, "System prompt", "User message")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "Hello from Ollama!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// @sk-task T3.1: Тест на пустой userMessage (AC-005)
func TestOllamaLLM_Generate_EmptyUserMessage(t *testing.T) {
	client := NewOllamaLLM(nil, "http://localhost", "", "model", nil, nil)

	ctx := context.Background()

	_, err := client.Generate(ctx, "System", "   ")
	if err == nil {
		t.Error("expected error for empty userMessage, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error message to contain 'empty', got %q", err.Error())
	}
}

// @sk-task T3.1: Тест на nil context (AC-005)
func TestOllamaLLM_Generate_NilContext(t *testing.T) {
	client := NewOllamaLLM(nil, "http://localhost", "", "model", nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context, but function did not panic")
		}
	}()

	client.Generate(nil, "System", "User")
}

// @sk-task T3.1: Тест на HTTP ошибки (AC-003)
func TestOllamaLLM_Generate_HTTPError(t *testing.T) {
	// Мок-сервер, возвращающий ошибку 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	client := NewOllamaLLM(nil, server.URL, "", "unknown-model", nil, nil)

	ctx := context.Background()
	_, err := client.Generate(ctx, "", "Hello")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected error to contain 'failed', got %q", err.Error())
	}
}

// @sk-task T3.1: Тест на таймаут контекста (AC-004)
func TestOllamaLLM_Generate_ContextTimeout(t *testing.T) {
	// Мок-сервер с задержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: "Late response"},
		})
	}))
	defer server.Close()

	client := NewOllamaLLM(nil, server.URL, "", "model", nil, nil)

	// Контекст с очень коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Ждём чтобы таймаут точно истёк
	time.Sleep(10 * time.Millisecond)

	_, err := client.Generate(ctx, "", "Hello")
	if err == nil {
		t.Fatal("expected error for timed out context, got nil")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

// @sk-task T3.1: Тест на пустой ответ от модели (краевой случай)
func TestOllamaLLM_Generate_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "   ", // пустой контент
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaLLM(nil, server.URL, "", "model", nil, nil)

	ctx := context.Background()
	_, err := client.Generate(ctx, "", "Hello")
	if err == nil {
		t.Fatal("expected error for empty response content, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error to contain 'empty', got %q", err.Error())
	}
}

// @sk-task T3.2: Тест конструктора с default base URL (DEC-003)
func TestNewOllamaLLM_DefaultBaseURL(t *testing.T) {
	client := NewOllamaLLM(nil, "", "", "llama3.2", nil, nil)

	if client.baseURL != ollamaDefaultBaseURL {
		t.Errorf("expected baseURL=%q, got %q", ollamaDefaultBaseURL, client.baseURL)
	}
}

// @sk-task T3.2: Тест конструктора с кастомным httpClient
func TestNewOllamaLLM_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}
	client := NewOllamaLLM(customClient, "http://localhost:11434", "", "model", nil, nil)

	if client.httpClient != customClient {
		t.Error("expected custom httpClient to be set")
	}
}

// @sk-task T3.2: Тест конструктора с параметрами temperature и maxTokens
func TestNewOllamaLLM_OptionalParameters(t *testing.T) {
	temp := 0.7
	maxTokens := 2048

	client := NewOllamaLLM(nil, "http://localhost", "", "model", &temp, &maxTokens)

	if client.temperature == nil || *client.temperature != temp {
		t.Errorf("expected temperature=%f, got %v", temp, client.temperature)
	}
	if client.maxTokens == nil || *client.maxTokens != maxTokens {
		t.Errorf("expected maxTokens=%d, got %v", maxTokens, client.maxTokens)
	}
}

// @sk-task T3.1: Тест с system prompt (проверка формата messages)
func TestOllamaLLM_Generate_WithSystemPrompt(t *testing.T) {
	var capturedReq ollamaChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedReq)

		resp := ollamaChatResponse{
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "Response with system",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaLLM(nil, server.URL, "", "model", nil, nil)

	ctx := context.Background()
	result, err := client.Generate(ctx, "You are helpful", "Hello")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Проверяем что messages содержит system и user
	if len(capturedReq.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(capturedReq.Messages))
	}
	if capturedReq.Messages[0].Role != "system" {
		t.Errorf("expected first message role=system, got %s", capturedReq.Messages[0].Role)
	}
	if capturedReq.Messages[1].Role != "user" {
		t.Errorf("expected second message role=user, got %s", capturedReq.Messages[1].Role)
	}

	if result != "Response with system" {
		t.Errorf("expected %q, got %q", "Response with system", result)
	}
}

// @sk-task T3.1: Тест с temperature и max_tokens в запросе (RQ-003)
func TestOllamaLLM_Generate_WithParameters(t *testing.T) {
	var capturedReq ollamaChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedReq)

		resp := ollamaChatResponse{
			Message: ollamaMessage{
				Role:    "assistant",
				Content: "Response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	temp := 0.5
	maxTokens := 512

	client := NewOllamaLLM(nil, server.URL, "", "model", &temp, &maxTokens)

	ctx := context.Background()
	_, err := client.Generate(ctx, "", "Hello")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if capturedReq.Temperature == nil || *capturedReq.Temperature != temp {
		t.Errorf("expected temperature=%f in request, got %v", temp, capturedReq.Temperature)
	}
	if capturedReq.MaxTokens == nil || *capturedReq.MaxTokens != maxTokens {
		t.Errorf("expected max_tokens=%d in request, got %v", maxTokens, capturedReq.MaxTokens)
	}
}
