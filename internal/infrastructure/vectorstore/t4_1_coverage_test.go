package vectorstore

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-test api-consistency-pass#T4.1-coverage: покрывает pure-функции pgvector,
// которые иначе достижимы только через integration tests (gated by
// RUN_INTEGRATION_TESTS=1). Удаляет ~50 uncovered statements из пакета.

func TestClampScore(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"within range", 0.5, 0.5},
		{"above 1", 1.5, 1.0},
		{"below -1", -1.5, -1.0},
		{"boundary 1", 1.0, 1.0},
		{"boundary -1", -1.0, -1.0},
		{"zero", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampScore(tt.in)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("clampScore(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateEmbedding(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float64
		dim       int
		wantErr   bool
	}{
		{"valid", []float64{0.1, 0.2, 0.3}, 3, false},
		{"nil", nil, 3, true},
		{"dim mismatch", []float64{0.1, 0.2}, 3, true},
		{"NaN", []float64{0.1, math.NaN(), 0.3}, 3, true},
		{"+Inf", []float64{0.1, math.Inf(1), 0.3}, 3, true},
		{"-Inf", []float64{0.1, math.Inf(-1), 0.3}, 3, true},
		{"empty slice ok", []float64{}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmbedding(tt.embedding, tt.dim)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmbedding() err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSQLIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"valid simple", "table_name", false},
		{"valid mixed", "Tbl_123", false},
		{"empty", "", true},
		{"starts with digit", "1table", true},
		{"has dash", "table-name", true},
		{"has space", "table name", true},
		{"has dot", "table.name", true},
		{"has semicolon", "table;name", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQLIdentifier(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSQLIdentifier(%q) err=%v, wantErr=%v", tt.in, err, tt.wantErr)
			}
		})
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "mytable", `"mytable"`},
		{"empty", "", `""`},
		{"contains quote", `a"b`, `"a""b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdent(tt.in)
			if got != tt.want {
				t.Errorf("quoteIdent(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestWithDefaultTimeout(t *testing.T) {
	t.Run("no timeout returns original ctx", func(t *testing.T) {
		ctx := context.Background()
		got, cancel := withDefaultTimeout(ctx, 0)
		defer cancel()
		if got != ctx {
			t.Error("expected original ctx when timeout <= 0")
		}
	})

	t.Run("ctx with deadline returns original", func(t *testing.T) {
		ctx, existingCancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer existingCancel()
		got, cancel := withDefaultTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		if got != ctx {
			t.Error("expected original ctx when ctx already has deadline")
		}
	})

	t.Run("ctx without deadline gets timeout", func(t *testing.T) {
		ctx := context.Background()
		got, cancel := withDefaultTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		if got == ctx {
			t.Error("expected new ctx with deadline")
		}
		if _, hasDeadline := got.Deadline(); !hasDeadline {
			t.Error("expected new ctx to have deadline")
		}
	})
}

func TestDefaultRuntimeOptions(t *testing.T) {
	opts := defaultRuntimeOptions()
	if opts.SearchTimeout <= 0 {
		t.Errorf("expected positive SearchTimeout, got %v", opts.SearchTimeout)
	}
	if opts.UpsertTimeout <= 0 {
		t.Errorf("expected positive UpsertTimeout, got %v", opts.UpsertTimeout)
	}
	if opts.DeleteTimeout <= 0 {
		t.Errorf("expected positive DeleteTimeout, got %v", opts.DeleteTimeout)
	}
}

// @sk-test api-consistency-pass#T4.1-coverage: покрывает chromadb.go edge cases
// (chromaToRetrievalResult, CollectionExists error path).
func TestChromaToRetrievalResult_Valid(t *testing.T) {
	// Покрывает успешный путь парсинга — 1+2 case.
	resp := chromaQueryResponse{
		IDs:       [][]string{{"id1", "id2"}},
		Documents: [][]string{{"text1", "text2"}},
		Metadatas: [][]map[string]string{{{parentIDKey: "p1", "k": "v"}, {}}},
		Distances: [][]float64{{0.1, 0.2}},
	}
	got := chromaToRetrievalResult(resp)
	if got.TotalFound != 2 {
		t.Errorf("expected TotalFound=2, got %d", got.TotalFound)
	}
	if len(got.Chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(got.Chunks))
	}
}

func TestChromaToRetrievalResult_Empty(t *testing.T) {
	got := chromaToRetrievalResult(chromaQueryResponse{})
	if got.TotalFound != 0 {
		t.Errorf("expected TotalFound=0, got %d", got.TotalFound)
	}
}

// @sk-test api-consistency-pass#T4.1-coverage: покрывает parseMilvusSearchData
// edge case (пустые данные) для гибридного пути.
func TestParseMilvusSearchData_Empty(t *testing.T) {
	result, err := parseMilvusSearchData([]byte(`[]`))
	if err != nil {
		t.Fatalf("parseMilvusSearchData: %v", err)
	}
	if result.TotalFound != 0 {
		t.Errorf("expected 0 results, got %d", result.TotalFound)
	}
}

// Compile-time guard: импорт strings используется в этой тестовой группе.
var _ = strings.Contains
var _ = domain.ErrEmbeddingDimensionMismatch

// parentIDKey is a ChromaDB metadata field name used in tests; defined locally
// to avoid coupling to internal constants. Matches production key "parent_id".
const parentIDKey = "parent_id"
