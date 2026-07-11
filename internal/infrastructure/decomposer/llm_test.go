package decomposer

import (
	"context"
	"errors"
	"testing"
)

type mockDecomposerLLM struct {
	reply string
	err   error
}

func (m *mockDecomposerLLM) Health(_ context.Context) error { return nil }
func (m *mockDecomposerLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return m.reply, m.err
}

// @sk-test sub-query-decomposition#T2.3: TestLLMQueryDecomposer_ValidJSON (AC-002)
func TestLLMQueryDecomposer_ValidJSON(t *testing.T) {
	d, err := NewLLMQueryDecomposer(&mockDecomposerLLM{
		reply: `["what are requirements?", "what is pricing?"]`,
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test query")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-queries, got %d", len(subs))
	}
	if subs[0] != "what are requirements?" {
		t.Fatalf("expected 'what are requirements?', got %q", subs[0])
	}
}

// @sk-test sub-query-decomposition#T2.3: TestLLMQueryDecomposer_InvalidJSON_Fallback (AC-002)
func TestLLMQueryDecomposer_InvalidJSON_Fallback(t *testing.T) {
	d, err := NewLLMQueryDecomposer(&mockDecomposerLLM{
		reply: `Here are the sub-questions: ["sub q1", "sub q2"]`,
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test query")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-queries from regex fallback, got %d", len(subs))
	}
}

// @sk-test sub-query-decomposition#T2.3: TestLLMQueryDecomposer_LLMError (AC-002)
func TestLLMQueryDecomposer_LLMError(t *testing.T) {
	d, err := NewLLMQueryDecomposer(&mockDecomposerLLM{
		reply: "",
		err:   errors.New("llm unavailable"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = d.Decompose(context.Background(), "test query")
	if err == nil {
		t.Fatal("expected error from LLM, got nil")
	}
}

// @sk-test sub-query-decomposition#T2.3: TestLLMQueryDecomposer_EmptyResponse (AC-002)
func TestLLMQueryDecomposer_EmptyResponse(t *testing.T) {
	d, err := NewLLMQueryDecomposer(&mockDecomposerLLM{
		reply: "",
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test query")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0] != "test query" {
		t.Fatalf("expected fallback to original query, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T2.3: TestLLMQueryDecomposer_NonJSONResponse (AC-002)
func TestLLMQueryDecomposer_NonJSONResponse(t *testing.T) {
	d, err := NewLLMQueryDecomposer(&mockDecomposerLLM{
		reply: "I cannot answer this question.",
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test query")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0] != "test query" {
		t.Fatalf("expected fallback to original query, got %v", subs)
	}
}

// @sk-test sub-query-decomposition#T2.3: TestLLMQueryDecomposer_CustomPrompt (AC-002)
func TestLLMQueryDecomposer_CustomPrompt(t *testing.T) {
	d, err := NewLLMQueryDecomposer(&mockDecomposerLLM{
		reply: `["custom sub question"]`,
	}, "custom prompt %s")
	if err != nil {
		t.Fatal(err)
	}

	subs, err := d.Decompose(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub-query, got %d", len(subs))
	}
	if subs[0] != "custom sub question" {
		t.Fatalf("expected 'custom sub question', got %q", subs[0])
	}
}

// @sk-test sub-query-decomposition#T2.3: TestNewLLMQueryDecomposer_NilLLM (AC-002)
func TestNewLLMQueryDecomposer_NilLLM(t *testing.T) {
	_, err := NewLLMQueryDecomposer(nil, "")
	if err == nil {
		t.Fatal("expected error for nil llm")
	}
}
