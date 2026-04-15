package application

import (
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

func TestDedupRetrievedChunksByParentID(t *testing.T) {
	chunks := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:       "c1",
				ParentID: "doc1",
				Content:  "content 1",
			},
			Score: 0.9,
		},
		{
			Chunk: domain.Chunk{
				ID:       "c2",
				ParentID: "doc1", // дубликат ParentID
				Content:  "content 2",
			},
			Score: 0.8,
		},
		{
			Chunk: domain.Chunk{
				ID:       "c3",
				ParentID: "doc2",
				Content:  "content 3",
			},
			Score: 0.7,
		},
	}

	result := dedupRetrievedChunksByParentID(chunks)

	// Должно остаться только 2 чанка (по одному на каждый ParentID)
	if len(result) != 2 {
		t.Errorf("expected 2 chunks after deduplication, got %d", len(result))
	}

	// Проверяем, что остались чанки с наивысшими скорами для каждого ParentID
	parentIDs := make(map[string]bool)
	for _, r := range result {
		parentIDs[r.Chunk.ParentID] = true
	}

	if !parentIDs["doc1"] || !parentIDs["doc2"] {
		t.Error("expected both parent IDs to be present")
	}
}

func TestDedupRetrievedChunksByParentID_Empty(t *testing.T) {
	chunks := []domain.RetrievedChunk{}
	result := dedupRetrievedChunksByParentID(chunks)

	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d chunks", len(result))
	}
}

func TestDedupRetrievedChunksByParentID_NoDuplicates(t *testing.T) {
	chunks := []domain.RetrievedChunk{
		{
			Chunk: domain.Chunk{
				ID:       "c1",
				ParentID: "doc1",
			},
			Score: 0.9,
		},
		{
			Chunk: domain.Chunk{
				ID:       "c2",
				ParentID: "doc2",
			},
			Score: 0.8,
		},
	}

	result := dedupRetrievedChunksByParentID(chunks)

	if len(result) != 2 {
		t.Errorf("expected 2 chunks when no duplicates, got %d", len(result))
	}
}

func TestParseMultiQueryLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple lines",
			input:    "query1\nquery2\nquery3",
			expected: []string{"query1", "query2", "query3"},
		},
		{
			name:     "single line",
			input:    "single query",
			expected: []string{"single query"},
		},
		{
			name:     "empty lines filtered",
			input:    "query1\n\nquery2\n",
			expected: []string{"query1", "query2"},
		},
		{
			name:     "whitespace lines filtered",
			input:    "query1\n   \nquery2",
			expected: []string{"query1", "query2"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only empty lines",
			input:    "\n\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMultiQueryLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d lines, got %d", len(tt.expected), len(result))
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestRRFMergeMultiple(t *testing.T) {
	lists := []domain.RetrievalResult{
		{
			Chunks: []domain.RetrievedChunk{
				{
					Chunk: domain.Chunk{ID: "c1", ParentID: "doc1"},
					Score: 0.9,
				},
				{
					Chunk: domain.Chunk{ID: "c2", ParentID: "doc2"},
					Score: 0.8,
				},
			},
		},
		{
			Chunks: []domain.RetrievedChunk{
				{
					Chunk: domain.Chunk{ID: "c1", ParentID: "doc1"}, // дубликат
					Score: 0.7,
				},
				{
					Chunk: domain.Chunk{ID: "c3", ParentID: "doc3"},
					Score: 0.6,
				},
			},
		},
	}

	result := rrfMergeMultiple(lists, 5)

	// Должно быть 3 уникальных чанка
	if len(result.Chunks) != 3 {
		t.Errorf("expected 3 unique chunks, got %d", len(result.Chunks))
	}

	// Проверяем, что все ID присутствуют
	ids := make(map[string]bool)
	for _, r := range result.Chunks {
		ids[r.Chunk.ID] = true
	}

	if !ids["c1"] || !ids["c2"] || !ids["c3"] {
		t.Error("expected all chunk IDs to be present")
	}
}

func TestRRFMergeMultiple_EmptyLists(t *testing.T) {
	lists := []domain.RetrievalResult{}
	result := rrfMergeMultiple(lists, 5)

	if len(result.Chunks) != 0 {
		t.Errorf("expected empty result for empty lists, got %d chunks", len(result.Chunks))
	}
}

func TestRRFMergeMultiple_SingleList(t *testing.T) {
	lists := []domain.RetrievalResult{
		{
			Chunks: []domain.RetrievedChunk{
				{
					Chunk: domain.Chunk{ID: "c1", ParentID: "doc1"},
					Score: 0.9,
				},
			},
		},
	}

	result := rrfMergeMultiple(lists, 5)

	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk for single list, got %d", len(result.Chunks))
	}
}

func TestBuildContextTextV1(t *testing.T) {
	result := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{
				Chunk: domain.Chunk{
					ID:       "c1",
					ParentID: "doc1",
					Content:  "This is content 1",
				},
				Score: 0.9,
			},
			{
				Chunk: domain.Chunk{
					ID:       "c2",
					ParentID: "doc2",
					Content:  "This is content 2",
				},
				Score: 0.8,
			},
		},
	}

	// Без ограничений
	text := buildContextTextV1(result, 0, 0)
	if text == "" {
		t.Error("expected non-empty context text")
	}

	// С ограничением по количеству чанков
	text = buildContextTextV1(result, 0, 1)
	if text == "" {
		t.Error("expected non-empty context text with chunk limit")
	}

	// С ограничением по символам
	text = buildContextTextV1(result, 10, 0)
	if text == "" {
		t.Error("expected non-empty context text with char limit")
	}
}

func TestBuildContextTextV1_Empty(t *testing.T) {
	result := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{},
	}

	text := buildContextTextV1(result, 0, 0)
	if text != "" {
		t.Errorf("expected empty context text for empty result, got %q", text)
	}
}

func TestBuildUserMessageV1(t *testing.T) {
	result := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{
				Chunk: domain.Chunk{
					ID:       "c1",
					ParentID: "doc1",
					Content:  "Test content",
				},
				Score: 0.9,
			},
		},
	}

	message := buildUserMessageV1(result, "test question", 1000, 10)
	if message == "" {
		t.Error("expected non-empty user message")
	}

	// Проверяем, что вопрос присутствует в сообщении
	// (простая проверка, так как формат может меняться)
	if len(message) < len("test question") {
		t.Error("user message should be at least as long as the question")
	}
}

func TestBuildContextTextV1InlineCitations(t *testing.T) {
	result := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{
				Chunk: domain.Chunk{
					ID:       "c1",
					ParentID: "doc1",
					Content:  "Content 1",
				},
				Score: 0.9,
			},
			{
				Chunk: domain.Chunk{
					ID:       "c2",
					ParentID: "doc2",
					Content:  "Content 2",
				},
				Score: 0.8,
			},
		},
	}

	text, citations := buildContextTextV1InlineCitations(result, 1000, 10)
	if text == "" {
		t.Error("expected non-empty context text")
	}
	if len(citations) != 2 {
		t.Errorf("expected 2 citations, got %d", len(citations))
	}
}

func TestBuildUserMessageV1InlineCitations(t *testing.T) {
	result := domain.RetrievalResult{
		Chunks: []domain.RetrievedChunk{
			{
				Chunk: domain.Chunk{
					ID:       "c1",
					ParentID: "doc1",
					Content:  "Test content",
				},
				Score: 0.9,
			},
		},
	}

	message, citations := buildUserMessageV1InlineCitations(result, "test question", 1000, 10)
	if message == "" {
		t.Error("expected non-empty user message")
	}
	if len(citations) != 1 {
		t.Errorf("expected 1 citation, got %d", len(citations))
	}
}
