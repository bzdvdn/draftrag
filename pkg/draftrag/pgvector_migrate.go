package draftrag

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	pgvectorMigrationExtension = "migrations/pgvector/0000_pgvector_extension.sql"
	pgvectorMigrationV1        = "migrations/pgvector/0001_chunks_table.sql"
	pgvectorMigrationV2        = "migrations/pgvector/0002_metadata_and_indexes.sql"
)

// PGVectorMigrateOptions задаёт параметры миграций схемы pgvector-хранилища.
type PGVectorMigrateOptions struct {
	PGVectorOptions

	// DDLTimeout — дефолтный таймаут для миграций, если у ctx нет deadline.
	// Если 0 — используется 30s.
	DDLTimeout time.Duration
}

// MigratePGVector применяет версионированные миграции схемы pgvector-хранилища.
//
// Миграции идемпотентны и безопасны при повторном запуске.
//
// Источник истины DDL — SQL-миграции, встроенные в бинарь через `go:embed`
// (см. `pkg/draftrag/migrations/pgvector/` и `pkg/draftrag/pgvector_migrations.md`).
func MigratePGVector(ctx context.Context, db *sql.DB, opts PGVectorMigrateOptions) error {
	if ctx == nil {
		panic("nil context")
	}
	if db == nil {
		panic("nil db")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	normalized, err := normalizePGVectorOptions(opts.PGVectorOptions)
	if err != nil {
		return err
	}

	ddlTimeout := opts.DDLTimeout
	if ddlTimeout == 0 {
		ddlTimeout = 30 * time.Second
	}

	ctxDDL, cancel := withDefaultTimeout(ctx, ddlTimeout)
	defer cancel()

	migrationsTable := normalized.TableName + "_schema_migrations"
	if err := validateSQLIdentifier(migrationsTable); err != nil {
		return err
	}

	if err := ensureMigrationsTable(ctxDDL, db, migrationsTable); err != nil {
		return err
	}

	current, err := currentSchemaVersion(ctxDDL, db, migrationsTable)
	if err != nil {
		return err
	}

	const latest = 2
	for v := current + 1; v <= latest; v++ {
		if err := applyMigration(ctxDDL, db, v, normalized); err != nil {
			return fmt.Errorf("apply migration v%d: %w", v, err)
		}
		if err := recordSchemaVersion(ctxDDL, db, migrationsTable, v); err != nil {
			return err
		}
	}

	// Конфигурационные аспекты (например, смена IndexMethod) могут требовать пересоздания индекса
	// даже при уже актуальной версии схемы.
	return ensureEmbeddingIndex(ctxDDL, db, normalized)
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB, table string) error {
	ddl := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		quoteIdent(table),
	)
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("create migrations table %s: %w", table, err)
	}
	return nil
}

func currentSchemaVersion(ctx context.Context, db *sql.DB, table string) (int, error) {
	query := fmt.Sprintf(`SELECT COALESCE(MAX(version), 0) FROM %s`, quoteIdent(table))
	var v int
	if err := db.QueryRowContext(ctx, query).Scan(&v); err != nil {
		return 0, fmt.Errorf("read migrations table %s: %w", table, err)
	}
	return v, nil
}

func recordSchemaVersion(ctx context.Context, db *sql.DB, table string, version int) error {
	query := fmt.Sprintf(`INSERT INTO %s (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, quoteIdent(table))
	if _, err := db.ExecContext(ctx, query, version); err != nil {
		return fmt.Errorf("record schema version v%d: %w", version, err)
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, version int, opts PGVectorOptions) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	switch version {
	case 1:
		if opts.CreateExtension {
			if err := execPGVectorMigrationTemplate(ctx, tx, pgvectorMigrationExtension, nil); err != nil {
				return fmt.Errorf("create pgvector extension: %w", err)
			}
		}

		if err := ensureEmbeddingDimCompatible(ctx, tx, opts.TableName, opts.EmbeddingDimension); err != nil {
			return err
		}

		repl := map[string]string{
			"TABLE": quoteIdent(opts.TableName),
			"DIM":   strconv.Itoa(opts.EmbeddingDimension),
		}
		if err := execPGVectorMigrationTemplate(ctx, tx, pgvectorMigrationV1, repl); err != nil {
			return fmt.Errorf("create table %s: %w", opts.TableName, err)
		}

		// Индекс создаём/проверяем вне tx в общем пост-этапе, чтобы поддержать пересоздание по конфигурации.
	case 2:
		repl := map[string]string{
			"TABLE":            quoteIdent(opts.TableName),
			"PARENT_ID_INDEX":  quoteIdent(opts.TableName + "_parent_id_idx"),
			"PARENT_POS_INDEX": quoteIdent(opts.TableName + "_parent_pos_idx"),
		}
		if err := execPGVectorMigrationTemplate(ctx, tx, pgvectorMigrationV2, repl); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown migration version %d", version)
	}

	return tx.Commit()
}

type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func execPGVectorMigrationTemplate(ctx context.Context, q sqlExecQuerier, path string, repl map[string]string) error {
	tpl, err := readPGVectorMigrationAsset(path)
	if err != nil {
		return err
	}
	sqlText, err := renderPGVectorSQLTemplate(tpl, repl)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, sqlText); err != nil {
		return err
	}
	return nil
}

func ensureEmbeddingIndex(ctx context.Context, db *sql.DB, opts PGVectorOptions) error {
	indexName := opts.TableName + "_embedding_idx"
	if err := validateSQLIdentifier(indexName); err != nil {
		return err
	}

	var indexDef sql.NullString
	if err := db.QueryRowContext(
		ctx,
		`SELECT indexdef
		   FROM pg_indexes
		  WHERE tablename = $1
		    AND indexname = $2
		    AND schemaname = ANY (current_schemas(true))`,
		opts.TableName,
		indexName,
	).Scan(&indexDef); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read index definition %s: %w", indexName, err)
	}

	wantMethod := strings.ToLower(opts.IndexMethod)
	wantLists := opts.Lists

	needsRecreate := false
	if !indexDef.Valid || indexDef.String == "" {
		needsRecreate = true
	} else {
		def := strings.ToLower(indexDef.String)
		switch wantMethod {
		case "ivfflat":
			if !strings.Contains(def, "using ivfflat") {
				needsRecreate = true
				break
			}
			// Параметр lists обычно отражается в indexdef.
			if wantLists > 0 {
				if !(strings.Contains(def, fmt.Sprintf("lists = %d", wantLists)) ||
					strings.Contains(def, fmt.Sprintf("lists=%d", wantLists))) {
					needsRecreate = true
				}
			}
		case "hnsw":
			if !strings.Contains(def, "using hnsw") {
				needsRecreate = true
			}
		default:
			return fmt.Errorf("unsupported IndexMethod %q", opts.IndexMethod)
		}
	}

	if !needsRecreate {
		return nil
	}

	// Стратегия смены IndexMethod: drop+create (без CONCURRENTLY).
	drop := fmt.Sprintf(`DROP INDEX IF EXISTS %s`, quoteIdent(indexName))
	if _, err := db.ExecContext(ctx, drop); err != nil {
		return fmt.Errorf("drop index %s: %w", indexName, err)
	}

	create, err := buildCreateIndexDDL(indexName, opts)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, create); err != nil {
		return fmt.Errorf("create index %s: %w", indexName, err)
	}
	return nil
}

func ensureEmbeddingDimCompatible(ctx context.Context, q sqlQuerier, tableName string, wantDim int) error {
	var reg sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT to_regclass($1)`, tableName).Scan(&reg); err != nil {
		return fmt.Errorf("check table existence %s: %w", tableName, err)
	}
	if !reg.Valid || reg.String == "" {
		return nil
	}

	var gotDim int
	if err := q.QueryRowContext(
		ctx,
		`SELECT (atttypmod - 4) AS dim
		   FROM pg_attribute
		  WHERE attrelid = ($1)::regclass
		    AND attname = 'embedding'
		    AND NOT attisdropped`,
		tableName,
	).Scan(&gotDim); err != nil {
		return fmt.Errorf("read embedding dimension for %s: %w", tableName, err)
	}
	if gotDim != wantDim {
		return fmt.Errorf("embedding dimension mismatch for table %s: got=%d want=%d", tableName, gotDim, wantDim)
	}
	return nil
}

type sqlQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func withDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	return ctx2, cancel
}
