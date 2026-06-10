package draftrag

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type mockQuerier struct {
	execFn func(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (m *mockQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return m.execFn(ctx, query, args...)
}

func (m *mockQuerier) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

// @sk-test hardening-2026q2#T3.3: execPGVectorMigrationTemplate
func TestExecPGVectorMigrationTemplate_ExecutesSQL(t *testing.T) {
	var executed bool
	mq := &mockQuerier{
		execFn: func(_ context.Context, query string, _ ...any) (sql.Result, error) {
			executed = true
			if query == "" {
				t.Error("expected non-empty SQL query")
			}
			return nil, nil
		},
	}

	err := execPGVectorMigrationTemplate(context.Background(), mq, "migrations/pgvector/0001_chunks_table.sql", map[string]string{
		"TABLE": "draftrag_chunks",
		"DIM":   "768",
	})
	if err != nil {
		t.Fatalf("execPGVectorMigrationTemplate failed: %v", err)
	}
	if !executed {
		t.Fatal("expected ExecContext to be called")
	}
}

func TestExecPGVectorMigrationTemplate_InvalidPath(t *testing.T) {
	mq := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
			return nil, nil
		},
	}

	err := execPGVectorMigrationTemplate(context.Background(), mq, "nonexistent/path.sql", nil)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestExecPGVectorMigrationTemplate_PropagatesExecError(t *testing.T) {
	want := errors.New("exec failed")
	mq := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
			return nil, want
		},
	}

	err := execPGVectorMigrationTemplate(context.Background(), mq, "migrations/pgvector/0001_chunks_table.sql", nil)
	if !errors.Is(err, want) {
		t.Fatalf("expected exec error, got %v", err)
	}
}
