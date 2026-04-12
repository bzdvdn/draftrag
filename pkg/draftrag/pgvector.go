package draftrag

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/infrastructure/vectorstore"
)

// PGVectorOptions задаёт параметры подключения pgvector-backed VectorStore.
type PGVectorOptions struct {
	// TableName — имя таблицы для хранения чанков.
	// В v1 поддерживается только простой идентификатор без схемы (например, "draftrag_chunks").
	TableName string

	// EmbeddingDimension — фиксированная размерность embedding-векторов.
	// При несоответствии размерности операции store возвращают ErrEmbeddingDimensionMismatch (errors.Is).
	EmbeddingDimension int

	// CreateExtension включает попытку выполнить `CREATE EXTENSION IF NOT EXISTS vector`.
	// Часто требует повышенных прав; при отсутствии прав будет возвращена ошибка.
	CreateExtension bool

	// IndexMethod — метод индекса: "ivfflat" (по умолчанию) или "hnsw".
	IndexMethod string

	// Lists — параметр ivfflat индекса (WITH (lists = N)).
	Lists int
}

// PGVectorStoreOptions — единый контейнер опций для NewPGVectorStore*.
//
// Объединяет параметры подключения/схемы (PGVectorOptions) и runtime ограничения (PGVectorRuntimeOptions),
// чтобы публичный API следовал единому options pattern.
type PGVectorStoreOptions struct {
	PGVectorOptions
	Runtime PGVectorRuntimeOptions
}

// PGVectorRuntimeOptions задаёт лимиты и таймауты выполнения операций VectorStore.
type PGVectorRuntimeOptions struct {
	// SearchTimeout — дефолтный таймаут для Search*, если у ctx нет deadline.
	SearchTimeout time.Duration
	// UpsertTimeout — дефолтный таймаут для Upsert, если у ctx нет deadline.
	UpsertTimeout time.Duration
	// DeleteTimeout — дефолтный таймаут для Delete, если у ctx нет deadline.
	DeleteTimeout time.Duration

	// MaxTopK ограничивает topK в Search*. 0 означает “без лимита”.
	MaxTopK int
	// MaxParentIDs ограничивает количество ParentIDs в фильтре. 0 означает “без лимита”.
	MaxParentIDs int
	// MaxContentBytes ограничивает размер chunk.Content в байтах. 0 означает “без лимита”.
	MaxContentBytes int
}

// NewPGVectorStoreWithOptions создаёт pgvector-backed реализацию VectorStore (канонический options pattern).
//
// Схема БД не создаётся автоматически: перед использованием примените миграции через MigratePGVector
// (или SetupPGVector как backward-compatible alias).
//
// Если у ctx нет deadline, операции store используют дефолтные таймауты (см. PGVectorRuntimeOptions).
func NewPGVectorStoreWithOptions(db *sql.DB, opts PGVectorStoreOptions) VectorStore {
	if db == nil {
		panic("nil db")
	}
	normalized, err := normalizePGVectorOptions(opts.PGVectorOptions)
	if err != nil {
		panic(err.Error())
	}

	runtime := normalizePGVectorRuntimeOptions(opts.Runtime)
	if runtime.MaxTopK < 0 || runtime.MaxParentIDs < 0 || runtime.MaxContentBytes < 0 {
		panic("invalid PGVectorRuntimeOptions")
	}
	internalRuntime := vectorstore.RuntimeOptions{
		SearchTimeout:   runtime.SearchTimeout,
		UpsertTimeout:   runtime.UpsertTimeout,
		DeleteTimeout:   runtime.DeleteTimeout,
		MaxTopK:         runtime.MaxTopK,
		MaxParentIDs:    runtime.MaxParentIDs,
		MaxContentBytes: runtime.MaxContentBytes,
	}
	return vectorstore.NewPGVectorStoreWithRuntimeOptions(
		db,
		normalized.TableName,
		normalized.EmbeddingDimension,
		internalRuntime,
	)
}

// NewPGVectorStore создаёт pgvector-backed реализацию VectorStore.
//
// Схема БД не создаётся автоматически: перед использованием примените миграции через MigratePGVector
// (или SetupPGVector как backward-compatible alias).
//
// Если у ctx нет deadline, операции store используют дефолтные таймауты (см. PGVectorRuntimeOptions).
func NewPGVectorStore(db *sql.DB, opts PGVectorOptions) VectorStore {
	return NewPGVectorStoreWithOptions(db, PGVectorStoreOptions{PGVectorOptions: opts})
}

// NewPGVectorStoreWithRuntimeOptions создаёт pgvector-backed реализацию VectorStore с runtime ограничениями.
//
// Deprecated: используйте NewPGVectorStoreWithOptions (PGVectorStoreOptions.Runtime).
func NewPGVectorStoreWithRuntimeOptions(db *sql.DB, opts PGVectorOptions, runtime PGVectorRuntimeOptions) VectorStore {
	return NewPGVectorStoreWithOptions(db, PGVectorStoreOptions{
		PGVectorOptions: opts,
		Runtime:         runtime,
	})
}

// SetupPGVector — backward-compatible alias для MigratePGVector.
//
// Примечание (production): рекомендуется запускать миграции отдельным шагом деплоя (deploy job / init container),
// т.к. DDL может требовать повышенных прав и занимать заметное время.
//
// Рекомендуемый подход:
//   - для production: применять SQL-миграции из `pkg/draftrag/migrations/pgvector/` отдельным шагом деплоя
//     (см. `pkg/draftrag/pgvector_migrations.md`);
//   - при необходимости — вызывать SetupPGVector/MigratePGVector явно в deploy job, но не “на старте сервиса”.
//
// Смена IndexMethod или параметров индекса приводит к стратегии drop+create для embedding-индекса (без CONCURRENTLY).
func SetupPGVector(ctx context.Context, db *sql.DB, opts PGVectorOptions) error {
	// Backward compatible alias: SetupPGVector поднимает/обновляет схему до актуальной версии.
	return MigratePGVector(ctx, db, PGVectorMigrateOptions{PGVectorOptions: opts})
}

func normalizePGVectorOptions(opts PGVectorOptions) (PGVectorOptions, error) {
	if opts.TableName == "" {
		opts.TableName = "draftrag_chunks"
	}
	if err := validateSQLIdentifier(opts.TableName); err != nil {
		return PGVectorOptions{}, err
	}
	if opts.EmbeddingDimension <= 0 {
		return PGVectorOptions{}, errors.New("EmbeddingDimension must be > 0")
	}
	if opts.IndexMethod == "" {
		opts.IndexMethod = "ivfflat"
	}
	opts.IndexMethod = strings.ToLower(opts.IndexMethod)
	switch opts.IndexMethod {
	case "ivfflat", "hnsw":
	default:
		return PGVectorOptions{}, fmt.Errorf("unsupported IndexMethod %q", opts.IndexMethod)
	}
	if opts.Lists == 0 {
		opts.Lists = 100
	}
	if opts.Lists < 0 {
		return PGVectorOptions{}, errors.New("Lists must be >= 0")
	}

	return opts, nil
}

func buildCreateIndexDDL(indexName string, opts PGVectorOptions) (string, error) {
	quotedIndex := quoteIdent(indexName)
	quotedTable := quoteIdent(opts.TableName)

	switch opts.IndexMethod {
	case "ivfflat":
		if opts.Lists <= 0 {
			return "", errors.New("Lists must be > 0 for ivfflat")
		}
		return fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON %s USING ivfflat (embedding vector_cosine_ops) WITH (lists = %d)`,
			quotedIndex,
			quotedTable,
			opts.Lists,
		), nil
	case "hnsw":
		return fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON %s USING hnsw (embedding vector_cosine_ops)`,
			quotedIndex,
			quotedTable,
		), nil
	default:
		return "", fmt.Errorf("unsupported IndexMethod %q", opts.IndexMethod)
	}
}

func normalizePGVectorRuntimeOptions(opts PGVectorRuntimeOptions) PGVectorRuntimeOptions {
	if opts.SearchTimeout == 0 {
		opts.SearchTimeout = 2 * time.Second
	}
	if opts.UpsertTimeout == 0 {
		opts.UpsertTimeout = 5 * time.Second
	}
	if opts.DeleteTimeout == 0 {
		opts.DeleteTimeout = 5 * time.Second
	}
	if opts.MaxTopK == 0 {
		opts.MaxTopK = 50
	}
	if opts.MaxParentIDs == 0 {
		opts.MaxParentIDs = 128
	}
	// MaxContentBytes по умолчанию не ограничиваем.
	return opts
}

func validateSQLIdentifier(name string) error {
	if name == "" {
		return errors.New("identifier is empty")
	}
	for _, r := range name {
		if !(r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return fmt.Errorf("invalid identifier %q: only [A-Za-z0-9_] allowed", name)
		}
	}
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("invalid identifier %q: must not start with a digit", name)
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
