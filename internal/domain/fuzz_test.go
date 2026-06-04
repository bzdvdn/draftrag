package domain

import (
	"testing"
)

// @sk-test fuzz-property-tests#T1.1: FuzzValidateDocument — no panic on random string inputs
func FuzzValidateDocument(f *testing.F) {
	seeds := []struct {
		id      string
		content string
	}{
		{"", "hello"},
		{"doc-1", ""},
		{"", ""},
		{"\x00", "hello"},
		{"doc-1", "\x00"},
		{"   ", "hello"},
		{"doc-1", "   "},
		{"кириллица", "unicode content"},
		{"😀", "emoji title"},
		{string(rune(0)), "null rune"},
	}
	for _, s := range seeds {
		f.Add(s.id, s.content)
	}

	f.Fuzz(func(t *testing.T, id, content string) {
		doc := Document{ID: id, Content: content}
		_ = doc.Validate()
	})
}

// @sk-test fuzz-property-tests#T1.1: FuzzValidateChunk — no panic on random string inputs
func FuzzValidateChunk(f *testing.F) {
	seeds := []struct {
		id       string
		content  string
		parentID string
	}{
		{"", "content", "parent"},
		{"chunk-1", "", "parent"},
		{"chunk-1", "content", ""},
		{"", "", ""},
		{"\x00", "content", "parent"},
		{"chunk-1", "\x00", "parent"},
		{"chunk-1", "content", "\x00"},
		{"   ", "content", "parent"},
		{"chunk-1", "   ", "parent"},
		{"chunk-1", "content", "   "},
	}
	for _, s := range seeds {
		f.Add(s.id, s.content, s.parentID)
	}

	f.Fuzz(func(t *testing.T, id, content, parentID string) {
		chunk := Chunk{ID: id, Content: content, ParentID: parentID}
		_ = chunk.Validate()
	})
}

// @sk-test fuzz-property-tests#T1.1: FuzzValidateQuery — no panic on random text+topK
func FuzzValidateQuery(f *testing.F) {
	seeds := []struct {
		text string
		topK int
	}{
		{"", 5},
		{"hello", 0},
		{"hello", -1},
		{"", 0},
		{"\x00", 5},
		{"hello", -2147483648},
		{"hello", 2147483647},
		{"   ", 5},
	}
	for _, s := range seeds {
		f.Add(s.text, s.topK)
	}

	f.Fuzz(func(t *testing.T, text string, topK int) {
		q := Query{Text: text, TopK: topK}
		_ = q.Validate()
	})
}
