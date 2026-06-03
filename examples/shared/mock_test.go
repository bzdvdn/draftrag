package shared

import (
	"context"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test docs-and-examples#T1.5: TestMockLLM_Echo проверяет echo с префиксом "[mock] " (AC-008).
func TestMockLLM_Echo(t *testing.T) {
	llm := NewMockLLM()
	out, err := llm.Generate(context.Background(), "system", "Что такое goroutine?")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if want := "[mock] echo answer for: Что такое goroutine?"; out != want {
		t.Errorf("Generate() = %q, want %q", out, want)
	}
}

// @sk-test docs-and-examples#T1.5: TestMockLLM_Truncation проверяет обрезку длинных вопросов.
func TestMockLLM_Truncation(t *testing.T) {
	llm := NewMockLLM()
	long := ""
	for i := 0; i < 500; i++ {
		long += "a"
	}
	out, err := llm.Generate(context.Background(), "system", long)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if !strings.Contains(out, "...") {
		t.Errorf("Generate() = %q, want truncation marker '...'", out)
	}
}

// @sk-test docs-and-examples#T1.5: TestMockLLM_ImplementsInterface — compile-time + runtime check.
func TestMockLLM_ImplementsInterface(t *testing.T) {
	var _ domain.LLMProvider = (*mockLLM)(nil)
	var llm domain.LLMProvider = &mockLLM{}
	if _, err := llm.Generate(context.Background(), "", "test"); err != nil {
		t.Errorf("Generate() error: %v", err)
	}
}

// @sk-test docs-and-examples#T1.5: TestMockEmbedder_Determinism проверяет детерминизм (DEC-007, AC-008).
func TestMockEmbedder_Determinism(t *testing.T) {
	emb := NewMockEmbedder(128)
	v1, err := emb.Embed(context.Background(), "goroutines")
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}
	v2, err := emb.Embed(context.Background(), "goroutines")
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}
	if len(v1) != 128 {
		t.Errorf("len(v1) = %d, want 128", len(v1))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("v1[%d] != v2[%d] (%f vs %f): mock not deterministic", i, i, v1[i], v2[i])
			break
		}
	}
}

// @sk-test docs-and-examples#T1.5: TestMockEmbedder_DifferentInputs проверяет, что разные тексты дают разные векторы.
func TestMockEmbedder_DifferentInputs(t *testing.T) {
	emb := NewMockEmbedder(64)
	v1, _ := emb.Embed(context.Background(), "foo")
	v2, _ := emb.Embed(context.Background(), "bar")
	differ := false
	for i := range v1 {
		if v1[i] != v2[i] {
			differ = true
			break
		}
	}
	if !differ {
		t.Error("v1 == v2 for different inputs: mock should produce different vectors")
	}
}

// @sk-test docs-and-examples#T1.5: TestMockEmbedder_ImplementsInterface — compile-time + runtime check.
func TestMockEmbedder_ImplementsInterface(t *testing.T) {
	var _ domain.Embedder = (*mockEmbedder)(nil)
	var emb domain.Embedder = &mockEmbedder{dim: 8}
	v, err := emb.Embed(context.Background(), "x")
	if err != nil {
		t.Errorf("Embed() error: %v", err)
	}
	if len(v) != 8 {
		t.Errorf("len(v) = %d, want 8", len(v))
	}
}
