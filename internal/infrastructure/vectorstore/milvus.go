package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// milvusBaseResponse — общая обёртка ответа Milvus REST API v2.
// Используется doRequest для разбора code/message и извлечения data.
// @ds-task T1.1: обёртка ответа для централизованной обработки ошибок (AC-008, DEC-004)
type milvusBaseResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// MilvusStore реализует domain.VectorStore, domain.VectorStoreWithFilters и domain.DocumentStore
// через Milvus REST API v2 (raw HTTP, без SDK).
// Паттерн аналогичен WeaviateStore и QdrantStore в этом пакете.
// @ds-task T1.1: Создать структуру MilvusStore (AC-007, DEC-001)
type MilvusStore struct {
	baseURL    string // базовый URL к Milvus REST, напр. "http://localhost:19121"
	collection string // имя коллекции Milvus
	token      string // Bearer-токен; пустая строка — без аутентификации (DEC-002)
	client     *http.Client
}

// Compile-time проверки: MilvusStore обязан реализовывать три domain-интерфейса.
// @ds-task T1.1: Compile-time assertions (AC-007)
var _ domain.VectorStore = (*MilvusStore)(nil)
var _ domain.VectorStoreWithFilters = (*MilvusStore)(nil)
var _ domain.DocumentStore = (*MilvusStore)(nil)

// Compile-time проверки: MilvusStore реализует HybridSearcher и HybridSearcherWithFilters.
// @sk-task T1.1: Добавить assertion для HybridSearcher (AC-001, DEC-001)
var _ domain.HybridSearcher = (*MilvusStore)(nil)

// @sk-task T1.2: Добавить assertion для HybridSearcherWithFilters (AC-004, DEC-001)
var _ domain.HybridSearcherWithFilters = (*MilvusStore)(nil)

// NewMilvusStore создаёт MilvusStore с указанными параметрами.
// baseURL: полный URL к Milvus REST API, напр. "http://localhost:19121".
// token: Bearer-токен для Authorization; передайте пустую строку если аутентификация не нужна (DEC-002).
// @ds-task T1.1: Конструктор MilvusStore (DEC-001, DEC-002)
func NewMilvusStore(baseURL, collection, token string) *MilvusStore {
	return &MilvusStore{
		baseURL:    baseURL,
		collection: collection,
		token:      token,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// doRequest отправляет POST-запрос на baseURL+path с JSON-сериализованным body.
// Добавляет Authorization: Bearer <token> при непустом token (DEC-002).
// HTTP 4xx/5xx или ненулевой code в ответе → возвращает fmt.Errorf("milvus: code=%d msg=%s") (AC-008).
// Возвращает поле data из ответа для дальнейшей десериализации вызывающей стороной.
// @ds-task T1.1: Централизованный HTTP-хелпер для всех операций (AC-008, DEC-004)
func (s *MilvusStore) doRequest(ctx context.Context, path string, body any) (json.RawMessage, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("milvus: marshal request: %w", err)
	}

	url := s.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("milvus: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// DEC-002: Bearer-токен аутентификация; при пустом token заголовок не добавляется
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("milvus: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("milvus: read response: %w", err)
	}

	// HTTP 4xx/5xx → ошибка без разбора тела (AC-008)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("milvus: status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	var milvusResp milvusBaseResponse
	if err := json.Unmarshal(respBytes, &milvusResp); err != nil {
		return nil, fmt.Errorf("milvus: decode response: %w", err)
	}

	// Ненулевой code в теле ответа → ошибка (AC-008)
	if milvusResp.Code != 0 {
		return nil, fmt.Errorf("milvus: code=%d msg=%s", milvusResp.Code, milvusResp.Message)
	}

	return milvusResp.Data, nil
}

// Upsert сохраняет или обновляет чанк в Milvus-коллекции.
// Сериализует domain.Chunk в тело DM-002 (Upsert body) и отправляет POST /v2/vectordb/entities/upsert.
// metadata передаётся как JSON-объект — encoding/json сериализует map[string]string напрямую (DEC-003).
// @ds-task T2.1: Upsert через POST /v2/vectordb/entities/upsert (AC-001, DEC-003)
func (s *MilvusStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	type milvusEntity struct {
		ID       string            `json:"id"`
		Text     string            `json:"text"`
		ParentID string            `json:"parent_id"`
		Metadata map[string]string `json:"metadata"`
		Vector   []float64         `json:"vector"`
	}
	body := map[string]any{
		"collectionName": s.collection,
		"data": []milvusEntity{
			{
				ID:       chunk.ID,
				Text:     chunk.Content,
				ParentID: chunk.ParentID,
				Metadata: chunk.Metadata,
				Vector:   chunk.Embedding,
			},
		},
	}
	_, err := s.doRequest(ctx, "/v2/vectordb/entities/upsert", body)
	return err
}

// Delete удаляет чанк по ID из Milvus.
// Отправляет POST /v2/vectordb/entities/delete с фильтром id == "<id>" (DM-002).
// @ds-task T2.2: Delete с фильтром id == "<id>" (AC-002)
func (s *MilvusStore) Delete(ctx context.Context, id string) error {
	body := map[string]any{
		"collectionName": s.collection,
		"filter":         fmt.Sprintf(`id == "%s"`, id),
	}
	_, err := s.doRequest(ctx, "/v2/vectordb/entities/delete", body)
	return err
}

// DeleteByParentID удаляет все чанки с указанным parentId из Milvus.
// Отправляет POST /v2/vectordb/entities/delete с фильтром parent_id == "<parentID>" (DM-002).
// @ds-task T2.2: DeleteByParentID с фильтром parent_id == "<parentID>" (AC-006)
func (s *MilvusStore) DeleteByParentID(ctx context.Context, parentID string) error {
	body := map[string]any{
		"collectionName": s.collection,
		"filter":         fmt.Sprintf(`parent_id == "%s"`, parentID),
	}
	_, err := s.doRequest(ctx, "/v2/vectordb/entities/delete", body)
	return err
}

// Search выполняет поиск похожих чанков по вектору без дополнительного фильтра.
// @ds-task T2.3: Search через POST /v2/vectordb/entities/search (AC-003, DEC-003)
func (s *MilvusStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	return s.doSearch(ctx, embedding, topK, "")
}

// doSearch — внутренний хелпер для всех Search-вариантов.
// Отправляет POST /v2/vectordb/entities/search с DM-002 Search body.
// При пустом filter поле не включается в тело запроса (DM-002).
func (s *MilvusStore) doSearch(ctx context.Context, embedding []float64, topK int, filter string) (domain.RetrievalResult, error) {
	body := map[string]any{
		"collectionName": s.collection,
		"data":           [][]float64{embedding},
		"limit":          topK,
		"outputFields":   []string{"id", "text", "parent_id", "metadata"},
	}
	if filter != "" {
		body["filter"] = filter
	}

	data, err := s.doRequest(ctx, "/v2/vectordb/entities/search", body)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	return parseMilvusSearchData(data)
}

// parseMilvusSearchData десериализует поле data из ответа Milvus Search в domain.RetrievalResult.
// Пустой data или null → пустой слайс, не ошибка (DM-003).
// metadata десериализуется из JSON-объекта в map[string]string (DEC-003).
func parseMilvusSearchData(data json.RawMessage) (domain.RetrievalResult, error) {
	if len(data) == 0 || string(data) == "null" {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}}, nil
	}

	var items []struct {
		ID       string          `json:"id"`
		Text     string          `json:"text"`
		ParentID string          `json:"parent_id"`
		Metadata json.RawMessage `json:"metadata"`
		Distance float64         `json:"distance"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("milvus: decode search results: %w", err)
	}
	if len(items) == 0 {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}}, nil
	}

	chunks := make([]domain.RetrievedChunk, 0, len(items))
	for _, item := range items {
		chunk := domain.Chunk{
			ID:       item.ID,
			Content:  item.Text,
			ParentID: item.ParentID,
		}
		// DEC-003: десериализовать metadata из JSON-объекта в map[string]string
		if len(item.Metadata) > 0 && string(item.Metadata) != "null" {
			var meta map[string]string
			if err := json.Unmarshal(item.Metadata, &meta); err == nil {
				chunk.Metadata = meta
			}
		}
		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: item.Distance,
		})
	}
	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}, nil
}

// SearchWithFilter выполняет поиск с фильтром по parent_id.
// При непустых ParentIDs добавляет filter: parent_id in ["a","b"] (DM-002).
// При пустом ParentIDs поле filter опускается.
// @ds-task T3.1: SearchWithFilter с parent_id in [...] фильтром (AC-004)
func (s *MilvusStore) SearchWithFilter(ctx context.Context, embedding []float64, topK int, filter domain.ParentIDFilter) (domain.RetrievalResult, error) {
	filterExpr := ""
	if len(filter.ParentIDs) > 0 {
		quoted := make([]string, len(filter.ParentIDs))
		for i, id := range filter.ParentIDs {
			quoted[i] = `"` + id + `"`
		}
		filterExpr = `parent_id in [` + strings.Join(quoted, ",") + `]`
	}
	return s.doSearch(ctx, embedding, topK, filterExpr)
}

// SearchWithMetadataFilter выполняет поиск с фильтром по полям metadata.
// Строит AND-выражение вида metadata["k"] == "v" && metadata["k2"] == "v2" (DEC-003).
// При пустом Fields поле filter опускается (DM-002).
// @ds-task T3.2: SearchWithMetadataFilter с metadata["k"] == "v" && ... (AC-005, DEC-003)
func (s *MilvusStore) SearchWithMetadataFilter(ctx context.Context, embedding []float64, topK int, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	filterExpr := ""
	if len(filter.Fields) > 0 {
		parts := make([]string, 0, len(filter.Fields))
		for k, v := range filter.Fields {
			parts = append(parts, fmt.Sprintf(`metadata["%s"] == "%s"`, k, v))
		}
		filterExpr = strings.Join(parts, " && ")
	}
	return s.doSearch(ctx, embedding, topK, filterExpr)
}

// SearchHybrid выполняет гибридный поиск: BM25 (sparse) + semantic (dense).
// Валидирует HybridConfig, создаёт AnnSearchRequest для text_sparse и text_dense,
// вызывает hybrid_search() через POST /v2/vectordb/entities/hybrid_search (DEC-002).
// @sk-task T2.1: Реализовать SearchHybrid с Multi-Vector Search API (AC-001, AC-002, AC-003, AC-005, AC-006, DEC-001, DEC-002)
func (s *MilvusStore) SearchHybrid(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig) (domain.RetrievalResult, error) {
	// Валидация HybridConfig перед выполнением поиска (AC-005)
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("milvus: invalid hybrid config: %w", err)
	}

	// Создаём AnnSearchRequest для BM25 (sparse) и dense векторов (DEC-002)
	sparseRequest := map[string]any{
		"data":      []string{query},
		"annsField": "text_sparse",
		"limit":     topK,
	}
	denseRequest := map[string]any{
		"data":      [][]float64{embedding},
		"annsField": "text_dense",
		"limit":     topK,
		"param":     map[string]any{"nprobe": 10},
	}
	requests := []map[string]any{sparseRequest, denseRequest}

	// Выбор rerank strategy на основе HybridConfig.UseRRF (AC-003)
	var rerank string
	if config.UseRRF {
		rerank = "rrf"
	} else {
		rerank = "weighted"
	}

	body := map[string]any{
		"collectionName": s.collection,
		"requests":       requests,
		"ranker": map[string]any{
			"type": rerank,
			"params": map[string]any{
				"k": config.RRFK,
			},
		},
		"topK": topK,
	}

	data, err := s.doRequest(ctx, "/v2/vectordb/entities/hybrid_search", body)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return parseMilvusHybridSearchData(data)
}

// parseMilvusHybridSearchData десериализует поле data из ответа Milvus hybrid_search в domain.RetrievalResult.
// Пустой data или null → пустой слайс, не ошибка (AC-006).
// metadata десериализуется из JSON-объекта в map[string]string (DEC-003).
// @sk-task T2.2: Реализовать парсинг ответа hybrid_search() (AC-002, AC-006)
func parseMilvusHybridSearchData(data json.RawMessage) (domain.RetrievalResult, error) {
	if len(data) == 0 || string(data) == "null" {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}}, nil
	}

	var response struct {
		Results []struct {
			ID       string          `json:"id"`
			Text     string          `json:"text"`
			ParentID string          `json:"parent_id"`
			Metadata json.RawMessage `json:"metadata"`
			Score    float64         `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("milvus: decode hybrid search results: %w", err)
	}
	if len(response.Results) == 0 {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}}, nil
	}

	chunks := make([]domain.RetrievedChunk, 0, len(response.Results))
	for _, item := range response.Results {
		chunk := domain.Chunk{
			ID:       item.ID,
			Content:  item.Text,
			ParentID: item.ParentID,
		}
		// DEC-003: десериализовать metadata из JSON-объекта в map[string]string
		if len(item.Metadata) > 0 && string(item.Metadata) != "null" {
			var meta map[string]string
			if err := json.Unmarshal(item.Metadata, &meta); err == nil {
				chunk.Metadata = meta
			}
		}
		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: item.Score,
		})
	}
	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}, nil
}

// SearchHybridWithParentIDFilter выполняет гибридный поиск с фильтрацией по parentId.
// Добавляет expr фильтр в AnnSearchRequest для text_sparse и text_dense (DEC-003).
// При пустом ParentIDs делегирует в SearchHybrid без фильтра.
// @sk-task T3.1: Реализовать SearchHybridWithParentIDFilter с фильтрацией (AC-004, DEC-003)
func (s *MilvusStore) SearchHybridWithParentIDFilter(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig, filter domain.ParentIDFilter) (domain.RetrievalResult, error) {
	// При пустом фильтре делегируем в SearchHybrid без фильтрации
	if len(filter.ParentIDs) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}

	// Валидация HybridConfig перед выполнением поиска (AC-005)
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("milvus: invalid hybrid config: %w", err)
	}

	// Создаём AnnSearchRequest для BM25 (sparse) и dense векторов с expr фильтром (DEC-003)
	quoted := make([]string, len(filter.ParentIDs))
	for i, id := range filter.ParentIDs {
		quoted[i] = `"` + id + `"`
	}
	expr := `parent_id in [` + strings.Join(quoted, ",") + `]`

	sparseRequest := map[string]any{
		"data":      []string{query},
		"annsField": "text_sparse",
		"limit":     topK,
		"expr":      expr,
	}
	denseRequest := map[string]any{
		"data":      [][]float64{embedding},
		"annsField": "text_dense",
		"limit":     topK,
		"param":     map[string]any{"nprobe": 10},
		"expr":      expr,
	}
	requests := []map[string]any{sparseRequest, denseRequest}

	// Выбор rerank strategy на основе HybridConfig.UseRRF (AC-003)
	var rerank string
	if config.UseRRF {
		rerank = "rrf"
	} else {
		rerank = "weighted"
	}

	body := map[string]any{
		"collectionName": s.collection,
		"requests":       requests,
		"ranker": map[string]any{
			"type": rerank,
			"params": map[string]any{
				"k": config.RRFK,
			},
		},
		"topK": topK,
	}

	data, err := s.doRequest(ctx, "/v2/vectordb/entities/hybrid_search", body)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return parseMilvusHybridSearchData(data)
}

// SearchHybridWithMetadataFilter выполняет гибридный поиск с фильтрацией по metadata.
// Добавляет expr фильтр в AnnSearchRequest для text_sparse и text_dense (DEC-003).
// При пустом Fields делегирует в SearchHybrid без фильтра.
// @sk-task T3.2: Реализовать SearchHybridWithMetadataFilter с фильтрацией (AC-004, DEC-003)
func (s *MilvusStore) SearchHybridWithMetadataFilter(ctx context.Context, query string, embedding []float64, topK int, config domain.HybridConfig, filter domain.MetadataFilter) (domain.RetrievalResult, error) {
	// При пустом фильтре делегируем в SearchHybrid без фильтрации
	if len(filter.Fields) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}

	// Валидация HybridConfig перед выполнением поиска (AC-005)
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("milvus: invalid hybrid config: %w", err)
	}

	// Создаём expr фильтр для metadata (DEC-003)
	parts := make([]string, 0, len(filter.Fields))
	for k, v := range filter.Fields {
		parts = append(parts, fmt.Sprintf(`metadata["%s"] == "%s"`, k, v))
	}
	expr := strings.Join(parts, " && ")

	// Создаём AnnSearchRequest для BM25 (sparse) и dense векторов с expr фильтром (DEC-003)
	sparseRequest := map[string]any{
		"data":      []string{query},
		"annsField": "text_sparse",
		"limit":     topK,
		"expr":      expr,
	}
	denseRequest := map[string]any{
		"data":      [][]float64{embedding},
		"annsField": "text_dense",
		"limit":     topK,
		"param":     map[string]any{"nprobe": 10},
		"expr":      expr,
	}
	requests := []map[string]any{sparseRequest, denseRequest}

	// Выбор rerank strategy на основе HybridConfig.UseRRF (AC-003)
	var rerank string
	if config.UseRRF {
		rerank = "rrf"
	} else {
		rerank = "weighted"
	}

	body := map[string]any{
		"collectionName": s.collection,
		"requests":       requests,
		"ranker": map[string]any{
			"type": rerank,
			"params": map[string]any{
				"k": config.RRFK,
			},
		},
		"topK": topK,
	}

	data, err := s.doRequest(ctx, "/v2/vectordb/entities/hybrid_search", body)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	return parseMilvusHybridSearchData(data)
}
