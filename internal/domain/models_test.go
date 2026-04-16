package domain

import (
	"context"
	"testing"
)

func TestDocumentValidate_EmptyContent(t *testing.T) {
	doc := Document{
		ID:      "doc-1",
		Content: "",
	}

	if err := doc.Validate(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDocumentValidate_EmptyID(t *testing.T) {
	doc := Document{
		ID:      "",
		Content: "content",
	}

	err := doc.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrEmptyDocumentID {
		t.Errorf("expected ErrEmptyDocumentID, got %v", err)
	}
}

func TestDocumentValidate_Valid(t *testing.T) {
	doc := Document{
		ID:      "doc-1",
		Content: "content",
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDocumentValidate_WithMetadata(t *testing.T) {
	doc := Document{
		ID:      "doc-1",
		Content: "content",
		Metadata: map[string]string{
			"source": "wiki",
			"lang":   "ru",
		},
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("expected no error with metadata, got %v", err)
	}
}

func TestChunkValidate_EmptyParentID(t *testing.T) {
	chunk := Chunk{
		ID:       "chunk-1",
		Content:  "hello",
		ParentID: "",
		Position: 0,
	}

	if err := chunk.Validate(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestChunkValidate_EmptyID(t *testing.T) {
	chunk := Chunk{
		ID:       "",
		Content:  "hello",
		ParentID: "doc1",
		Position: 0,
	}

	err := chunk.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrEmptyChunkID {
		t.Errorf("expected ErrEmptyChunkID, got %v", err)
	}
}

func TestChunkValidate_EmptyContent(t *testing.T) {
	chunk := Chunk{
		ID:       "chunk-1",
		Content:  "",
		ParentID: "doc1",
		Position: 0,
	}

	err := chunk.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrEmptyChunkContent {
		t.Errorf("expected ErrEmptyChunkContent, got %v", err)
	}
}

func TestChunkValidate_Valid(t *testing.T) {
	chunk := Chunk{
		ID:       "chunk-1",
		Content:  "hello",
		ParentID: "doc1",
		Position: 0,
	}

	if err := chunk.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestChunkValidate_WithMetadata(t *testing.T) {
	chunk := Chunk{
		ID:       "chunk-1",
		Content:  "hello",
		ParentID: "doc1",
		Position: 0,
		Metadata: map[string]string{
			"source": "wiki",
		},
	}

	if err := chunk.Validate(); err != nil {
		t.Fatalf("expected no error with metadata, got %v", err)
	}
}

func TestChunkValidate_WithEmbedding(t *testing.T) {
	chunk := Chunk{
		ID:        "chunk-1",
		Content:   "hello",
		ParentID:  "doc1",
		Position:  0,
		Embedding: []float64{0.1, 0.2, 0.3},
	}

	if err := chunk.Validate(); err != nil {
		t.Fatalf("expected no error with embedding, got %v", err)
	}
}

func TestQueryValidate_EmptyText(t *testing.T) {
	query := Query{
		Text: "",
		TopK: 5,
	}

	err := query.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrEmptyQueryText {
		t.Errorf("expected ErrEmptyQueryText, got %v", err)
	}
}

func TestQueryValidate_InvalidTopK(t *testing.T) {
	query := Query{
		Text: "test",
		TopK: 0,
	}

	err := query.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrInvalidQueryTopK {
		t.Errorf("expected ErrInvalidQueryTopK, got %v", err)
	}
}

func TestQueryValidate_NegativeTopK(t *testing.T) {
	query := Query{
		Text: "test",
		TopK: -1,
	}

	if err := query.Validate(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestQueryValidate_Valid(t *testing.T) {
	query := Query{
		Text: "test query",
		TopK: 5,
	}

	if err := query.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQueryValidate_WithFilter(t *testing.T) {
	query := Query{
		Text: "test query",
		TopK: 5,
		Filter: map[string]string{
			"category": "tech",
		},
	}

	if err := query.Validate(); err != nil {
		t.Fatalf("expected no error with filter, got %v", err)
	}
}

func TestDefaultHybridConfig(t *testing.T) {
	cfg := DefaultHybridConfig()

	if cfg.SemanticWeight != 0.7 {
		t.Errorf("expected SemanticWeight=0.7, got %f", cfg.SemanticWeight)
	}
	if !cfg.UseRRF {
		t.Error("expected UseRRF=true")
	}
	if cfg.RRFK != 60 {
		t.Errorf("expected RRFK=60, got %d", cfg.RRFK)
	}
	if cfg.BMFinalK != 0 {
		t.Errorf("expected BMFinalK=0, got %d", cfg.BMFinalK)
	}

	// Дефолтная конфигурация должна проходить валидацию
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestHybridConfig_Validate_SemanticWeight(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		valid  bool
	}{
		{"negative", -0.1, false},
		{"zero", 0.0, true},
		{"one", 1.0, true},
		{"valid", 0.7, true},
		{"greater than one", 1.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HybridConfig{SemanticWeight: tt.weight, RRFK: 60, BMFinalK: 0}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestHybridConfig_Validate_RRFK(t *testing.T) {
	tests := []struct {
		name  string
		rrfk  int
		valid bool
	}{
		{"zero", 0, false},
		{"negative", -1, false},
		{"one", 1, true},
		{"default", 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HybridConfig{SemanticWeight: 0.7, RRFK: tt.rrfk, BMFinalK: 0}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestHybridConfig_Validate_BMFinalK(t *testing.T) {
	tests := []struct {
		name     string
		bmFinaKK int
		valid    bool
	}{
		{"negative", -1, false},
		{"zero", 0, true},
		{"positive", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HybridConfig{SemanticWeight: 0.7, RRFK: 60, BMFinalK: tt.bmFinaKK}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestNoopLogger(t *testing.T) {
	t.Parallel()

	logger := NoopLogger()
	ctx := context.Background()

	// NoopLogger не должен паниковать при вызове Log
	logger.Log(ctx, LogLevelInfo, "test message", LogField{Key: "key", Value: "value"})
}

func TestSafeLog_NilLogger(t *testing.T) {
	t.Parallel()

	// SafeLog с nil logger не должен паниковать
	SafeLog(context.Background(), nil, LogLevelInfo, "test", LogField{Key: "k", Value: "v"})
}

func TestSafeLog_PanicInLogger(t *testing.T) {
	t.Parallel()

	panicLogger := struct {
		Logger
	}{
		Logger: panicLoggerImpl{},
	}

	// SafeLog должен защищать от паники в логгере
	SafeLog(context.Background(), panicLogger, LogLevelInfo, "test", LogField{Key: "k", Value: "v"})
}

type panicLoggerImpl struct{}

func (panicLoggerImpl) Log(_ context.Context, _ LogLevel, _ string, _ ...LogField) {
	panic("logger panic")
}

func TestRedactSecret(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		secret string
		want   string
	}{
		{"basic replacement", "my secret is abc123", "abc123", "my secret is <redacted>"},
		{"multiple occurrences", "abc123 and abc123 again", "abc123", "<redacted> and <redacted> again"},
		{"empty secret", "my secret is abc123", "", "my secret is abc123"},
		{"whitespace secret", "my secret is abc123", "   ", "my secret is abc123"},
		{"secret not found", "my secret is xyz", "abc123", "my secret is xyz"},
		{"empty text", "", "abc123", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactSecret(tt.text, tt.secret)
			if got != tt.want {
				t.Errorf("RedactSecret() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		secrets []string
		want    string
	}{
		{"multiple secrets", "key1=abc and key2=def", []string{"abc", "def"}, "key1=<redacted> and key2=<redacted>"},
		{"empty secrets list", "key1=abc", []string{}, "key1=abc"},
		{"nil secrets", "key1=abc", nil, "key1=abc"},
		{"overlapping secrets", "abc123 and abc456", []string{"abc", "123"}, "<redacted><redacted> and <redacted>456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactSecrets(tt.text, tt.secrets...)
			if got != tt.want {
				t.Errorf("RedactSecrets() = %q, want %q", got, tt.want)
			}
		})
	}
}
