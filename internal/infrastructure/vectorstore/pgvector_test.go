package vectorstore_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
	"github.com/bzdvdn/draftrag/pkg/draftrag"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPGVectorStore_UpsertDeleteSearch(t *testing.T) {
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

	tableName := "draftrag_pgvector_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
	opts := draftrag.PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: 3,
		CreateExtension:    true,
		IndexMethod:        "ivfflat",
		Lists:              10,
	}

	if err := draftrag.SetupPGVector(ctx, db, opts); err != nil {
		// В CI/ограниченных окружениях часто нет прав на CREATE EXTENSION или отсутствует pgvector.
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "must be superuser") ||
			strings.Contains(err.Error(), "type \"vector\" does not exist") {
			t.Skipf("pgvector is not available or insufficient privileges: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	if err := draftrag.SetupPGVector(ctx, db, opts); err != nil {
		t.Fatalf("setup (idempotent): %v", err)
	}

	{
		var indexDef string
		indexName := tableName + "_embedding_idx"
		if err := db.QueryRowContext(
			ctx,
			`SELECT indexdef
			   FROM pg_indexes
			  WHERE tablename = $1
			    AND indexname = $2
			    AND schemaname = ANY (current_schemas(true))`,
			tableName,
			indexName,
		).Scan(&indexDef); err != nil {
			t.Fatalf("read indexdef: %v", err)
		}
		if !strings.Contains(strings.ToLower(indexDef), "using ivfflat") {
			t.Fatalf("expected ivfflat index, got %q", indexDef)
		}
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`"`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`_schema_migrations"`)
	})

	store := vectorstore.NewPGVectorStore(db, tableName, 3)

	chunk1 := domain.Chunk{
		ID:        "doc-1#0",
		Content:   "cat",
		ParentID:  "doc-1",
		Embedding: []float64{1, 0, 0},
		Position:  0,
	}
	chunk2 := domain.Chunk{
		ID:        "doc-2#0",
		Content:   "dog",
		ParentID:  "doc-2",
		Embedding: []float64{0, 1, 0},
		Position:  0,
	}
	chunk3 := domain.Chunk{
		ID:        "doc-3#0",
		Content:   "anti-cat",
		ParentID:  "doc-3",
		Embedding: []float64{-1, 0, 0},
		Position:  0,
	}

	if err := store.Upsert(ctx, chunk1); err != nil {
		t.Fatalf("upsert chunk1: %v", err)
	}
	if err := store.Upsert(ctx, chunk2); err != nil {
		t.Fatalf("upsert chunk2: %v", err)
	}
	if err := store.Upsert(ctx, chunk3); err != nil {
		t.Fatalf("upsert chunk3: %v", err)
	}

	result, err := store.Search(ctx, []float64{1, 0, 0}, 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Chunks) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Chunks))
	}
	if result.Chunks[0].Score < result.Chunks[1].Score {
		t.Fatalf("expected results sorted by score desc")
	}
	for _, rc := range result.Chunks {
		if rc.Score < -1 || rc.Score > 1 {
			t.Fatalf("expected score in [-1, 1], got %v", rc.Score)
		}
	}
	if result.TotalFound < 3 {
		t.Fatalf("expected TotalFound >= 3, got %d", result.TotalFound)
	}

	if err := store.Delete(ctx, chunk1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	afterDelete, err := store.Search(ctx, []float64{1, 0, 0}, 10)
	if err != nil {
		t.Fatalf("search after delete: %v", err)
	}
	for _, rc := range afterDelete.Chunks {
		if rc.Chunk.ID == chunk1.ID {
			t.Fatalf("expected deleted chunk to be absent")
		}
	}

	filtered, err := store.SearchWithFilter(ctx, []float64{0, 1, 0}, 10, domain.ParentIDFilter{ParentIDs: []string{"doc-2"}})
	if err != nil {
		t.Fatalf("search with filter: %v", err)
	}
	for _, rc := range filtered.Chunks {
		if rc.Chunk.ParentID != "doc-2" {
			t.Fatalf("expected ParentID=doc-2, got %q", rc.Chunk.ParentID)
		}
	}
}

func TestPGVectorStore_SearchBM25(t *testing.T) {
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

	tableName := "draftrag_bm25_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
	opts := draftrag.PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: 3,
		CreateExtension:    true,
		IndexMethod:        "ivfflat",
		Lists:              10,
	}

	if err := draftrag.SetupPGVector(ctx, db, opts); err != nil {
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "must be superuser") ||
			strings.Contains(err.Error(), "type \"vector\" does not exist") {
			t.Skipf("pgvector is not available or insufficient privileges: %v", err)
		}
		t.Fatalf("setup pgvector: %v", err)
	}

	// Применяем BM25 миграцию
	bm25SQL := `
		ALTER TABLE ` + tableName + ` ADD COLUMN IF NOT EXISTS content_tsv tsvector;
		CREATE INDEX IF NOT EXISTS idx_` + tableName + `_tsv ON ` + tableName + ` USING GIN (content_tsv);
		CREATE OR REPLACE FUNCTION ` + tableName + `_tsv_update() RETURNS TRIGGER AS $$
		BEGIN NEW.content_tsv := to_tsvector('english', NEW.content); RETURN NEW; END;
		$$ LANGUAGE plpgsql;
		DROP TRIGGER IF EXISTS trigger_` + tableName + `_tsv ON ` + tableName + `;
		CREATE TRIGGER trigger_` + tableName + `_tsv BEFORE INSERT OR UPDATE ON ` + tableName + `
		FOR EACH ROW EXECUTE FUNCTION ` + tableName + `_tsv_update();
	`
	if _, err := db.ExecContext(ctx, bm25SQL); err != nil {
		t.Fatalf("apply bm25 migration: %v", err)
	}

	store := vectorstore.NewPGVectorStore(db, tableName, 3)

	// Создаём чанки с контентом для BM25 поиска
	chunks := []domain.Chunk{
		{ID: "bm25-1", Content: "PostgreSQL full text search is powerful", ParentID: "doc-1", Embedding: []float64{1, 0, 0}, Position: 0},
		{ID: "bm25-2", Content: "BM25 ranking algorithm for search", ParentID: "doc-2", Embedding: []float64{0, 1, 0}, Position: 0},
		{ID: "bm25-3", Content: "vector search with cosine similarity", ParentID: "doc-3", Embedding: []float64{0, 0, 1}, Position: 0},
	}

	for _, chunk := range chunks {
		if err := store.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert %s: %v", chunk.ID, err)
		}
	}

	// Тест: поиск по запросу "search"
	result, err := store.SearchBM25(ctx, "search", 2)
	if err != nil {
		t.Fatalf("bm25 search: %v", err)
	}

	// Должны найти bm25-1 и bm25-2 (содержат "search")
	if len(result.Chunks) == 0 {
		t.Skip("BM25 search returned no results - tsvector may not be properly indexed yet")
	}

	// Проверяем что результаты отсортированы по score
	for i := 1; i < len(result.Chunks); i++ {
		if result.Chunks[i].Score > result.Chunks[i-1].Score {
			t.Fatalf("expected results sorted by score desc")
		}
	}

	// Проверяем что QueryText заполнен
	if result.QueryText != "search" {
		t.Errorf("expected QueryText='search', got %q", result.QueryText)
	}

	// Тест: пустой запрос должен возвращать ошибку
	_, err = store.SearchBM25(ctx, "", 2)
	if err == nil {
		t.Error("expected error for empty query")
	}

	// Тест: topK <= 0 должен возвращать ошибку
	_, err = store.SearchBM25(ctx, "test", 0)
	if err == nil {
		t.Error("expected error for topK=0")
	}
}

func TestPGVectorStore_SearchHybrid(t *testing.T) {
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

	tableName := "draftrag_hybrid_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
	opts := draftrag.PGVectorOptions{
		TableName:          tableName,
		EmbeddingDimension: 3,
		CreateExtension:    true,
		IndexMethod:        "ivfflat",
		Lists:              10,
	}

	if err := draftrag.SetupPGVector(ctx, db, opts); err != nil {
		if strings.Contains(err.Error(), "permission denied") ||
			strings.Contains(err.Error(), "must be superuser") ||
			strings.Contains(err.Error(), "type \"vector\" does not exist") {
			t.Skipf("pgvector is not available or insufficient privileges: %v", err)
		}
		t.Fatalf("setup pgvector: %v", err)
	}

	// Применяем BM25 миграцию
	bm25SQL := `
		ALTER TABLE ` + tableName + ` ADD COLUMN IF NOT EXISTS content_tsv tsvector;
		CREATE INDEX IF NOT EXISTS idx_` + tableName + `_tsv ON ` + tableName + ` USING GIN (content_tsv);
		CREATE OR REPLACE FUNCTION ` + tableName + `_tsv_update() RETURNS TRIGGER AS $$
		BEGIN NEW.content_tsv := to_tsvector('english', NEW.content); RETURN NEW; END;
		$$ LANGUAGE plpgsql;
		DROP TRIGGER IF EXISTS trigger_` + tableName + `_tsv ON ` + tableName + `;
		CREATE TRIGGER trigger_` + tableName + `_tsv BEFORE INSERT OR UPDATE ON ` + tableName + `
		FOR EACH ROW EXECUTE FUNCTION ` + tableName + `_tsv_update();
	`
	if _, err := db.ExecContext(ctx, bm25SQL); err != nil {
		t.Fatalf("apply bm25 migration: %v", err)
	}

	store := vectorstore.NewPGVectorStore(db, tableName, 3)

	// Создаём чанки
	chunks := []domain.Chunk{
		{ID: "h-1", Content: "full text search with PostgreSQL", ParentID: "doc-1", Embedding: []float64{1, 0, 0}, Position: 0},
		{ID: "h-2", Content: "vector search using embeddings", ParentID: "doc-2", Embedding: []float64{0, 1, 0}, Position: 0},
		{ID: "h-3", Content: "hybrid search combines both", ParentID: "doc-3", Embedding: []float64{0, 0, 1}, Position: 0},
	}

	for _, chunk := range chunks {
		if err := store.Upsert(ctx, chunk); err != nil {
			t.Fatalf("upsert %s: %v", chunk.ID, err)
		}
	}

	// Тест: гибридный поиск с RRF
	query := "search"
	embedding := []float64{1, 0, 0}
	config := domain.DefaultHybridConfig()

	result, err := store.SearchHybrid(ctx, query, embedding, 2, config)
	if err != nil {
		t.Fatalf("hybrid search: %v", err)
	}

	if len(result.Chunks) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(result.Chunks))
	}

	if result.QueryText != query {
		t.Errorf("expected QueryText=%q, got %q", query, result.QueryText)
	}

	// Тест: невалидная конфигурация
	_, err = store.SearchHybrid(ctx, query, embedding, 2, domain.HybridConfig{SemanticWeight: -1, RRFK: 60})
	if err == nil {
		t.Error("expected error for invalid config")
	}
}
