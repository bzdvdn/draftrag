package chunker

import (
	"context"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test cjk-tokenization#T2.1: AC-001 Chinese period split (AC-001)
func TestCJK_SplitSentences_Chinese(t *testing.T) {
	text := "今天天气很好。我们去公园散步。明天还会下雨。"
	sentences := splitSentences(text)
	if len(sentences) != 3 {
		t.Fatalf("expected 3 sentences, got %d: %v", len(sentences), sentences)
	}
	for i, s := range sentences {
		if !strings.HasSuffix(s, "。") && i < len(sentences)-1 {
			t.Errorf("sentence %d should end with 。, got %q", i, s)
		}
	}
}

// @sk-test cjk-tokenization#T2.1: AC-002 Japanese exclamation and question (AC-002)
func TestCJK_SplitSentences_Japanese(t *testing.T) {
	text := "こんにちは！今日はいい天気ですね？明日は何をしますか？"
	sentences := splitSentences(text)
	if len(sentences) != 3 {
		t.Fatalf("expected 3 sentences, got %d: %v", len(sentences), sentences)
	}
	if !strings.Contains(sentences[0], "！") {
		t.Errorf("sentence 0 should contain ！, got %q", sentences[0])
	}
	if !strings.Contains(sentences[1], "？") {
		t.Errorf("sentence 1 should contain ？, got %q", sentences[1])
	}
	if !strings.Contains(sentences[2], "？") {
		t.Errorf("sentence 2 should contain ？, got %q", sentences[2])
	}
}

// @sk-test cjk-tokenization#T2.2: AC-003 CJK sentence boundary (AC-003)
func TestCJK_SentenceBoundary(t *testing.T) {
	text := "今天天气很好。我们去公园散步。"
	runes := []rune(text)
	// Find the position of first 。
	firstPunct := -1
	for i, ch := range runes {
		if ch == '。' {
			firstPunct = i
			break
		}
	}
	if firstPunct < 0 {
		t.Fatal("expected to find 。 in text")
	}
	if !isSentenceBoundary(runes, firstPunct, '。') {
		t.Error("expected isSentenceBoundary to return true for 。 followed by CJK char")
	}
}

// @sk-test cjk-tokenization#T2.1: AC-005 Latin text no regression (AC-005)
func TestCJK_SplitSentences_LatinRegression(t *testing.T) {
	text := "Hello world. This is a test! Is it working? Yes."
	sentences := splitSentences(text)
	if len(sentences) != 4 {
		t.Fatalf("expected 4 sentences, got %d: %v", len(sentences), sentences)
	}
}

// @sk-test cjk-tokenization#T2.1: mixed CJK/Latin split
func TestCJK_SplitSentences_Mixed(t *testing.T) {
	text := "Hello世界。How are you？Good。"
	sentences := splitSentences(text)
	if len(sentences) != 3 {
		t.Fatalf("expected 3 sentences, got %d: %v", len(sentences), sentences)
	}
}

// @sk-test cjk-tokenization#T2.1: CJK text without punctuation
func TestCJK_SplitSentences_NoPunctuation(t *testing.T) {
	text := "今天天气很好我们去公园散步"
	sentences := splitSentences(text)
	if len(sentences) != 1 {
		t.Fatalf("expected 1 sentence, got %d: %v", len(sentences), sentences)
	}
}

// @sk-test cjk-tokenization#T2.1: empty CJK text
func TestCJK_SplitSentences_Empty(t *testing.T) {
	sentences := splitSentences("")
	if sentences != nil {
		t.Fatalf("expected nil, got %v", sentences)
	}
}

// @sk-test cjk-tokenization#T2.3: AC-004 SemanticChunker produces multiple chunks (AC-004)
func TestCJK_SemanticChunker_MultipleChunks(t *testing.T) {
	text := "今天天气很好。我们去公园散步。明天还会下雨。请带伞。"
	doc := domain.Document{ID: "cjk-test", Content: text}
	chunker := &SemanticChunker{
		embedder:            &mockIdentityEmbedder{},
		similarityThreshold: 1.1,
		minChunkSize:        1,
		maxChunkSize:        200,
	}
	chunks, err := chunker.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected 2+ chunks for 4 CJK sentences, got %d", len(chunks))
	}
}

// mockIdentityEmbedder returns the same vector for any input.
type mockIdentityEmbedder struct{}

func (m *mockIdentityEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{1.0}, nil
}

func (m *mockIdentityEmbedder) Health(_ context.Context) error { return nil }

// @sk-test cjk-tokenization#T2.4: AC-006 BasicChunker CJK rune split (AC-006)
func TestCJK_BasicChunker_RuneSplit(t *testing.T) {
	text := "今天天气很好我们去公园"
	doc := domain.Document{ID: "cjk-basic", Content: text}
	ch := NewBasicRuneChunker(5, 0, 10)
	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	for _, chunk := range chunks {
		if chunk.Content == "" {
			t.Error("chunk content should not be empty")
		}
	}
}

// @sk-test cjk-tokenization#T2.4: BasicChunker CJK with overlap
func TestCJK_BasicChunker_RuneSplitWithOverlap(t *testing.T) {
	text := "今日はいい天気ですね。明日も晴れるでしょう。"
	doc := domain.Document{ID: "cjk-overlap", Content: text}
	ch := NewBasicRuneChunker(8, 2, 10)
	chunks, err := ch.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk")
	}
	for _, chunk := range chunks {
		runes := []rune(chunk.Content)
		if runes[0] == 0 || len(runes) == 0 {
			t.Error("chunk content should start with valid CJK rune")
		}
	}
}
