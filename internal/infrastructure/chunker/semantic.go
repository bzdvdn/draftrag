package chunker

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/bzdvdn/draftrag/internal/domain"
)

var sentenceExceptions = map[string]bool{
	"dr": true, "mr": true, "mrs": true, "ms": true,
	"e.g": true, "i.e": true, "vs": true, "etc": true,
}

func isCJKPunct(ch rune) bool {
	return ch == '。' || ch == '！' || ch == '？'
}

// @sk-task chunker-semantic#T1.1: sentence splitter (AC-005)
// @sk-task cjk-tokenization#T1.1: CJK punctuation delimiters (AC-001, AC-002)
func splitSentences(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	runes := []rune(trimmed)
	var sentences []string
	start := 0

	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch != '.' && ch != '!' && ch != '?' && !isCJKPunct(ch) {
			continue
		}

		if ch == '.' && isAbbreviation(runes, start, i) {
			continue
		}

		if !isSentenceBoundary(runes, i, ch) {
			continue
		}

		end := i + 1
		sentences = append(sentences, strings.TrimSpace(string(runes[start:end])))
		start = end
	}

	if start < len(runes) {
		remaining := strings.TrimSpace(string(runes[start:]))
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}

	return sentences
}

func isAbbreviation(runes []rune, start, dotIdx int) bool {
	if dotIdx <= start {
		return false
	}

	wordStart := dotIdx - 1
	for wordStart > start && runes[wordStart-1] != ' ' && runes[wordStart-1] != '\t' && runes[wordStart-1] != '\n' && runes[wordStart-1] != '\r' {
		wordStart--
	}

	if wordStart >= dotIdx {
		return false
	}

	word := strings.ToLower(string(runes[wordStart:dotIdx]))
	return sentenceExceptions[word]
}

// @sk-task cjk-tokenization#T1.2: CJK sentence boundary detection (AC-003)
func isSentenceBoundary(runes []rune, idx int, delim rune) bool {
	if isCJKPunct(delim) {
		return true
	}

	next := idx + 1

	if next >= len(runes) {
		return true
	}

	ch := runes[next]
	if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
		j := next
		for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t' || runes[j] == '\n' || runes[j] == '\r') {
			j++
		}
		if j >= len(runes) {
			return true
		}
		if unicode.IsUpper(runes[j]) || runes[j] == '"' || runes[j] == '«' || runes[j] == '`' {
			return true
		}
	}

	return false
}

// SemanticChunkerOptions — параметры семантического чанкера (post-validation).
type SemanticChunkerOptions struct {
	Embedder            domain.Embedder
	SimilarityThreshold float64
	MinChunkSize        int
	MaxChunkSize        int
}

// @sk-task chunker-semantic#T2.1: semantic chunker algorithm (AC-001–AC-006, AC-008)
type SemanticChunker struct {
	embedder            domain.Embedder
	similarityThreshold float64
	minChunkSize        int
	maxChunkSize        int
}

func NewSemanticChunker(opts SemanticChunkerOptions) *SemanticChunker {
	return &SemanticChunker{
		embedder:            opts.Embedder,
		similarityThreshold: opts.SimilarityThreshold,
		minChunkSize:        opts.MinChunkSize,
		maxChunkSize:        opts.MaxChunkSize,
	}
}

func (c *SemanticChunker) Chunk(ctx context.Context, doc domain.Document) ([]domain.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sentences := splitSentences(doc.Content)
	if len(sentences) == 0 {
		return []domain.Chunk{}, nil
	}

	var chunks []domain.Chunk
	var current []string
	var prevEmb []float64
	position := 0

	for _, s := range sentences {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		candidate := make([]string, len(current)+1)
		copy(candidate, current)
		candidate[len(current)] = s
		candidateText := strings.Join(candidate, " ")
		candidateRuneLen := len([]rune(candidateText))

		if c.maxChunkSize > 0 && candidateRuneLen > c.maxChunkSize && len(current) > 0 {
			chunks = append(chunks, buildChunk(doc, current, position))
			position++
			current = []string{s}
			emb, err := c.embedder.Embed(ctx, s)
			if err != nil {
				return nil, err
			}
			prevEmb = emb
			continue
		}

		if c.minChunkSize > 0 && candidateRuneLen < c.minChunkSize {
			current = candidate
			continue
		}

		emb, err := c.embedder.Embed(ctx, candidateText)
		if err != nil {
			return nil, err
		}

		if prevEmb == nil {
			current = candidate
			prevEmb = emb
			continue
		}

		sim := cosineSimilarity(emb, prevEmb)
		if sim >= c.similarityThreshold {
			current = candidate
			prevEmb = emb
		} else {
			chunks = append(chunks, buildChunk(doc, current, position))
			position++
			current = []string{s}
			emb, err := c.embedder.Embed(ctx, s)
			if err != nil {
				return nil, err
			}
			prevEmb = emb
		}
	}

	if len(current) > 0 {
		chunks = append(chunks, buildChunk(doc, current, position))
	}

	return chunks, nil
}

func buildChunk(doc domain.Document, sentences []string, position int) domain.Chunk {
	return domain.Chunk{
		ID:       fmt.Sprintf("%s:%d", doc.ID, position),
		Content:  strings.Join(sentences, " "),
		ParentID: doc.ID,
		Position: position,
	}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
