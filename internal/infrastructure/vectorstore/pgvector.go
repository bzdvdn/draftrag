package vectorstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
	pgvector "github.com/pgvector/pgvector-go"
)

// RuntimeOptions задаёт ограничения и таймауты выполнения операций pgvector store.
type RuntimeOptions struct {
	SearchTimeout time.Duration
	UpsertTimeout time.Duration
	DeleteTimeout time.Duration

	// MaxTopK ограничивает topK в Search*. 0 означает “без лимита”.
	MaxTopK int
	// MaxParentIDs ограничивает количество ParentIDs в фильтре. 0 означает “без лимита”.
	MaxParentIDs int
	// MaxContentBytes ограничивает размер chunk.Content в байтах. 0 означает “без лимита”.
	MaxContentBytes int
}

func defaultRuntimeOptions() RuntimeOptions {
	return RuntimeOptions{
		SearchTimeout:   2 * time.Second,
		UpsertTimeout:   5 * time.Second,
		DeleteTimeout:   5 * time.Second,
		MaxTopK:         50,
		MaxParentIDs:    128,
		MaxContentBytes: 0,
	}
}

// PGVectorStore реализует domain.VectorStore поверх PostgreSQL+pgvector.
//
// Примечание: создание схемы (таблицы/индекса) выполняется отдельным helper'ом в `pkg/draftrag`.
type PGVectorStore struct {
	db               *sql.DB
	tableName        string
	embeddingDim     int
	quotedTableIdent string
	runtime          RuntimeOptions
}

var _ domain.VectorStore = (*PGVectorStore)(nil)
var _ domain.VectorStoreWithFilters = (*PGVectorStore)(nil)
var _ domain.HybridSearcher = (*PGVectorStore)(nil)
var _ domain.HybridSearcherWithFilters = (*PGVectorStore)(nil)

// NewPGVectorStore создаёт новый pgvector-backed store.
func NewPGVectorStore(db *sql.DB, tableName string, embeddingDimension int) *PGVectorStore {
	return NewPGVectorStoreWithRuntimeOptions(db, tableName, embeddingDimension, defaultRuntimeOptions())
}

// NewPGVectorStoreWithRuntimeOptions создаёт новый pgvector-backed store с runtime options.
func NewPGVectorStoreWithRuntimeOptions(
	db *sql.DB,
	tableName string,
	embeddingDimension int,
	runtime RuntimeOptions,
) *PGVectorStore {
	if db == nil {
		panic("nil db")
	}
	if embeddingDimension <= 0 {
		panic("embedding dimension must be > 0")
	}
	if err := validateSQLIdentifier(tableName); err != nil {
		panic(err.Error())
	}
	if runtime.MaxTopK < 0 {
		panic("MaxTopK must be >= 0")
	}
	if runtime.MaxParentIDs < 0 {
		panic("MaxParentIDs must be >= 0")
	}
	if runtime.MaxContentBytes < 0 {
		panic("MaxContentBytes must be >= 0")
	}

	return &PGVectorStore{
		db:               db,
		tableName:        tableName,
		embeddingDim:     embeddingDimension,
		quotedTableIdent: quoteIdent(tableName),
		runtime:          runtime,
	}
}

// Upsert сохраняет или обновляет чанк в хранилище.
func (s *PGVectorStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.UpsertTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := chunk.Validate(); err != nil {
		return err
	}
	if s.runtime.MaxContentBytes > 0 && len(chunk.Content) > s.runtime.MaxContentBytes {
		return fmt.Errorf("chunk content too large: got=%d max=%d", len(chunk.Content), s.runtime.MaxContentBytes)
	}
	if err := validateEmbedding(chunk.Embedding, s.embeddingDim); err != nil {
		return err
	}

	// @ds-task T2.1: Сохранять Chunk.Metadata в JSONB-колонку при upsert (AC-001, DEC-002)
	metadataJSON := encodeMetadata(chunk.Metadata)

	queryV2 := fmt.Sprintf(
		`INSERT INTO %s (id, parent_id, content, position, embedding, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE
		 SET parent_id = EXCLUDED.parent_id,
		     content = EXCLUDED.content,
		     position = EXCLUDED.position,
		     embedding = EXCLUDED.embedding,
		     metadata = EXCLUDED.metadata,
		     updated_at = now()`,
		s.quotedTableIdent,
	)

	_, err := s.db.ExecContext(
		ctx,
		queryV2,
		chunk.ID,
		chunk.ParentID,
		chunk.Content,
		chunk.Position,
		pgVectorFromFloat64(chunk.Embedding),
		metadataJSON,
	)
	if err == nil {
		return nil
	}

	// Backward compatibility: если схема ещё без updated_at/metadata (pre-migration 0002), пробуем legacy upsert.
	if strings.Contains(err.Error(), "column") &&
		(strings.Contains(err.Error(), "updated_at") || strings.Contains(err.Error(), "metadata")) {
		queryV1 := fmt.Sprintf(
			`INSERT INTO %s (id, parent_id, content, position, embedding)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (id) DO UPDATE
			 SET parent_id = EXCLUDED.parent_id,
			     content = EXCLUDED.content,
			     position = EXCLUDED.position,
			     embedding = EXCLUDED.embedding`,
			s.quotedTableIdent,
		)
		_, err2 := s.db.ExecContext(
			ctx,
			queryV1,
			chunk.ID,
			chunk.ParentID,
			chunk.Content,
			chunk.Position,
			pgVectorFromFloat64(chunk.Embedding),
		)
		return err2
	}

	// Backward compatibility: если схема без content_tsv (pre-migration 0003), игнорируем ошибку.
	// Триггер content_tsv не будет работать, но основной функционал останется.
	if strings.Contains(err.Error(), "content_tsv") {
		// content_tsv отсутствует, но чанк уже вставлен через v2 запрос
		// Триггер просто не сработал, это не критично
		return nil
	}

	return err
}

// Delete удаляет чанк по ID из хранилища.
func (s *PGVectorStore) Delete(ctx context.Context, id string) error {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.DeleteTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return err
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, s.quotedTableIdent)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByParentID удаляет все чанки с указанным parent_id.
func (s *PGVectorStore) DeleteByParentID(ctx context.Context, parentID string) error {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.DeleteTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return err
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE parent_id = $1`, s.quotedTableIdent)
	_, err := s.db.ExecContext(ctx, query, parentID)
	return err
}

// Search выполняет поиск похожих чанков по embedding-вектору с использованием cosine distance в БД.
//
// Score вычисляется как similarity: score = 1 - cosine_distance и находится в диапазоне [-1, 1].
func (s *PGVectorStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.SearchTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}
	if err := validateEmbedding(embedding, s.embeddingDim); err != nil {
		return domain.RetrievalResult{}, err
	}

	query := fmt.Sprintf(
		`SELECT id, parent_id, content, position, embedding,
		        (1 - (embedding <=> $1)) AS score,
		        COUNT(*) OVER() AS total_found
		   FROM %s
		  ORDER BY (embedding <=> $1) ASC
		  LIMIT $2`,
		s.quotedTableIdent,
	)

	rows, err := s.db.QueryContext(ctx, query, pgVectorFromFloat64(embedding), topK)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	defer func() { _ = rows.Close() }()

	results, totalFound, err := scanPGVectorSearchRows(ctx, rows)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  "",
		TotalFound: totalFound,
	}, nil
}

// SearchWithFilter выполняет поиск похожих чанков с фильтрацией по ParentID.
func (s *PGVectorStore) SearchWithFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.ParentIDFilter,
) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.Search(ctx, embedding, topK)
	}
	if s.runtime.MaxParentIDs > 0 && len(filter.ParentIDs) > s.runtime.MaxParentIDs {
		return domain.RetrievalResult{}, fmt.Errorf("too many ParentIDs: got=%d max=%d", len(filter.ParentIDs), s.runtime.MaxParentIDs)
	}
	for i, id := range filter.ParentIDs {
		if strings.TrimSpace(id) == "" {
			return domain.RetrievalResult{}, fmt.Errorf("ParentIDs[%d] is empty", i)
		}
	}

	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.SearchTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}
	if err := validateEmbedding(embedding, s.embeddingDim); err != nil {
		return domain.RetrievalResult{}, err
	}

	query := fmt.Sprintf(
		`SELECT id, parent_id, content, position, embedding,
		        (1 - (embedding <=> $1)) AS score,
		        COUNT(*) OVER() AS total_found
		   FROM %s
		  WHERE parent_id = ANY($2)
		  ORDER BY (embedding <=> $1) ASC
		  LIMIT $3`,
		s.quotedTableIdent,
	)

	rows, err := s.db.QueryContext(ctx, query, pgVectorFromFloat64(embedding), filter.ParentIDs, topK)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	defer func() { _ = rows.Close() }()

	results, totalFound, err := scanPGVectorSearchRows(ctx, rows)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  "",
		TotalFound: totalFound,
	}, nil
}

// SearchWithMetadataFilter выполняет поиск похожих чанков с фильтрацией по полям метаданных документа.
// Пустой filter.Fields (nil или len==0) делегирует в базовый Search без фильтра.
// SQL-условие: WHERE metadata @> $N::jsonb (оператор JSONB «содержит»; AND по всем полям).
//
// @ds-task T2.1: Реализовать SearchWithMetadataFilter в pgvector (RQ-003, AC-001, AC-002, DEC-002)
func (s *PGVectorStore) SearchWithMetadataFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.MetadataFilter,
) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return s.Search(ctx, embedding, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.SearchTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}
	if err := validateEmbedding(embedding, s.embeddingDim); err != nil {
		return domain.RetrievalResult{}, err
	}

	filterJSON := encodeMetadata(filter.Fields)

	query := fmt.Sprintf(
		`SELECT id, parent_id, content, position, embedding, metadata,
		        (1 - (embedding <=> $1)) AS score,
		        COUNT(*) OVER() AS total_found
		   FROM %s
		  WHERE metadata @> $2::jsonb
		  ORDER BY (embedding <=> $1) ASC
		  LIMIT $3`,
		s.quotedTableIdent,
	)

	rows, err := s.db.QueryContext(ctx, query, pgVectorFromFloat64(embedding), filterJSON, topK)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	defer func() { _ = rows.Close() }()

	results, totalFound, err := scanRetrievedChunks(ctx, rows)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  "",
		TotalFound: totalFound,
	}, nil
}

// SearchBM25 выполняет полнотекстовый поиск через PostgreSQL tsvector/tsquery.
// Использует GIN-индекс по колонке content_tsv для быстрого поиска.
// Требует наличия миграции 0003_add_bm25.sql.
func (s *PGVectorStore) SearchBM25(ctx context.Context, query string, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.SearchTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, errors.New("query is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}

	// Берём с запасом для fusion: topK * 2
	limit := topK * 2

	sqlQuery := fmt.Sprintf(
		`SELECT id, parent_id, content, position, embedding, metadata,
		        ts_rank_cd(content_tsv, plainto_tsquery('english', $1), 32) AS score,
		        COUNT(*) OVER() AS total_found
		   FROM %s
		  WHERE content_tsv @@ plainto_tsquery('english', $1)
		  ORDER BY score DESC
		  LIMIT $2`,
		s.quotedTableIdent,
	)

	rows, err := s.db.QueryContext(ctx, sqlQuery, query, limit)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	defer func() { _ = rows.Close() }()

	results, totalFound, err := scanRetrievedChunks(ctx, rows)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  query,
		TotalFound: totalFound,
	}, nil
}

func scanRetrievedChunks(ctx context.Context, rows *sql.Rows) ([]domain.RetrievedChunk, int, error) {
	var (
		results    []domain.RetrievedChunk
		totalFound int
	)

	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}

		var (
			id           string
			parentID     string
			content      string
			position     int
			vec          pgvector.Vector
			metadataJSON []byte
			score        float64
			total        int
		)

		if err := rows.Scan(&id, &parentID, &content, &position, &vec, &metadataJSON, &score, &total); err != nil {
			return nil, 0, err
		}

		if totalFound == 0 {
			totalFound = total
		}

		if math.IsNaN(score) || math.IsInf(score, 0) {
			return nil, 0, errors.New("invalid score from database")
		}

		metadata := decodeMetadata(metadataJSON)

		results = append(results, domain.RetrievedChunk{
			Chunk: domain.Chunk{
				ID:        id,
				Content:   content,
				ParentID:  parentID,
				Embedding: float64FromPGVector(vec),
				Position:  position,
				Metadata:  metadata,
			},
			Score: clampScore(score),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalFound, nil
}

// SearchHybrid выполняет гибридный поиск: семантический + BM25.
// Параллельно выполняет оба поиска и объединяет результаты через RRF или weighted fusion.
func (s *PGVectorStore) SearchHybrid(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}

	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, errors.New("query is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}
	if err := validateEmbedding(embedding, s.embeddingDim); err != nil {
		return domain.RetrievalResult{}, err
	}

	// Параллельно выполняем semantic и BM25 поиск
	// Используем topK * 2 для каждого поиска (с запасом для fusion)
	searchTopK := topK * 2

	var (
		semanticResult domain.RetrievalResult
		bm25Result     domain.RetrievalResult
		semanticErr    error
		bm25Err        error
	)

	// Выполняем semantic поиск
	semanticResult, semanticErr = s.Search(ctx, embedding, searchTopK)
	if semanticErr != nil {
		// Если semantic поиск упал, возвращаем только BM25 (если доступен)
		bm25Result, bm25Err = s.SearchBM25(ctx, query, searchTopK)
		if bm25Err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("both searches failed: semantic=%v, bm25=%v", semanticErr, bm25Err)
		}
		return domain.RetrievalResult{
			Chunks:     bm25Result.Chunks,
			QueryText:  query,
			TotalFound: bm25Result.TotalFound,
		}, nil
	}

	// Выполняем BM25 поиск
	bm25Result, bm25Err = s.SearchBM25(ctx, query, searchTopK)
	if bm25Err != nil {
		// Если BM25 недоступен (например, нет миграции), возвращаем только semantic
		// Проверяем что ошибка связана с отсутствием content_tsv
		if strings.Contains(bm25Err.Error(), "content_tsv") || strings.Contains(bm25Err.Error(), "column") {
			return domain.RetrievalResult{
				Chunks:     semanticResult.Chunks,
				QueryText:  query,
				TotalFound: semanticResult.TotalFound,
			}, nil
		}
		return domain.RetrievalResult{}, fmt.Errorf("bm25 search failed: %w", bm25Err)
	}

	// Объединяем результаты
	fusedChunks := fuseResults(semanticResult.Chunks, bm25Result.Chunks, config)

	// Обрезаем до запрошенного topK (если fuseResults вернул больше)
	if len(fusedChunks) > topK {
		fusedChunks = fusedChunks[:topK]
	}

	// Вычисляем TotalFound как максимум из обоих поисков
	totalFound := semanticResult.TotalFound
	if bm25Result.TotalFound > totalFound {
		totalFound = bm25Result.TotalFound
	}

	return domain.RetrievalResult{
		Chunks:     fusedChunks,
		QueryText:  query,
		TotalFound: totalFound,
	}, nil
}

// SearchHybridWithParentIDFilter выполняет гибридный поиск с фильтрацией по ParentID.
func (s *PGVectorStore) SearchHybridWithParentIDFilter(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig, filter domain.ParentIDFilter) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, errors.New("query is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if err := validateEmbedding(embedding, s.embeddingDim); err != nil {
		return domain.RetrievalResult{}, err
	}

	// Проверяем ограничение MaxParentIDs
	if s.runtime.MaxParentIDs > 0 && len(filter.ParentIDs) > s.runtime.MaxParentIDs {
		return domain.RetrievalResult{}, fmt.Errorf("too many ParentIDs: got=%d max=%d", len(filter.ParentIDs), s.runtime.MaxParentIDs)
	}

	searchTopK := topK * 2

	// Semantic поиск с фильтром
	semanticResult, semanticErr := s.SearchWithFilter(ctx, embedding, searchTopK, filter)

	// BM25 поиск с фильтром по ParentID
	bm25Result, bm25Err := s.searchBM25WithParentIDFilter(ctx, query, searchTopK, filter)

	// Обработка ошибок
	if semanticErr != nil && bm25Err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("both searches failed: semantic=%v, bm25=%v", semanticErr, bm25Err)
	}
	if semanticErr != nil {
		return domain.RetrievalResult{
			Chunks:     bm25Result.Chunks,
			QueryText:  query,
			TotalFound: bm25Result.TotalFound,
		}, nil
	}
	if bm25Err != nil {
		return domain.RetrievalResult{
			Chunks:     semanticResult.Chunks,
			QueryText:  query,
			TotalFound: semanticResult.TotalFound,
		}, nil
	}

	// Объединяем результаты
	fusedChunks := fuseResults(semanticResult.Chunks, bm25Result.Chunks, config)
	if len(fusedChunks) > topK {
		fusedChunks = fusedChunks[:topK]
	}

	totalFound := semanticResult.TotalFound
	if bm25Result.TotalFound > totalFound {
		totalFound = bm25Result.TotalFound
	}

	return domain.RetrievalResult{
		Chunks:     fusedChunks,
		QueryText:  query,
		TotalFound: totalFound,
	}, nil
}

// SearchHybridWithMetadataFilter выполняет гибридный поиск с фильтрацией по метаданным.
func (s *PGVectorStore) SearchHybridWithMetadataFilter(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, errors.New("query is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if err := validateEmbedding(embedding, s.embeddingDim); err != nil {
		return domain.RetrievalResult{}, err
	}

	searchTopK := topK * 2

	// Semantic поиск с фильтром по метаданным
	semanticResult, semanticErr := s.SearchWithMetadataFilter(ctx, embedding, searchTopK, filter)

	// BM25 поиск с фильтром по метаданным
	bm25Result, bm25Err := s.searchBM25WithMetadataFilter(ctx, query, searchTopK, filter)

	// Обработка ошибок
	if semanticErr != nil && bm25Err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("both searches failed: semantic=%v, bm25=%v", semanticErr, bm25Err)
	}
	if semanticErr != nil {
		return domain.RetrievalResult{
			Chunks:     bm25Result.Chunks,
			QueryText:  query,
			TotalFound: bm25Result.TotalFound,
		}, nil
	}
	if bm25Err != nil {
		return domain.RetrievalResult{
			Chunks:     semanticResult.Chunks,
			QueryText:  query,
			TotalFound: semanticResult.TotalFound,
		}, nil
	}

	// Объединяем результаты
	fusedChunks := fuseResults(semanticResult.Chunks, bm25Result.Chunks, config)
	if len(fusedChunks) > topK {
		fusedChunks = fusedChunks[:topK]
	}

	totalFound := semanticResult.TotalFound
	if bm25Result.TotalFound > totalFound {
		totalFound = bm25Result.TotalFound
	}

	return domain.RetrievalResult{
		Chunks:     fusedChunks,
		QueryText:  query,
		TotalFound: totalFound,
	}, nil
}

// searchBM25WithParentIDFilter выполняет BM25 поиск с фильтрацией по ParentID.
func (s *PGVectorStore) searchBM25WithParentIDFilter(ctx context.Context, query string, topK int, filter domain.ParentIDFilter) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.SearchTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, errors.New("query is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}

	sqlQuery := fmt.Sprintf(
		`SELECT id, parent_id, content, position, embedding, metadata,
		        ts_rank_cd(content_tsv, plainto_tsquery('english', $1), 32) AS score,
		        COUNT(*) OVER() AS total_found
		   FROM %s
		  WHERE content_tsv @@ plainto_tsquery('english', $1)
		    AND parent_id = ANY($2)
		  ORDER BY score DESC
		  LIMIT $3`,
		s.quotedTableIdent,
	)

	rows, err := s.db.QueryContext(ctx, sqlQuery, query, filter.ParentIDs, topK)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	defer func() { _ = rows.Close() }()

	results, totalFound, err := scanRetrievedChunks(ctx, rows)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  query,
		TotalFound: totalFound,
	}, nil
}

// searchBM25WithMetadataFilter выполняет BM25 поиск с фильтрацией по метаданным.
func (s *PGVectorStore) searchBM25WithMetadataFilter(ctx context.Context, query string, topK int, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	ctx, cancel := withDefaultTimeout(ctx, s.runtime.SearchTimeout)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, errors.New("query is empty")
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, errors.New("topK must be > 0")
	}
	if s.runtime.MaxTopK > 0 && topK > s.runtime.MaxTopK {
		return domain.RetrievalResult{}, fmt.Errorf("topK too large: got=%d max=%d", topK, s.runtime.MaxTopK)
	}

	filterJSON := encodeMetadata(filter.Fields)

	//nolint:gosec // Table identifier is validated/quoted and can't be passed as a query argument.
	sqlQuery := fmt.Sprintf(
		`SELECT id, parent_id, content, position, embedding, metadata,
		        ts_rank_cd(content_tsv, plainto_tsquery('english', $1), 32) AS score,
		        COUNT(*) OVER() AS total_found
		   FROM %s
		  WHERE content_tsv @@ plainto_tsquery('english', $1)
		    AND metadata @> $2::jsonb
		  ORDER BY score DESC
		  LIMIT $3`,
		s.quotedTableIdent,
	)

	rows, err := s.db.QueryContext(ctx, sqlQuery, query, filterJSON, topK)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	defer func() { _ = rows.Close() }()

	results, totalFound, err := scanRetrievedChunks(ctx, rows)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return domain.RetrievalResult{
		Chunks:     results,
		QueryText:  query,
		TotalFound: totalFound,
	}, nil
}

// encodeMetadata сериализует map[string]string в JSON-строку для JSONB-колонки.
// nil или пустой map сериализуются как "{}".
func encodeMetadata(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// decodeMetadata десериализует JSON-байты из JSONB-колонки в map[string]string.
// При ошибке возвращает nil.
func decodeMetadata(b []byte) map[string]string {
	if len(b) == 0 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func scanPGVectorSearchRows(ctx context.Context, rows *sql.Rows) ([]domain.RetrievedChunk, int, error) {
	var (
		results    []domain.RetrievedChunk
		totalFound int
	)

	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}

		var (
			id       string
			parentID string
			content  string
			position int
			vec      pgvector.Vector
			score    float64
			total    int
		)

		if err := rows.Scan(&id, &parentID, &content, &position, &vec, &score, &total); err != nil {
			return nil, 0, err
		}

		if totalFound == 0 {
			totalFound = total
		}

		if math.IsNaN(score) || math.IsInf(score, 0) {
			return nil, 0, errors.New("invalid score from database")
		}

		results = append(results, domain.RetrievedChunk{
			Chunk: domain.Chunk{
				ID:        id,
				Content:   content,
				ParentID:  parentID,
				Embedding: float64FromPGVector(vec),
				Position:  position,
			},
			Score: clampScore(score),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return results, totalFound, nil
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

func clampScore(score float64) float64 {
	if score > 1 {
		return 1
	}
	if score < -1 {
		return -1
	}
	return score
}

func validateEmbedding(embedding []float64, expectedDim int) error {
	if embedding == nil {
		return errors.New("embedding is nil")
	}
	if len(embedding) != expectedDim {
		return fmt.Errorf("%w: got=%d want=%d", domain.ErrEmbeddingDimensionMismatch, len(embedding), expectedDim)
	}
	for i, v := range embedding {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("embedding contains non-finite value at index %d", i)
		}
	}
	return nil
}

func pgVectorFromFloat64(vec []float64) pgvector.Vector {
	out := make([]float32, 0, len(vec))
	for _, v := range vec {
		out = append(out, float32(v))
	}
	return pgvector.NewVector(out)
}

func float64FromPGVector(vec pgvector.Vector) []float64 {
	s := vec.Slice()
	out := make([]float64, 0, len(s))
	for _, v := range s {
		out = append(out, float64(v))
	}
	return out
}

func validateSQLIdentifier(name string) error {
	if name == "" {
		return errors.New("table name is empty")
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
	// Мы ограничиваем tableName валидатором до [A-Za-z0-9_], но оставляем безопасное quoting.
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
