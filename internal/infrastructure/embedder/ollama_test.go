package embedder

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

// @sk-task T3.3: Тест на успешный embedding (AC-002)
func TestOllamaEmbedder_Embed_Success(t *testing.T) {
	// Мок-сервер, возвращающий валидный Ollama-ответ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Проверка пути
		if r.URL.Path != ollamaEmbeddingsPath {
			t.Errorf("expected path %s, got %s", ollamaEmbeddingsPath, r.URL.Path)
		}

		// Проверка Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", contentType)
		}

		// Чтение и проверка тела запроса (AC-002)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var req ollamaEmbedRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if req.Model == "" {
			t.Error("model field is empty")
		}
		if req.Prompt == "" {
			t.Error("prompt field is empty (should contain text, not 'input' as in OpenAI)")
		}

		// Ответ в формате Ollama (embedding напрямую, не в data[0])
		resp := ollamaEmbedResponse{
			Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaEmbedder(nil, server.URL, "", "nomic-embed-text")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Embed(ctx, "Hello world")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(result) != 5 {
		t.Errorf("expected embedding length 5, got %d", len(result))
	}
	expected := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("expected embedding[%d]=%f, got %f", i, expected[i], v)
		}
	}
}

// @sk-task T3.3: Тест на пустой текст (AC-005)
func TestOllamaEmbedder_Embed_EmptyText(t *testing.T) {
	client := NewOllamaEmbedder(nil, "http://localhost", "", "model")

	ctx := context.Background()

	_, err := client.Embed(ctx, "   ")
	if err == nil {
		t.Error("expected error for empty text, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error message to contain 'empty', got %q", err.Error())
	}
}

// @sk-task T3.3: Тест на nil context (AC-005)
func TestOllamaEmbedder_Embed_NilContext(t *testing.T) {
	client := NewOllamaEmbedder(nil, "http://localhost", "", "model")

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context, but function did not panic")
		}
	}()

	//nolint:staticcheck // Нам нужно передать nil context, чтобы проверить, что метод паникует.
	_, _ = client.Embed(nil, "Hello")
}

// @sk-task T3.3: Тест на HTTP ошибки (AC-003)
func TestOllamaEmbedder_Embed_HTTPError(t *testing.T) {
	// Мок-сервер, возвращающий ошибку 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	client := NewOllamaEmbedder(nil, server.URL, "", "unknown-model")

	ctx := context.Background()
	_, err := client.Embed(ctx, "Hello")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected error to contain 'failed', got %q", err.Error())
	}
}

// @sk-task T3.3: Тест на таймаут контекста (AC-004)
func TestOllamaEmbedder_Embed_ContextTimeout(t *testing.T) {
	// Мок-сервер с задержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		})
	}))
	defer server.Close()

	client := NewOllamaEmbedder(nil, server.URL, "", "model")

	// Контекст с очень коротким таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Ждём чтобы таймаут точно истёк
	time.Sleep(10 * time.Millisecond)

	_, err := client.Embed(ctx, "Hello")
	if err == nil {
		t.Fatal("expected error for timed out context, got nil")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

// @sk-task T3.3: Тест на пустой embedding (краевой случай)
func TestOllamaEmbedder_Embed_EmptyEmbedding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ollamaEmbedResponse{
			Embedding: []float64{}, // пустой embedding
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaEmbedder(nil, server.URL, "", "model")

	ctx := context.Background()
	_, err := client.Embed(ctx, "Hello")
	if err == nil {
		t.Fatal("expected error for empty embedding, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error to contain 'empty', got %q", err.Error())
	}
}

// @sk-task T3.4: Тест конструктора с default base URL (DEC-003)
func TestNewOllamaEmbedder_DefaultBaseURL(t *testing.T) {
	client := NewOllamaEmbedder(nil, "", "", "nomic-embed-text")

	if client.baseURL != ollamaDefaultBaseURL {
		t.Errorf("expected baseURL=%q, got %q", ollamaDefaultBaseURL, client.baseURL)
	}
}

// @sk-task T3.4: Тест конструктора с кастомным httpClient
func TestNewOllamaEmbedder_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}
	client := NewOllamaEmbedder(customClient, "http://localhost:11434", "", "model")

	if client.httpClient != customClient {
		t.Error("expected custom httpClient to be set")
	}
}

// @sk-task T3.4: Тест конструктора с пустым apiKey (DEC-002)
func TestNewOllamaEmbedder_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем что Authorization заголовок НЕ установлен (Ollama не требует его)
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}

		resp := ollamaEmbedResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaEmbedder(nil, server.URL, "", "model")

	ctx := context.Background()
	_, err := client.Embed(ctx, "Hello")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
}

// @sk-task T3.3: Тест с проверкой что используется 'prompt' а не 'input' (AC-002)
func TestOllamaEmbedder_Embed_PromptField(t *testing.T) {
	var capturedReq ollamaEmbedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &capturedReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		resp := ollamaEmbedResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOllamaEmbedder(nil, server.URL, "", "model")

	ctx := context.Background()
	_, err := client.Embed(ctx, "Test prompt text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if capturedReq.Prompt != "Test prompt text" {
		t.Errorf("expected prompt=%q, got %q", "Test prompt text", capturedReq.Prompt)
	}
	// Проверяем что нет поля Input (как в OpenAI)
	// В структуре ollamaEmbedRequest нет поля Input, так что если бы мы
	// отправили неправильный JSON, сервер бы не распарсил его корректно
}
