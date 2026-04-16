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

const (
	testAnthropicAPIKey = "sk-secret-key-12345"
	roleUser            = "user"
)

// @sk-task T1.3: Тест на успешную генерацию (AC-001)
func TestClaudeLLM_Generate_Success(t *testing.T) {
	// Мок-сервер, возвращающий валидный Anthropic-ответ
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Проверка заголовка anthropic-version (AC-003)
		version := r.Header.Get("anthropic-version")
		if version == "" {
			t.Error("anthropic-version header is missing")
		}
		if version != defaultAnthropicVersion {
			t.Errorf("expected anthropic-version=%s, got %s", defaultAnthropicVersion, version)
		}

		// Проверка X-API-Key заголовка
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			t.Error("X-API-Key header is missing")
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

		var req messagesRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		if req.Model == "" {
			t.Error("model field is empty")
		}
		if req.MaxTokens <= 0 {
			t.Errorf("max_tokens should be positive, got %d", req.MaxTokens)
		}
		if len(req.Messages) == 0 {
			t.Error("messages array is empty")
		}
		if req.Messages[0].Role != roleUser {
			t.Errorf("expected role=user, got %s", req.Messages[0].Role)
		}

		// Ответ в формате Anthropic
		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: "Hello from Claude!"},
			},
			Role: "assistant",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClaudeLLM(nil, server.URL, "test-api-key", "", "", nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Generate(ctx, "System prompt", "User message")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "Hello from Claude!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// @sk-task T1.3: Тест на пустой userMessage
func TestClaudeLLM_Generate_EmptyUserMessage(t *testing.T) {
	client := NewClaudeLLM(nil, "http://localhost", "test-key", "", "", nil, nil)

	_, err := client.Generate(context.Background(), "System", "   ")
	if err == nil {
		t.Error("expected error for empty userMessage")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error containing 'empty', got %v", err)
	}
}

// @sk-task T1.3: Тест на nil context
func TestClaudeLLM_Generate_NilContext(t *testing.T) {
	client := NewClaudeLLM(nil, "http://localhost", "test-key", "", "", nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context")
		}
	}()

	//nolint:staticcheck // Нам нужно передать nil context, чтобы проверить, что метод паникует.
	_, _ = client.Generate(nil, "System", "User")
}

// @sk-task T1.3: Тест на отмену контекста
func TestClaudeLLM_Generate_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClaudeLLM(nil, server.URL, "test-key", "", "", nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Сразу отменяем

	_, err := client.Generate(ctx, "System", "User")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

// @sk-task T1.3: Тест на пустой ответ от API
func TestClaudeLLM_Generate_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := messagesResponse{
			Content: []contentBlock{},
			Role:    "assistant",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClaudeLLM(nil, server.URL, "test-key", "", "", nil, nil)

	_, err := client.Generate(context.Background(), "System", "User")
	if err == nil {
		t.Error("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("expected error containing 'missing', got %v", err)
	}
}

// @sk-task T2.3: Тест на streaming (AC-004)
func TestClaudeLLM_GenerateStream_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверка заголовка Accept
		accept := r.Header.Get("Accept")
		if accept != "text/event-stream" {
			t.Errorf("expected Accept=text/event-stream, got %s", accept)
		}

		// SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}

		events := []string{
			`{"type": "content_block_delta", "delta": {"type": "text", "text": "Hello"}}`,
			`{"type": "content_block_delta", "delta": {"type": "text", "text": " "}}`,
			`{"type": "content_block_delta", "delta": {"type": "text", "text": "world!"}}`,
		}

		for _, event := range events {
			_, _ = w.Write([]byte("data: " + event + "\n\n"))
			flusher.Flush()
		}

		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClaudeLLM(nil, server.URL, "test-api-key", "", "", nil, nil)

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

	// Собираем все чанки вместе
	result := strings.Join(chunks, "")
	expected := "Hello world!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// @sk-task T2.3: Тест на streaming с пустым userMessage
func TestClaudeLLM_GenerateStream_EmptyUserMessage(t *testing.T) {
	client := NewClaudeLLM(nil, "http://localhost", "test-key", "", "", nil, nil)

	_, err := client.GenerateStream(context.Background(), "System", "   ")
	if err == nil {
		t.Error("expected error for empty userMessage")
	}
}

// @sk-task T2.3: Тест на nil context в streaming
func TestClaudeLLM_GenerateStream_NilContext(t *testing.T) {
	client := NewClaudeLLM(nil, "http://localhost", "test-key", "", "", nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil context")
		}
	}()

	//nolint:staticcheck // Нам нужно передать nil context, чтобы проверить, что метод паникует.
	_, _ = client.GenerateStream(nil, "System", "User")
}

// @sk-task T2.4: Тест на ошибку 401 (AC-005)
func TestClaudeLLM_Generate_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"type": "authentication_error", "message": "Invalid API key"}}`))
	}))
	defer server.Close()

	apiKey := testAnthropicAPIKey
	client := NewClaudeLLM(nil, server.URL, apiKey, "", "", nil, nil)

	_, err := client.Generate(context.Background(), "System", "User")
	if err == nil {
		t.Error("expected error for 401")
	}

	// Проверка редатации ключа
	if strings.Contains(err.Error(), apiKey) {
		t.Error("error message should not contain raw API key")
	}
}

// @sk-task T2.4: Тест на ошибку 429 (AC-005)
func TestClaudeLLM_Generate_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	apiKey := testAnthropicAPIKey
	client := NewClaudeLLM(nil, server.URL, apiKey, "", "", nil, nil)

	_, err := client.Generate(context.Background(), "System", "User")
	if err == nil {
		t.Error("expected error for 429")
	}

	// Проверка редатации ключа
	if strings.Contains(err.Error(), apiKey) {
		t.Error("error message should not contain raw API key")
	}
}

// @sk-task T2.4: Тест на ошибку в streaming (AC-005)
func TestClaudeLLM_GenerateStream_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"type": "invalid_request_error", "message": "Invalid request"}}`))
	}))
	defer server.Close()

	apiKey := testAnthropicAPIKey
	client := NewClaudeLLM(nil, server.URL, apiKey, "", "", nil, nil)

	_, err := client.GenerateStream(context.Background(), "System", "User")
	if err == nil {
		t.Error("expected error for bad request")
	}

	// Проверка редатации ключа
	if strings.Contains(err.Error(), apiKey) {
		t.Error("error message should not contain raw API key")
	}
}

// @sk-task T2.4: Тест на редатацию ключа в ошибке (AC-005)
func TestClaudeLLM_Generate_KeyRedaction(t *testing.T) {
	apiKey := "sk-ant-test-secret-key-1234567890abcdef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Возвращаем ошибку, содержащую API ключ (имитация случайной утечки)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid key: ` + apiKey + `"}}`))
	}))
	defer server.Close()

	client := NewClaudeLLM(nil, server.URL, apiKey, "", "", nil, nil)

	_, err := client.Generate(context.Background(), "System", "User")
	if err == nil {
		t.Error("expected error")
	}

	// Ключ должен быть редатирован
	if strings.Contains(err.Error(), apiKey) {
		t.Error("API key should be redacted in error message")
	}

	// Должен быть маркер редатации
	if !strings.Contains(err.Error(), "<redacted>") {
		t.Error("expected <redacted> marker in error")
	}
}

// Тест на значения по умолчанию
func TestNewClaudeLLM_Defaults(t *testing.T) {
	client := NewClaudeLLM(nil, "http://localhost", "key", "", "", nil, nil)

	if client.model != defaultAnthropicModel {
		t.Errorf("expected default model %s, got %s", defaultAnthropicModel, client.model)
	}

	if client.anthropicVersion != defaultAnthropicVersion {
		t.Errorf("expected default version %s, got %s", defaultAnthropicVersion, client.anthropicVersion)
	}
}

// Тест на кастомные значения
func TestNewClaudeLLM_CustomValues(t *testing.T) {
	customModel := "claude-3-opus-20240229"
	customVersion := "2024-01-01"
	maxTokens := 2048

	client := NewClaudeLLM(nil, "http://localhost", "key", customModel, customVersion, nil, &maxTokens)

	if client.model != customModel {
		t.Errorf("expected model %s, got %s", customModel, client.model)
	}

	if client.anthropicVersion != customVersion {
		t.Errorf("expected version %s, got %s", customVersion, client.anthropicVersion)
	}

	if *client.maxTokens != maxTokens {
		t.Errorf("expected maxTokens %d, got %d", maxTokens, *client.maxTokens)
	}
}

// Тест на отсутствие system prompt в запросе, когда он пустой
func TestClaudeLLM_Generate_NoSystemPrompt(t *testing.T) {
	var capturedReq messagesRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if err := json.Unmarshal(body, &capturedReq); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: "Response"}},
			Role:    "assistant",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClaudeLLM(nil, server.URL, "key", "", "", nil, nil)

	_, _ = client.Generate(context.Background(), "", "User message")

	// System поле должно быть пустым (omitempty)
	if capturedReq.System != "" {
		t.Errorf("expected empty system field, got %q", capturedReq.System)
	}
}

// Тест на buildAnthropicURL
func TestBuildAnthropicURL(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		want    string
		wantErr bool
	}{
		{
			name: "valid URL",
			base: "https://api.anthropic.com",
			want: "https://api.anthropic.com/v1/messages",
		},
		{
			name: "URL with trailing slash",
			base: "https://api.anthropic.com/",
			want: "https://api.anthropic.com/v1/messages",
		},
		{
			name:    "invalid URL - no scheme",
			base:    "api.anthropic.com",
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
			got, err := buildAnthropicURL(tt.base)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAnthropicURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildAnthropicURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
