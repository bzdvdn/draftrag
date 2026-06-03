package shared

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// @sk-test docs-and-examples#T1.5: TestPrintAnswer_EmptySources проверяет формат с 0 источниками (AC-006).
func TestPrintAnswer_EmptySources(t *testing.T) {
	out := captureStdout(t, func() {
		PrintAnswer("Q?", "A.", nil)
	})
	if !strings.Contains(out, "[Q] Q?") {
		t.Errorf("output missing [Q] Q?: %q", out)
	}
	if !strings.Contains(out, "[A] A.") {
		t.Errorf("output missing [A] A.: %q", out)
	}
	if !strings.Contains(out, "[Sources] 0 chunks") {
		t.Errorf("output missing [Sources] 0 chunks: %q", out)
	}
}

// @sk-test docs-and-examples#T1.5: TestPrintAnswer_WithSources проверяет формат с источниками (AC-006).
func TestPrintAnswer_WithSources(t *testing.T) {
	sources := []draftrag.RetrievalResult{
		{
			Chunks: []domain.RetrievedChunk{
				{Chunk: domain.Chunk{ParentID: "doc-1"}, Score: 0.95},
				{Chunk: domain.Chunk{ParentID: "doc-2"}, Score: 0.80},
			},
		},
	}
	out := captureStdout(t, func() {
		PrintAnswer("Q?", "A.", sources)
	})
	if !strings.Contains(out, "[Sources] 2 chunks") {
		t.Errorf("output missing source count: %q", out)
	}
	if !strings.Contains(out, "parent_id=doc-1") {
		t.Errorf("output missing parent_id=doc-1: %q", out)
	}
}

// @sk-test docs-and-examples#T1.5: TestPrintAnswer_Truncation проверяет обрезку длинных вопросов.
func TestPrintAnswer_Truncation(t *testing.T) {
	long := strings.Repeat("x", 500)
	out := captureStdout(t, func() {
		PrintAnswer(long, "A.", nil)
	})
	if !strings.Contains(out, "...") {
		t.Errorf("output missing truncation marker: %q", out)
	}
}

// @sk-test docs-and-examples#T1.5: TestPrintInfo проверяет [info] prefix.
func TestPrintInfo(t *testing.T) {
	out := captureStdout(t, func() {
		PrintInfo("indexed %d documents", 10)
	})
	if !strings.Contains(out, "[info] indexed 10 documents") {
		t.Errorf("output: %q", out)
	}
}

// @sk-test docs-and-examples#T1.5: TestPrintError проверяет [error] prefix в stderr.
func TestPrintError(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintError("connection failed: %s", "timeout")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "[error] connection failed: timeout") {
		t.Errorf("stderr: %q", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
