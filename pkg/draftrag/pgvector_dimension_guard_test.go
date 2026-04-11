package draftrag

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"
)

var errDBUsed = errors.New("db used")

func init() {
	sql.Register("pgvector_dimension_guard_test", denyDBDriver{})
}

type denyDBDriver struct{}

func (denyDBDriver) Open(name string) (driver.Conn, error) {
	return denyDBConn{}, nil
}

type denyDBConn struct{}

func (denyDBConn) Prepare(query string) (driver.Stmt, error) { return denyStmt{}, nil }
func (denyDBConn) Close() error                              { return nil }
func (denyDBConn) Begin() (driver.Tx, error)                 { return denyTx{}, nil }

func (denyDBConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return nil, errDBUsed
}

func (denyDBConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return nil, errDBUsed
}

type denyStmt struct{}

func (denyStmt) Close() error                                    { return nil }
func (denyStmt) NumInput() int                                   { return -1 }
func (denyStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, errDBUsed }
func (denyStmt) Query(args []driver.Value) (driver.Rows, error)  { return nil, errDBUsed }

type denyTx struct{}

func (denyTx) Commit() error   { return errDBUsed }
func (denyTx) Rollback() error { return nil }

func TestPGVector_DimensionMismatch_ErrorIs(t *testing.T) {
	db, err := sql.Open("pgvector_dimension_guard_test", "")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store := NewPGVectorStore(db, PGVectorOptions{
		TableName:          "draftrag_chunks",
		EmbeddingDimension: 3,
		CreateExtension:    false,
		IndexMethod:        "ivfflat",
		Lists:              100,
	})

	ctx := context.Background()

	t.Run("Upsert mismatch", func(t *testing.T) {
		err := store.Upsert(ctx, Chunk{
			ID:        "c1",
			ParentID:  "p1",
			Content:   "x",
			Position:  0,
			Embedding: []float64{1, 2},
		})
		if !errors.Is(err, ErrEmbeddingDimensionMismatch) {
			t.Fatalf("expected ErrEmbeddingDimensionMismatch, got %v", err)
		}
	})

	t.Run("Search mismatch", func(t *testing.T) {
		_, err := store.Search(ctx, []float64{1, 2}, 1)
		if !errors.Is(err, ErrEmbeddingDimensionMismatch) {
			t.Fatalf("expected ErrEmbeddingDimensionMismatch, got %v", err)
		}
	})

	t.Run("Nil embedding is not mismatch", func(t *testing.T) {
		_, err := store.Search(ctx, nil, 1)
		if err == nil {
			t.Fatalf("expected error")
		}
		if errors.Is(err, ErrEmbeddingDimensionMismatch) {
			t.Fatalf("did not expect ErrEmbeddingDimensionMismatch, got %v", err)
		}
	})

	t.Run("Happy path does not return mismatch", func(t *testing.T) {
		err := store.Upsert(ctx, Chunk{
			ID:        "c2",
			ParentID:  "p2",
			Content:   "x",
			Position:  0,
			Embedding: []float64{1, 2, 3},
		})
		if err == nil {
			t.Fatalf("expected error from deny driver")
		}
		if errors.Is(err, ErrEmbeddingDimensionMismatch) {
			t.Fatalf("did not expect ErrEmbeddingDimensionMismatch, got %v", err)
		}
		if !errors.Is(err, errDBUsed) && !errors.Is(err, io.EOF) {
			// На некоторых платформах sql может вернуть другую ошибку при попытке использовать driver.
			// В любом случае важно, что это не классифицируется как dimension mismatch.
			t.Fatalf("expected db usage error, got %v", err)
		}
	})
}
