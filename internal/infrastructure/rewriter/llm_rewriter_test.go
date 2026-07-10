package rewriter

import (
	"context"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test query-rewriting#T4.2: TestLLMRewriter_Rewrite (AC-006)

type mockLLM struct {
	reply string
}

func (m *mockLLM) Health(_ context.Context) error { return nil }
func (m *mockLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return m.reply, nil
}

// TestLLMRewriter_Rewrite проверяет базовую работу LLMRewriter с mock LLM.
func TestLLMRewriter_Rewrite(t *testing.T) {
	rw, err := NewLLMRewriter(&mockLLM{reply: "rewritten query"}, "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := rw.Rewrite(context.Background(), "original query", domain.QueryHistory{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one rewritten query")
	}
	if result[0].Query != "rewritten query" {
		t.Fatalf("expected 'rewritten query', got %q", result[0].Query)
	}
}

// TestLLMRewriter_EmptyResult проверяет fallback при пустом ответе LLM.
func TestLLMRewriter_EmptyResult(t *testing.T) {
	rw, err := NewLLMRewriter(&mockLLM{reply: ""}, "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := rw.Rewrite(context.Background(), "original", domain.QueryHistory{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("expected fallback to original query")
	}
	if result[0].Query != "original" {
		t.Fatalf("expected fallback 'original', got %q", result[0].Query)
	}
}

// TestLLMRewriter_MultiLine проверяет парсинг multi-line ответа (multi-query).
func TestLLMRewriter_MultiLine(t *testing.T) {
	rw, err := NewLLMRewriter(&mockLLM{reply: "variant one\nvariant two\nvariant three"}, "")
	if err != nil {
		t.Fatal(err)
	}

	result, err := rw.Rewrite(context.Background(), "original", domain.QueryHistory{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(result))
	}
}

// TestLLMRewriter_History проверяет, что история передаётся в user message.
func TestLLMRewriter_History(t *testing.T) {
	var capturedUserMsg string
	customLLM := &mockLLMWithCapture{
		fn: func(_, userMsg string) string {
			capturedUserMsg = userMsg
			return "rewritten"
		},
	}

	rw, err := NewLLMRewriter(customLLM, "custom prompt")
	if err != nil {
		t.Fatal(err)
	}

	history := domain.QueryHistory{
		Entries: []domain.Message{
			{Role: "user", Content: "previous question"},
			{Role: "assistant", Content: "previous answer"},
		},
	}

	_, err = rw.Rewrite(context.Background(), "new question", history)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(capturedUserMsg, "previous question") {
		t.Fatal("expected history in user message")
	}
	if !strings.Contains(capturedUserMsg, "new question") {
		t.Fatal("expected current query in user message")
	}
}

type mockLLMWithCapture struct {
	fn func(system, user string) string
}

func (m *mockLLMWithCapture) Health(_ context.Context) error { return nil }
func (m *mockLLMWithCapture) Generate(_ context.Context, system, user string) (string, error) {
	return m.fn(system, user), nil
}
