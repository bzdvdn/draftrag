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

type pgvectorTestEmbedder struct{}

func (pgvectorTestEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	text = strings.ToLower(text)
	if strings.Contains(text, "cat") {
		return []float64{1, 0, 0}, nil
	}
	return []float64{0, 1, 0}, nil
}

type pgvectorTestLLM struct{}

func (pgvectorTestLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "ok", nil
}

func TestPipeline_WithPGVectorStore(t *testing.T) {
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

	tableName := "draftrag_pipeline_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
	opts := PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: 3,
		CreateExtension:    true,
		IndexMethod:        "ivfflat",
		Lists:              10,
	}

	if err := SetupPGVector(ctx, db, opts); err != nil {
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "must be superuser") ||
			strings.Contains(err.Error(), "type \"vector\" does not exist") {
			t.Skipf("pgvector is not available or insufficient privileges: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`"`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`_schema_migrations"`)
	})

	store := NewPGVectorStore(db, opts)
	pipeline := NewPipeline(store, pgvectorTestLLM{}, pgvectorTestEmbedder{})

	if err := pipeline.Index(ctx, []Document{{ID: "doc-1", Content: "cat"}}); err != nil {
		t.Fatalf("index: %v", err)
	}
	if err := pipeline.Index(ctx, []Document{{ID: "doc-2", Content: "dog"}}); err != nil {
		t.Fatalf("index: %v", err)
	}

	result, err := pipeline.Search("cat").TopK(5).Retrieve(ctx)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(result.Chunks) == 0 {
		t.Fatalf("expected non-empty results")
	}

	filtered, err := pipeline.Search("dog").TopK(5).ParentIDs("doc-2").Retrieve(ctx)
	if err != nil {
		t.Fatalf("query with parent filter: %v", err)
	}
	for _, rc := range filtered.Chunks {
		if rc.Chunk.ParentID != "doc-2" {
			t.Fatalf("expected ParentID=doc-2, got %q", rc.Chunk.ParentID)
		}
	}
}
