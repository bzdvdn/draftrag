package draftrag

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type recordLogger struct {
	mu     sync.Mutex
	events []string
}

func (l *recordLogger) Log(ctx context.Context, level LogLevel, msg string, fields ...LogField) {
	_ = ctx
	_ = level

	var b strings.Builder
	b.WriteString(msg)
	for _, f := range fields {
		fmt.Fprintf(&b, " %s=%v", f.Key, f.Value)
	}

	l.mu.Lock()
	l.events = append(l.events, b.String())
	l.mu.Unlock()
}

func (l *recordLogger) Snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.events))
	copy(out, l.events)
	return out
}

func TestRetryLLMProvider_LoggerDoesNotLeakAPIKey(t *testing.T) {
	apiKey := "secret-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request: " + apiKey))
	}))
	t.Cleanup(srv.Close)

	base := NewOpenAICompatibleLLM(OpenAICompatibleLLMOptions{
		BaseURL: srv.URL,
		APIKey:  apiKey,
		Model:   "test-model",
	})

	logger := &recordLogger{}
	llm := NewRetryLLMProvider(base, RetryOptions{
		MaxRetries: 1,
		Logger:     logger,
	})

	_, _ = llm.Generate(context.Background(), "sys", "user")

	for _, ev := range logger.Snapshot() {
		if strings.Contains(ev, apiKey) {
			t.Fatalf("expected APIKey to be redacted from logs, got: %s", ev)
		}
	}
}
