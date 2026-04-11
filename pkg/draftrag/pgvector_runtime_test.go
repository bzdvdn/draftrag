package draftrag

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestNormalizePGVectorRuntimeOptions_Defaults(t *testing.T) {
	got := normalizePGVectorRuntimeOptions(PGVectorRuntimeOptions{})
	if got.SearchTimeout != 2*time.Second {
		t.Fatalf("SearchTimeout: expected 2s, got %v", got.SearchTimeout)
	}
	if got.UpsertTimeout != 5*time.Second {
		t.Fatalf("UpsertTimeout: expected 5s, got %v", got.UpsertTimeout)
	}
	if got.DeleteTimeout != 5*time.Second {
		t.Fatalf("DeleteTimeout: expected 5s, got %v", got.DeleteTimeout)
	}
	if got.MaxTopK != 50 {
		t.Fatalf("MaxTopK: expected 50, got %d", got.MaxTopK)
	}
	if got.MaxParentIDs != 128 {
		t.Fatalf("MaxParentIDs: expected 128, got %d", got.MaxParentIDs)
	}
	if got.MaxContentBytes != 0 {
		t.Fatalf("MaxContentBytes: expected 0, got %d", got.MaxContentBytes)
	}
}

func TestNewPGVectorStoreWithRuntimeOptions_PanicsOnInvalidLimits(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()

	// db не используется до panic, но проверка nil db была бы первой, поэтому используем non-nil.
	db := &sql.DB{}
	_ = NewPGVectorStoreWithRuntimeOptions(db, PGVectorOptions{EmbeddingDimension: 1}, PGVectorRuntimeOptions{MaxTopK: -1})
}

// @sk-task T4.2: Integration-тест SearchWithMetadataFilter в pgvector (AC-001, AC-004, DEC-002)

// TestPipeline_SearchWithMetadataFilter_Integration проверяет корректность фильтрации по метаданным
// в реальной БД PostgreSQL+pgvector.
// Тест пропускается, если переменная окружения PGVECTOR_TEST_DSN не задана.
func TestPipeline_SearchWithMetadataFilter_Integration(t *testing.T) {
	dsn := os.Getenv("PGVECTOR_TEST_DSN")
	if dsn == "" {
		t.Skip("PGVECTOR_TEST_DSN is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	tableName := "draftrag_meta_filter_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
	opts := PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: 2,
		CreateExtension:    true,
	}

	if err := MigratePGVector(ctx, db, PGVectorMigrateOptions{PGVectorOptions: opts}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+`"`+tableName+`"`)
	})

	store := NewPGVectorStoreWithRuntimeOptions(db, opts, PGVectorRuntimeOptions{
		MaxTopK:      50,
		MaxParentIDs: 128,
	})

	// Индексируем документы с разными категориями напрямую через store.
	chunks := []Chunk{
		{
			ID:        "legal-1#0",
			Content:   "legal document about contracts",
			ParentID:  "legal-1",
			Embedding: []float64{1, 0},
			Metadata:  map[string]string{"category": "legal"},
		},
		{
			ID:        "legal-2#0",
			Content:   "legal document about compliance",
			ParentID:  "legal-2",
			Embedding: []float64{0.9, 0.1},
			Metadata:  map[string]string{"category": "legal"},
		},
		{
			ID:        "finance-1#0",
			Content:   "finance document about budget",
			ParentID:  "finance-1",
			Embedding: []float64{0.8, 0.2},
			Metadata:  map[string]string{"category": "finance"},
		},
	}

	for _, c := range chunks {
		if err := store.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert chunk %s: %v", c.ID, err)
		}
	}

	vs, ok := store.(VectorStoreWithFilters)
	if !ok {
		t.Fatal("store does not implement VectorStoreWithFilters")
	}

	// AC-001: фильтр category=legal возвращает только legal-чанки.
	result, err := vs.SearchWithMetadataFilter(ctx, []float64{1, 0}, 10, MetadataFilter{
		Fields: map[string]string{"category": "legal"},
	})
	if err != nil {
		t.Fatalf("search with metadata filter: %v", err)
	}
	if len(result.Chunks) != 2 {
		t.Fatalf("expected 2 legal chunks, got %d", len(result.Chunks))
	}
	for _, rc := range result.Chunks {
		if rc.Chunk.Metadata["category"] != "legal" {
			t.Errorf("unexpected category %q for chunk %s", rc.Chunk.Metadata["category"], rc.Chunk.ID)
		}
	}

	// AC-004: несуществующая категория — пустой результат без ошибки.
	empty, err := vs.SearchWithMetadataFilter(ctx, []float64{1, 0}, 10, MetadataFilter{
		Fields: map[string]string{"category": "nonexistent"},
	})
	if err != nil {
		t.Fatalf("expected nil error for no-match filter, got %v", err)
	}
	if len(empty.Chunks) != 0 {
		t.Fatalf("expected 0 chunks for nonexistent filter, got %d", len(empty.Chunks))
	}

	// AC-002: пустой фильтр возвращает все чанки (как обычный Search).
	all, err := vs.SearchWithMetadataFilter(ctx, []float64{1, 0}, 10, MetadataFilter{})
	if err != nil {
		t.Fatalf("empty filter search: %v", err)
	}
	base, err := store.Search(ctx, []float64{1, 0}, 10)
	if err != nil {
		t.Fatalf("base search: %v", err)
	}
	if len(all.Chunks) != len(base.Chunks) {
		t.Fatalf("empty filter: expected %d chunks, got %d", len(base.Chunks), len(all.Chunks))
	}
}
