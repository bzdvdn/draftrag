package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// QdrantStore реализует domain.VectorStore и domain.VectorStoreWithFilters через Qdrant REST API.
//
// @ds-task T1.1: Создать структуру QdrantStore с HTTP клиентом (RQ-001, RQ-002)
type QdrantStore struct {
	baseURL    string
	collection string
	dimension  int
	client     *http.Client
}

// RuntimeOptions задаёт ограничения и таймауты для QdrantStore.
type QdrantRuntimeOptions struct {
	SearchTimeout time.Duration
	UpsertTimeout time.Duration
	DeleteTimeout time.Duration
	MaxTopK       int
}

// Compile-time проверка интерфейсов
var _ domain.VectorStore = (*QdrantStore)(nil)
var _ domain.VectorStoreWithFilters = (*QdrantStore)(nil)
var _ domain.DocumentStore = (*QdrantStore)(nil)
var _ domain.HybridSearcher = (*QdrantStore)(nil) // @sk-task T1.1: Добавить assertion для HybridSearcher (AC-001)
var _ domain.HybridSearcherWithFilters = (*QdrantStore)(nil) // @sk-task T3.1: Добавить assertion для HybridSearcherWithFilters (AC-004)

// NewQdrantStore создаёт новый Qdrant-backed store.
//
// @ds-task T1.1: Factory-функция для создания клиента (RQ-003)
func NewQdrantStore(baseURL, collection string, dimension int) *QdrantStore {
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}
	return &QdrantStore{
		baseURL:    baseURL,
		collection: collection,
		dimension:  dimension,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Upsert сохраняет или обновляет чанк в Qdrant.
// Маппит Chunk на Qdrant point с плоским payload.
//
// @ds-task T2.1: Реализовать Upsert через HTTP PUT /collections/{name}/points (AC-004, DEC-002, DEC-003)
func (s *QdrantStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
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

	// Формирование payload
	payload := map[string]interface{}{
		"id":        chunk.ID,
		"content":   chunk.Content,
		"parent_id": chunk.ParentID,
		"position":  chunk.Position,
	}

	// Добавление метаданных с плоским ключом metadata.k
	for k, v := range chunk.Metadata {
		payload["metadata."+k] = v
	}

	// Формирование запроса
	point := map[string]interface{}{
		"id":      chunk.ID,
		"vector":  chunk.Embedding,
		"payload": payload,
	}

	body := map[string]interface{}{
		"points": []interface{}{point},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// Delete удаляет чанк по ID из Qdrant.
//
// @ds-task T2.1: Реализовать Delete через HTTP POST /collections/{name}/points/delete (AC-004)
func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if id == "" {
		return domain.ErrEmptyChunkID
	}

	body := map[string]interface{}{
		"points": []string{id},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteByParentID удаляет все точки с указанным parent_id через Qdrant filter API.
func (s *QdrantStore) DeleteByParentID(ctx context.Context, parentID string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "parent_id",
					"match": map[string]interface{}{
						"value": parentID,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(b))
	}

	return nil
}

// Search выполняет векторный поиск в Qdrant.
//
// @ds-task T2.1: Реализовать Search через HTTP POST /collections/{name}/points/search (AC-001)
func (s *QdrantStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
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
		"vector":       embedding,
		"limit":        topK,
		"with_payload": true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.Result))
	for _, r := range result.Result {
		chunk := domain.Chunk{
			ID: r.ID,
		}

		if content, ok := r.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := r.Payload["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := r.Payload["position"].(float64); ok {
			chunk.Position = int(pos)
		}

		// Извлечение метаданных
		chunk.Metadata = make(map[string]string)
		for k, v := range r.Payload {
			if len(k) > 9 && k[:9] == "metadata." {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[k[9:]] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}, nil
}

// SearchWithFilter выполняет поиск с фильтрацией по ParentID.
//
// @ds-task T2.2: Реализовать SearchWithFilter с маппингом на Qdrant payload filter (AC-002)
func (s *QdrantStore) SearchWithFilter(
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

	// Формирование фильтра "should" для OR по ParentIDs
	shouldConditions := make([]map[string]interface{}, 0, len(filter.ParentIDs))
	for _, parentID := range filter.ParentIDs {
		shouldConditions = append(shouldConditions, map[string]interface{}{
			"key": "parent_id",
			"match": map[string]interface{}{
				"value": parentID,
			},
		})
	}

	body := map[string]interface{}{
		"vector":       embedding,
		"limit":        topK,
		"with_payload": true,
		"filter": map[string]interface{}{
			"should": shouldConditions,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.Result))
	for _, r := range result.Result {
		chunk := domain.Chunk{
			ID: r.ID,
		}

		if content, ok := r.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := r.Payload["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := r.Payload["position"].(float64); ok {
			chunk.Position = int(pos)
		}

		chunk.Metadata = make(map[string]string)
		for k, v := range r.Payload {
			if len(k) > 9 && k[:9] == "metadata." {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[k[9:]] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}, nil
}

// SearchWithMetadataFilter выполняет поиск с фильтрацией по метаданным.
//
// @ds-task T2.3: Реализовать SearchWithMetadataFilter с маппингом на Qdrant must filter (AC-003, RQ-010)
func (s *QdrantStore) SearchWithMetadataFilter(
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

	// Формирование фильтра "must" для AND по metadata полям
	mustConditions := make([]map[string]interface{}, 0, len(filter.Fields))
	for k, v := range filter.Fields {
		mustConditions = append(mustConditions, map[string]interface{}{
			"key": "metadata." + k,
			"match": map[string]interface{}{
				"value": v,
			},
		})
	}

	body := map[string]interface{}{
		"vector":       embedding,
		"limit":        topK,
		"with_payload": true,
		"filter": map[string]interface{}{
			"must": mustConditions,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.Result))
	for _, r := range result.Result {
		chunk := domain.Chunk{
			ID: r.ID,
		}

		if content, ok := r.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := r.Payload["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := r.Payload["position"].(float64); ok {
			chunk.Position = int(pos)
		}

		chunk.Metadata = make(map[string]string)
		for k, v := range r.Payload {
			if len(k) > 9 && k[:9] == "metadata." {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[k[9:]] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}, nil
}

// SearchHybrid выполняет гибридный поиск: семантический + BM25 через Query API Qdrant.
// Использует Prefetch для multi-vector retrieval и Fusion.RRF для объединения результатов.
//
// @sk-task T2.1: Реализовать SearchHybrid с Query API Prefetch и Fusion.RRF (AC-001, AC-002, AC-003, AC-005, AC-006, DEC-001, DEC-002)
func (s *QdrantStore) SearchHybrid(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig) (domain.RetrievalResult, error) {
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
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	// Валидация размерности
	if len(embedding) != s.dimension {
		return domain.RetrievalResult{}, fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(embedding))
	}

	// Формируем Query API запрос с Prefetch для dense и sparse векторов
	// Используем topK * 2 для каждого prefetch для fusion
	searchTopK := topK * 2

	body := map[string]interface{}{
		"prefetch": []map[string]interface{}{
			{
				"prefetch": []map[string]interface{}{
					{
						"query": embedding,
						"using": "dense",
						"limit": searchTopK,
					},
				},
				"query": embedding,
				"using": "dense",
				"limit": searchTopK,
			},
			{
				"query": map[string]interface{}{
					"indices": []int{},    // Sparse vector indices (BM25)
					"values":  []float64{}, // Sparse vector values
				},
				"using": "sparse",
				"limit": searchTopK,
			},
		},
		"query": map[string]interface{}{
			"fusion": map[string]interface{}{
				"type": "rrf",
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	// Query API endpoint
	url := fmt.Sprintf("%s/collections/%s/points/query", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return domain.RetrievalResult{}, ctxErr
		}
		return domain.RetrievalResult{}, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.Result))
	for _, r := range result.Result {
		chunk := domain.Chunk{
			ID: r.ID,
		}

		if content, ok := r.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := r.Payload["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := r.Payload["position"].(float64); ok {
			chunk.Position = int(pos)
		}

		// Извлечение метаданных
		chunk.Metadata = make(map[string]string)
		for k, v := range r.Payload {
			if len(k) > 9 && k[:9] == "metadata." {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[k[9:]] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	// Ограничиваем результат до topK
	if len(chunks) > topK {
		chunks = chunks[:topK]
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		QueryText:  query,
		TotalFound: len(chunks),
	}, nil
}

// SearchHybridWithParentIDFilter выполняет гибридный поиск с фильтрацией по ParentID.
// Использует Query API с Prefetch для multi-vector retrieval и Fusion.RRF для объединения результатов.
//
// @sk-task T3.1: Реализовать SearchHybridWithParentIDFilter с фильтрацией (AC-004)
func (s *QdrantStore) SearchHybridWithParentIDFilter(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig, filter domain.ParentIDFilter) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}

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
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	// Валидация размерности
	if len(embedding) != s.dimension {
		return domain.RetrievalResult{}, fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(embedding))
	}

	// Формируем фильтр "should" для OR по ParentIDs
	shouldConditions := make([]map[string]interface{}, 0, len(filter.ParentIDs))
	for _, parentID := range filter.ParentIDs {
		shouldConditions = append(shouldConditions, map[string]interface{}{
			"key": "parent_id",
			"match": map[string]interface{}{
				"value": parentID,
			},
		})
	}

	// Формируем Query API запрос с Prefetch для dense и sparse векторов
	searchTopK := topK * 2

	body := map[string]interface{}{
		"prefetch": []map[string]interface{}{
			{
				"prefetch": []map[string]interface{}{
					{
						"query": embedding,
						"using": "dense",
						"limit": searchTopK,
						"filter": map[string]interface{}{
							"should": shouldConditions,
						},
					},
				},
				"query": embedding,
				"using": "dense",
				"limit": searchTopK,
				"filter": map[string]interface{}{
					"should": shouldConditions,
				},
			},
			{
				"query": map[string]interface{}{
					"indices": []int{},
					"values":  []float64{},
				},
				"using": "sparse",
				"limit": searchTopK,
				"filter": map[string]interface{}{
					"should": shouldConditions,
				},
			},
		},
		"query": map[string]interface{}{
			"fusion": map[string]interface{}{
				"type": "rrf",
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/query", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return domain.RetrievalResult{}, ctxErr
		}
		return domain.RetrievalResult{}, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.Result))
	for _, r := range result.Result {
		chunk := domain.Chunk{
			ID: r.ID,
		}

		if content, ok := r.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := r.Payload["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := r.Payload["position"].(float64); ok {
			chunk.Position = int(pos)
		}

		chunk.Metadata = make(map[string]string)
		for k, v := range r.Payload {
			if len(k) > 9 && k[:9] == "metadata." {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[k[9:]] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	if len(chunks) > topK {
		chunks = chunks[:topK]
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		QueryText:  query,
		TotalFound: len(chunks),
	}, nil
}

// SearchHybridWithMetadataFilter выполняет гибридный поиск с фильтрацией по метаданным.
// Использует Query API с Prefetch для multi-vector retrieval и Fusion.RRF для объединения результатов.
//
// @sk-task T3.1: Реализовать SearchHybridWithMetadataFilter с фильтрацией (AC-004)
func (s *QdrantStore) SearchHybridWithMetadataFilter(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}

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
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	// Валидация размерности
	if len(embedding) != s.dimension {
		return domain.RetrievalResult{}, fmt.Errorf("%w: expected %d, got %d", domain.ErrEmbeddingDimensionMismatch, s.dimension, len(embedding))
	}

	// Формируем фильтр "must" для AND по metadata полям
	mustConditions := make([]map[string]interface{}, 0, len(filter.Fields))
	for k, v := range filter.Fields {
		mustConditions = append(mustConditions, map[string]interface{}{
			"key": "metadata." + k,
			"match": map[string]interface{}{
				"value": v,
			},
		})
	}

	// Формируем Query API запрос с Prefetch для dense и sparse векторов
	searchTopK := topK * 2

	body := map[string]interface{}{
		"prefetch": []map[string]interface{}{
			{
				"prefetch": []map[string]interface{}{
					{
						"query": embedding,
						"using": "dense",
						"limit": searchTopK,
						"filter": map[string]interface{}{
							"must": mustConditions,
						},
					},
				},
				"query": embedding,
				"using": "dense",
				"limit": searchTopK,
				"filter": map[string]interface{}{
					"must": mustConditions,
				},
			},
			{
				"query": map[string]interface{}{
					"indices": []int{},
					"values":  []float64{},
				},
				"using": "sparse",
				"limit": searchTopK,
				"filter": map[string]interface{}{
					"must": mustConditions,
				},
			},
		},
		"query": map[string]interface{}{
			"fusion": map[string]interface{}{
				"type": "rrf",
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/query", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return domain.RetrievalResult{}, ctxErr
		}
		return domain.RetrievalResult{}, fmt.Errorf("qdrant request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("qdrant error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}

	chunks := make([]domain.RetrievedChunk, 0, len(result.Result))
	for _, r := range result.Result {
		chunk := domain.Chunk{
			ID: r.ID,
		}

		if content, ok := r.Payload["content"].(string); ok {
			chunk.Content = content
		}
		if parentID, ok := r.Payload["parent_id"].(string); ok {
			chunk.ParentID = parentID
		}
		if pos, ok := r.Payload["position"].(float64); ok {
			chunk.Position = int(pos)
		}

		chunk.Metadata = make(map[string]string)
		for k, v := range r.Payload {
			if len(k) > 9 && k[:9] == "metadata." {
				if strVal, ok := v.(string); ok {
					chunk.Metadata[k[9:]] = strVal
				}
			}
		}

		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: r.Score,
		})
	}

	if len(chunks) > topK {
		chunks = chunks[:topK]
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		QueryText:  query,
		TotalFound: len(chunks),
	}, nil
}
