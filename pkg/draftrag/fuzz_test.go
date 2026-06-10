package draftrag

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// @sk-test fuzz-property-tests#T1.2: FuzzSearchBuilderValidate — no panic on random question+topK combos
func FuzzSearchBuilderValidate(f *testing.F) {
	seeds := []struct {
		question string
		topK     int
	}{
		{"", 5},
		{"hello", 0},
		{"hello", -1},
		{"", -1},
		{"\x00", 5},
		{"   ", 5},
		{"\t\n", 5},
		{"кириллица", 5},
		{"hello", -2147483648},
		{"hello", 2147483647},
	}
	for _, s := range seeds {
		f.Add(s.question, s.topK)
	}

	store := vectorstore.NewInMemoryStore()
	p, err := NewPipeline(store, benchLLM{}, benchEmbedder{})
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, question string, topK int) {
		sb := p.Search(question).TopK(topK)
		_, _ = sb.validate()
	})
}
