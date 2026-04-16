package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// ChromaStore реализует domain.VectorStore и domain.VectorStoreWithFilters через ChromaDB REST API.
// Поддерживает ChromaDB 0.4.x+ с HTTP API v1.
//
// @ds-task T1.1: Создать структуру ChromaStore с HTTP клиентом (RQ-001, RQ-002, DEC-001)
type ChromaStore struct {
	baseURL    string
	collection string
	dimension  int
	client     *http.Client
}

// ChromaRuntimeOptions задаёт ограничения и таймауты для ChromaStore.
type ChromaRuntimeOptions struct {
	SearchTimeout time.Duration
	UpsertTimeout time.Duration
	DeleteTimeout time.Duration
	MaxTopK       int
}

type chromaQueryResponse struct {
	IDs       [][]string            `json:"ids"`
	Distances [][]float64           `json:"distances"`
	Metadatas [][]map[string]string `json:"metadatas"`
	Documents [][]string            `json:"documents"`
}

func chromaToRetrievalResult(result chromaQueryResponse) domain.RetrievalResult {
	// ChromaDB возвращает массив результатов для каждого query_embedding.
	// Для одиночного запроса берём result[0].
	if len(result.IDs) == 0 || len(result.IDs[0]) == 0 {
		return domain.RetrievalResult{
			Chunks:     []domain.RetrievedChunk{},
			TotalFound: 0,
		}
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.IDs[0]))
	for i, id := range result.IDs[0] {
		chunk := domain.Chunk{ID: id}

		if len(result.Metadatas) > 0 && len(result.Metadatas[0]) > i {
			meta := result.Metadatas[0][i]
			if parentID, ok := meta["parent_id"]; ok {
				chunk.ParentID = parentID
			}
			if content, ok := meta["content"]; ok {
				chunk.Content = content
			}
			if posStr, ok := meta["position"]; ok {
				if pos, err := strconv.Atoi(posStr); err == nil {
					chunk.Position = pos
				}
			}
			chunk.Metadata = make(map[string]string)
			for k, v := range meta {
				if k != "parent_id" && k != "content" && k != "position" {
					chunk.Metadata[k] = v
				}
			}
		}

		// Content из documents если не был в metadata.
		if chunk.Content == "" && len(result.Documents) > 0 && len(result.Documents[0]) > i {
			chunk.Content = result.Documents[0][i]
		}

		score := 0.0
		if len(result.Distances) > 0 && len(result.Distances[0]) > i {
			// cosine distance -> similarity score: 1 - distance
			score = 1.0 - result.Distances[0][i]
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: score,
		})
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}
}

// Compile-time проверка интерфейсов
var _ domain.VectorStore = (*ChromaStore)(nil)
var _ domain.VectorStoreWithFilters = (*ChromaStore)(nil)
var _ domain.DocumentStore = (*ChromaStore)(nil)

// @ds-task T2.3: Compile-time assertion для CollectionManager (AC-006)
var _ domain.CollectionManager = (*ChromaStore)(nil)

// NewChromaStore создаёт новый ChromaDB-backed store.
// baseURL по умолчанию: http://localhost:8000 (ChromaDB 0.4.x+)
//
// @ds-task T1.1: Factory-функция для создания клиента (RQ-001, DEC-001)
func NewChromaStore(baseURL, collection string, dimension int) *ChromaStore {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &ChromaStore{
		baseURL:    baseURL,
		collection: collection,
		dimension:  dimension,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Upsert сохраняет или обновляет чанк в ChromaDB.
// Маппит Chunk на ChromaDB record с плоским metadata.
//
// @ds-task T2.1: Реализовать Upsert через HTTP POST /api/v1/collections/{name}/upsert (AC-001, AC-005, RQ-003, RQ-008, DEC-002)
func (s *ChromaStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// Валидация чанка
	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	// Валидация размерности эмбеддинга
	if len(chunk.Embedding) != s.dimension {
		return fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(chunk.Embedding))
	}

	// Формирование metadata — плоская структура для where-фильтров
	metadata := make(map[string]string)
	metadata["parent_id"] = chunk.ParentID
	metadata["content"] = chunk.Content
	metadata["position"] = fmt.Sprintf("%d", chunk.Position)
	for k, v := range chunk.Metadata {
		metadata[k] = v
	}

	// Формирование запроса upsert
	body := map[string]interface{}{
		"ids":        []string{chunk.ID},
		"embeddings": [][]float64{chunk.Embedding},
		"metadatas":  []map[string]string{metadata},
		"documents":  []string{chunk.Content},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/upsert", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete удаляет чанк по ID из ChromaDB.
//
// @ds-task T2.3: Реализовать Delete через HTTP POST /api/v1/collections/{name}/delete (AC-004, RQ-004)
func (s *ChromaStore) Delete(ctx context.Context, id string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}

	if id == "" {
		return domain.ErrEmptyChunkID
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", s.baseURL, s.collection)
	// ChromaDB возвращает 200 даже если ID не существует (idempotent)
	return doJSONAndExpectStatus(
		ctx, s.client,
		http.MethodPost, url,
		map[string]any{"ids": []string{id}},
		http.StatusOK,
		"chromadb",
		"chromadb",
	)
}

// DeleteByParentID удаляет все документы с указанным parent_id через ChromaDB where-фильтр.
func (s *ChromaStore) DeleteByParentID(ctx context.Context, parentID string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	body := map[string]interface{}{
		"where": map[string]interface{}{
			"parent_id": parentID,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// ChromaDB возвращает 200 даже если документы не найдены (idempotent)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(b))
	}

	return nil
}

// Search выполняет векторный поиск в ChromaDB.
//
// @ds-task T2.2: Реализовать Search через HTTP POST /api/v1/collections/{name}/query (AC-002, RQ-005)
func (s *ChromaStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	// Валидация размерности
	if len(embedding) != s.dimension {
		return domain.RetrievalResult{}, fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(embedding))
	}

	body := map[string]interface{}{
		"query_embeddings": [][]float64{embedding},
		"n_results":        topK,
		"include":          []string{"metadatas", "documents", "distances"},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result chromaQueryResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	return chromaToRetrievalResult(result), nil
}

// SearchWithFilter выполняет поиск с фильтрацией по ParentID.
//
// @ds-task T2.3: Реализовать SearchWithFilter с where-фильтром ChromaDB (AC-004, RQ-004)
func (s *ChromaStore) SearchWithFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.ParentIDFilter,
) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.Search(ctx, embedding, topK)
	}

	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	if len(embedding) != s.dimension {
		return domain.RetrievalResult{}, fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(embedding))
	}

	// Формирование where-фильтра для ChromaDB: $or по parent_id
	whereConditions := make([]map[string]interface{}, 0, len(filter.ParentIDs))
	for _, parentID := range filter.ParentIDs {
		whereConditions = append(whereConditions, map[string]interface{}{
			"parent_id": parentID,
		})
	}

	body := map[string]interface{}{
		"query_embeddings": [][]float64{embedding},
		"n_results":        topK,
		"include":          []string{"metadatas", "documents", "distances"},
	}

	if len(whereConditions) == 1 {
		// Простой фильтр
		body["where"] = whereConditions[0]
	} else {
		// OR фильтр
		body["where"] = map[string]interface{}{
			"$or": whereConditions,
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result chromaQueryResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	return chromaToRetrievalResult(result), nil
}

// SearchWithMetadataFilter выполняет поиск с фильтрацией по полям метаданных.
// Поддерживает автосоздание коллекции при отсутствии.
//
// @ds-task T2.4: Реализовать SearchWithMetadataFilter с where-фильтром и autocreate (AC-003, AC-007, RQ-006, RQ-007, DEC-003)
func (s *ChromaStore) SearchWithMetadataFilter(
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
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	if len(embedding) != s.dimension {
		return domain.RetrievalResult{}, fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(embedding))
	}

	// Формирование where-фильтра для ChromaDB: AND по всем полям metadata
	whereFilter := make(map[string]interface{})
	for k, v := range filter.Fields {
		whereFilter[k] = v
	}

	body := map[string]interface{}{
		"query_embeddings": [][]float64{embedding},
		"n_results":        topK,
		"include":          []string{"metadatas", "documents", "distances"},
		"where":            whereFilter,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Автосоздание коллекции при 404 (DEC-002: делегируем публичному CreateCollection)
	if resp.StatusCode == http.StatusNotFound {
		if err := s.CreateCollection(ctx); err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("create collection: %w", err)
		}
		// Повторный запрос
		req, _ = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err = s.client.Do(req)
		if err != nil {
			return domain.RetrievalResult{}, fmt.Errorf("chromadb request (retry): %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result chromaQueryResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	return chromaToRetrievalResult(result), nil
}

// CreateCollection создаёт коллекцию в ChromaDB.
// Idempotent: использует get_or_create=true, повторный вызов возвращает nil.
// Реализует domain.CollectionManager.
//
// @ds-task T2.1: Публичный CreateCollection заменяет приватный createCollection (AC-001, DEC-002)
func (s *ChromaStore) CreateCollection(ctx context.Context) error {
	body := map[string]interface{}{
		"name": s.collection,
		"metadata": map[string]interface{}{
			"hnsw:space": "cosine",
		},
		"get_or_create": true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/collections", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("chromadb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chromadb error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteCollection удаляет коллекцию из ChromaDB.
// Idempotent: возвращает nil при HTTP 200, 204 и 404 (коллекция не существует).
// При других статусах возвращает ошибку с кодом. Реализует domain.CollectionManager.
//
// @ds-task T2.2: DeleteCollection через DELETE /api/v1/collections/{name} (AC-002, AC-003, DEC-001)
func (s *ChromaStore) DeleteCollection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/collections/%s", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("chromadb: create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("chromadb: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		// 200, 204 — удалено; 404 — не существует (idempotent, AC-003)
		return nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chromadb: status=%d body=%s", resp.StatusCode, string(b))
	}
}

// CollectionExists проверяет существование коллекции в ChromaDB.
// Возвращает (true, nil) при HTTP 200, (false, nil) при 404,
// (false, error) при других сбоях. Реализует domain.CollectionManager.
//
// @ds-task T2.3: CollectionExists через GET /api/v1/collections/{name} (AC-004, AC-005, DEC-003)
func (s *ChromaStore) CollectionExists(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/collections/%s", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("chromadb: create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("chromadb: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("chromadb: status=%d body=%s", resp.StatusCode, string(b))
	}
}
