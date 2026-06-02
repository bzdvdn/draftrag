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

// @sk-test api-consistency-pass#T3.2: integration test для PGVector.BeginTx
// (DEC-005, AC-008). Проверяет, что rollback восстанавливает исходные чанки,
// смоделировав сценарий "failing embedder на 3-м чанке → store unchanged"
// на уровне TransactionalDocumentStore (а не через полный Pipeline).
//
// Pipeline-уровень (UpdateDocument с chunker + failing embedder) уже
// покрыт unit-тестом TestPipeline_UpdateDocument_BestEffort_ReturnsErrUpdateNotAtomic
// в internal/application/pipeline_test.go на in-memory store; тут мы
// проверяем именно pgvector-реализацию TransactionalDocumentStore.
//
// Гейтинг: требует RUN_INTEGRATION_TESTS=1 И PGVECTOR_TEST_DSN. Без обоих
// тест скипается (по аналогии с остальными integration-тестами).
func TestPGVector_BeginTx_RollbackPreservesOldChunks(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("RUN_INTEGRATION_TESTS != 1; set RUN_INTEGRATION_TESTS=1 to run")
	}
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
		t.Skipf("ping: %v", err)
	}

	tableName := "draftrag_pgvector_atomic_test_" +
		strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
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
			t.Skipf("pgvector not available: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`"`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`_schema_migrations"`)
	})

	store := vectorstore.NewPGVectorStore(db, tableName, 3)

	// 1. Index 3 начальных чанка для doc-1.
	initialChunks := []domain.Chunk{
		{ID: "doc-1#0", Content: "hello", ParentID: "doc-1", Position: 0, Embedding: []float64{1, 0, 0}},
		{ID: "doc-1#1", Content: "world", ParentID: "doc-1", Position: 1, Embedding: []float64{0, 1, 0}},
		{ID: "doc-1#2", Content: "test", ParentID: "doc-1", Position: 2, Embedding: []float64{0, 0, 1}},
	}
	for _, ch := range initialChunks {
		if err := store.Upsert(ctx, ch); err != nil {
			t.Fatalf("upsert initial: %v", err)
		}
	}

	// Sanity: 3 чанка в store.
	pre, err := store.SearchWithFilter(ctx, []float64{1, 0, 0}, 10,
		domain.ParentIDFilter{ParentIDs: []string{"doc-1"}})
	if err != nil {
		t.Fatalf("search pre: %v", err)
	}
	if len(pre.Chunks) != 3 {
		t.Fatalf("expected 3 initial chunks, got %d", len(pre.Chunks))
	}

	// 2. Симулируем pipeline'овский tx-flow: BeginTx → DeleteByParentID →
	//    Upsert нового чанка (первый из 3) → "failing embedder на 3-м чанке"
	//    → Rollback. Эквивалентно updateDocumentAtomicTransactional с
	//    3-чанковым pipeline'ом, где embed падает на 3-м.
	txStore := domain.TransactionalDocumentStore(store)
	tx, err := txStore.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	if err := tx.DeleteByParentID(ctx, "doc-1"); err != nil {
		t.Fatalf("delete parent in tx: %v", err)
	}

	// 1 из 3 upsert'ов успел (имитация: до embed-failure на 3-м чанке
	// уже произошли 2 успешных embed'а, 1-й chunk upsert'ится в tx).
	firstNew := domain.Chunk{
		ID:        "doc-1#0",
		Content:   "new-content-1",
		ParentID:  "doc-1",
		Position:  0,
		Embedding: []float64{0.5, 0.5, 0},
	}
	if err := tx.Upsert(ctx, firstNew); err != nil {
		t.Fatalf("upsert first new in tx: %v", err)
	}

	// 2-й upsert — имитация сбоя. Коммитим Rollback (не Commit).
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// 3. Verify: store должен содержать исходные 3 чанка (не 1 новый).
	post, err := store.SearchWithFilter(ctx, []float64{1, 0, 0}, 10,
		domain.ParentIDFilter{ParentIDs: []string{"doc-1"}})
	if err != nil {
		t.Fatalf("search post: %v", err)
	}
	if len(post.Chunks) != 3 {
		t.Fatalf("expected 3 chunks after rollback (old preserved), got %d", len(post.Chunks))
	}
	// Содержимое должно быть исходным.
	gotContents := make(map[string]bool, 3)
	for _, rc := range post.Chunks {
		gotContents[rc.Chunk.Content] = true
	}
	for _, want := range []string{"hello", "world", "test"} {
		if !gotContents[want] {
			t.Errorf("missing original content %q after rollback; got contents=%v", want, gotContents)
		}
	}
	if gotContents["new-content-1"] {
		t.Error("new content found after rollback — tx was not properly rolled back")
	}
}

// @sk-test api-consistency-pass#T3.2: Commit фиксирует изменения; после
// Commit второй BeginTx видит новые чанки.
func TestPGVector_BeginTx_CommitPersistsNewChunks(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("RUN_INTEGRATION_TESTS != 1; set RUN_INTEGRATION_TESTS=1 to run")
	}
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
		t.Skipf("ping: %v", err)
	}

	tableName := "draftrag_pgvector_atomic_commit_test_" +
		strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "_")
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
			t.Skipf("pgvector not available: %v", err)
		}
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`"`)
		_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS "`+tableName+`_schema_migrations"`)
	})

	store := vectorstore.NewPGVectorStore(db, tableName, 3)

	txStore := domain.TransactionalDocumentStore(store)
	tx, err := txStore.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	newChunks := []domain.Chunk{
		{ID: "doc-2#0", Content: "alpha", ParentID: "doc-2", Position: 0, Embedding: []float64{0.7, 0.7, 0}},
		{ID: "doc-2#1", Content: "beta", ParentID: "doc-2", Position: 1, Embedding: []float64{0.6, 0.6, 0.1}},
	}
	for _, ch := range newChunks {
		if err := tx.Upsert(ctx, ch); err != nil {
			t.Fatalf("upsert in tx: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	result, err := store.SearchWithFilter(ctx, []float64{0.7, 0.7, 0}, 10,
		domain.ParentIDFilter{ParentIDs: []string{"doc-2"}})
	if err != nil {
		t.Fatalf("search post-commit: %v", err)
	}
	if len(result.Chunks) != 2 {
		t.Fatalf("expected 2 chunks after commit, got %d", len(result.Chunks))
	}
}
