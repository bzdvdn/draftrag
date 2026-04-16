package vectorstore

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // UUID v5 uses SHA-1; this is for deterministic IDs, not for security.
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// WeaviateStore реализует domain.VectorStore и domain.VectorStoreWithFilters
// через Weaviate REST API v1 (raw HTTP, без официального SDK).
// Паттерн аналогичен QdrantStore и ChromaStore.
//
// @ds-task T1.1: Создать структуру WeaviateStore с HTTP клиентом (DEC-001)
type WeaviateStore struct {
	baseURL    string // базовый URL: scheme://host[:port]
	collection string // имя коллекции (Weaviate class)
	apiKey     string // опциональный API key для Weaviate Cloud
	client     *http.Client
}

// Compile-time проверка интерфейсов.
// @ds-task T1.1: Compile-time assertions (DEC-001)
var _ domain.VectorStore = (*WeaviateStore)(nil)
var _ domain.VectorStoreWithFilters = (*WeaviateStore)(nil)

// @sk-task T1.1: Добавить assertion для HybridSearcher (AC-001)
var _ domain.HybridSearcher = (*WeaviateStore)(nil)

// @sk-task T4.1: Добавить assertion для HybridSearcherWithFilters (AC-004, DEC-003)
var _ domain.HybridSearcherWithFilters = (*WeaviateStore)(nil)

// NewWeaviateStore создаёт WeaviateStore с указанными параметрами.
// scheme: "http" или "https"; host: "localhost:8080" или аналог.
func NewWeaviateStore(scheme, host, collection, apiKey string) *WeaviateStore {
	return &WeaviateStore{
		baseURL:    fmt.Sprintf("%s://%s", scheme, host),
		collection: collection,
		apiKey:     apiKey,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// uuidFromID генерирует детерминированный UUID v5 из строки.
// Используется DNS namespace UUID (RFC 4122, приложение C).
// Не требует внешних зависимостей — только crypto/sha1.
//
// @ds-task T1.1: UUID v5 через stdlib без новых зависимостей в go.mod (DEC-002)
func uuidFromID(id string) string {
	// DNS namespace UUID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
	ns := []byte{
		0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1,
		0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8,
	}
	h := sha1.New() //nolint:gosec // UUID v5 uses SHA-1; not used for cryptographic security.
	h.Write(ns)
	h.Write([]byte(id))
	d := h.Sum(nil)
	// Версия 5, вариант RFC4122
	d[6] = (d[6] & 0x0f) | 0x50
	d[8] = (d[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		d[0:4], d[4:6], d[6:8], d[8:10], d[10:16])
}

// setAuthHeader добавляет заголовок авторизации, если задан apiKey.
func (s *WeaviateStore) setAuthHeader(req *http.Request) {
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
}

// Upsert сохраняет или обновляет чанк в Weaviate.
// Стратегия DEC-005: сначала PUT (replace если существует), при 404 — POST (create).
// Metadata хранится дважды (DEC-003): JSON-строка chunkMetadata + flat-свойства meta_{key}.
//
// @ds-task T1.2: Upsert с dual-write и PUT→POST стратегией (AC-001, RQ-002, DEC-003, DEC-005)
func (s *WeaviateStore) Upsert(ctx context.Context, chunk domain.Chunk) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	uuid := uuidFromID(chunk.ID)

	// Формирование properties (DEC-003: dual-write metadata)
	metaJSON, err := json.Marshal(chunk.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	properties := map[string]interface{}{
		"chunkId":       chunk.ID,
		"content":       chunk.Content,
		"parentId":      chunk.ParentID,
		"position":      chunk.Position,
		"chunkMetadata": string(metaJSON),
	}
	// Dual-write: meta_{key} для server-side WHERE-фильтра в SearchWithMetadataFilter (AC-003)
	for k, v := range chunk.Metadata {
		properties["meta_"+k] = v
	}

	body := map[string]interface{}{
		"class":      s.collection,
		"id":         uuid,
		"vector":     chunk.Embedding,
		"properties": properties,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Шаг 1: PUT — заменить существующий объект (DEC-005)
	putURL := fmt.Sprintf("%s/v1/objects/%s/%s", s.baseURL, s.collection, uuid)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	s.setAuthHeader(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("weaviate PUT request: %w", err)
	}
	putStatus := resp.StatusCode
	_ = resp.Body.Close()

	if putStatus == http.StatusOK {
		return nil
	}

	// Шаг 2: объект не существует — создаём через POST
	if putStatus == http.StatusNotFound {
		postURL := fmt.Sprintf("%s/v1/objects", s.baseURL)
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("create POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		s.setAuthHeader(req)

		resp, err = s.client.Do(req)
		if err != nil {
			return fmt.Errorf("weaviate POST request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			return nil
		}
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
	}

	return fmt.Errorf("weaviate PUT error: status=%d", putStatus)
}

// Delete удаляет чанк по ID из Weaviate. Идемпотентен: 404 не является ошибкой.
//
// @ds-task T1.3: Delete с идемпотентностью — 204 и 404 возвращают nil (AC-004, RQ-006)
func (s *WeaviateStore) Delete(ctx context.Context, id string) error {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return domain.ErrEmptyChunkID
	}

	uuid := uuidFromID(id)
	url := fmt.Sprintf("%s/v1/objects/%s/%s", s.baseURL, s.collection, uuid)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	s.setAuthHeader(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("weaviate request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 204 = успешно удалено; 404 = не существовало — оба идемпотентны
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
}

// Search выполняет near-vector поиск через Weaviate GraphQL API.
// Score = certainty (0–1 для cosine similarity).
//
// @ds-task T2.1: Search через GraphQL near-vector запрос (AC-001, RQ-003, DEC-004)
func (s *WeaviateStore) Search(ctx context.Context, embedding []float64, topK int) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}
	return s.searchWithWhere(ctx, embedding, topK, "")
}

// SearchWithFilter выполняет поиск с фильтрацией по parentId.
// При пустом ParentIDs делегирует в Search без WHERE-клаузы.
//
// @ds-task T2.2: SearchWithFilter с WHERE по parentId (AC-002, RQ-004)
func (s *WeaviateStore) SearchWithFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.ParentIDFilter,
) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.Search(ctx, embedding, topK)
	}
	return s.searchWithWhereValidated(ctx, embedding, topK, whereParentIDs(filter.ParentIDs))
}

// SearchWithMetadataFilter выполняет поиск с фильтрацией по meta_* свойствам.
// При пустом filter.Fields делегирует в Search без WHERE-клаузы.
//
// @ds-task T2.3: SearchWithMetadataFilter с WHERE по meta_* (AC-003, RQ-005)
func (s *WeaviateStore) SearchWithMetadataFilter(
	ctx context.Context,
	embedding []float64,
	topK int,
	filter domain.MetadataFilter,
) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return s.Search(ctx, embedding, topK)
	}
	return s.searchWithWhereValidated(ctx, embedding, topK, whereMetadataFields(filter.Fields))
}

func (s *WeaviateStore) searchWithWhereValidated(ctx context.Context, embedding []float64, topK int, whereClause string) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}
	return s.searchWithWhere(ctx, embedding, topK, whereClause)
}

// SearchHybrid выполняет гибридный поиск (BM25 + semantic fusion) через Weaviate GraphQL API.
// Использует bm25 и nearVector с fusion-стратегией (RRF или weighted) в зависимости от HybridConfig.
//
// @sk-task T2.1: Реализация SearchHybrid с GraphQL API (AC-001, AC-002, AC-003, AC-005, AC-006, DEC-001, DEC-002, DEC-003)
func (s *WeaviateStore) SearchHybrid(
	ctx context.Context,
	query string,
	embedding []float64,
	topK int,
	config domain.HybridConfig,
) (domain.RetrievalResult, error) {
	return s.searchHybridValidated(ctx, query, embedding, topK, config, "")
}

func (s *WeaviateStore) searchHybridValidated(
	ctx context.Context,
	query string,
	embedding []float64,
	topK int,
	config domain.HybridConfig,
	whereClause string,
) (domain.RetrievalResult, error) {
	if ctx == nil {
		panic("nil context")
	}
	if err := ctx.Err(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if err := config.Validate(); err != nil {
		return domain.RetrievalResult{}, err
	}
	if strings.TrimSpace(query) == "" {
		return domain.RetrievalResult{}, domain.ErrEmptyQueryText
	}
	if topK <= 0 {
		return domain.RetrievalResult{}, domain.ErrInvalidQueryTopK
	}

	return s.searchHybridGraphQL(ctx, query, embedding, topK, config, whereClause)
}

// SearchHybridWithParentIDFilter выполняет гибридный поиск с фильтрацией по ParentID.
// При пустом ParentIDs делегирует в SearchHybrid без WHERE-клаузы.
//
// @sk-task T4.1: Реализация SearchHybridWithParentIDFilter (AC-004, DEC-003)
func (s *WeaviateStore) SearchHybridWithParentIDFilter(
	ctx context.Context,
	query string,
	embedding []float64,
	topK int,
	config domain.HybridConfig,
	filter domain.ParentIDFilter,
) (domain.RetrievalResult, error) {
	if len(filter.ParentIDs) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}
	return s.searchHybridValidated(ctx, query, embedding, topK, config, whereParentIDs(filter.ParentIDs))
}

// SearchHybridWithMetadataFilter выполняет гибридный поиск с фильтрацией по метаданным.
// При пустом filter.Fields делегирует в SearchHybrid без WHERE-клаузы.
//
// @sk-task T4.1: Реализация SearchHybridWithMetadataFilter (AC-004, DEC-003)
func (s *WeaviateStore) SearchHybridWithMetadataFilter(
	ctx context.Context,
	query string,
	embedding []float64,
	topK int,
	config domain.HybridConfig,
	filter domain.MetadataFilter,
) (domain.RetrievalResult, error) {
	if len(filter.Fields) == 0 {
		return s.SearchHybrid(ctx, query, embedding, topK, config)
	}
	return s.searchHybridValidated(ctx, query, embedding, topK, config, whereMetadataFields(filter.Fields))
}

// searchHybridGraphQL выполняет GraphQL hybrid search запрос с bm25, nearVector и fusion.
// whereClause — готовая строка вида `where: {...}` или пустая строка для фильтрации.
//
// @sk-task T2.1: GraphQL запрос с bm25 и nearVector (AC-002, AC-003, DEC-001)
func (s *WeaviateStore) searchHybridGraphQL(
	ctx context.Context,
	query string,
	embedding []float64,
	topK int,
	config domain.HybridConfig,
	whereClause string,
) (domain.RetrievalResult, error) {
	vecBytes, err := json.Marshal(embedding)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal vector: %w", err)
	}

	// Формирование hybrid аргументов в зависимости от fusion-стратегии
	var hybridArgs string
	if config.UseRRF {
		// RRF fusion: fusionType: "RankedFusion" с query и vector
		hybridArgs = fmt.Sprintf(
			`hybrid: {query: %q, vector: %s, fusionType: "RankedFusion"}`,
			query, string(vecBytes),
		)
	} else {
		// Weighted fusion: alpha = SemanticWeight (0.0 - 1.0)
		hybridArgs = fmt.Sprintf(
			`hybrid: {query: %q, vector: %s, alpha: %f}`,
			query, string(vecBytes), config.SemanticWeight,
		)
	}

	args := fmt.Sprintf("%s, limit: %d", hybridArgs, topK)
	if whereClause != "" {
		args += ", " + whereClause
	}

	gqlQuery := fmt.Sprintf(
		`{ Get { %s(%s) { chunkId content parentId position chunkMetadata _additional { id score } } } }`,
		s.collection, args,
	)

	reqBody := map[string]string{"query": gqlQuery}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/graphql", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	s.setAuthHeader(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("weaviate request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("read response: %w", err)
	}

	return s.parseHybridGraphQLResponse(respBody)
}

// parseHybridGraphQLResponse разбирает ответ Weaviate GraphQL hybrid search в domain.RetrievalResult.
// Score = _additional.score (fusion score от Weaviate).
//
// @sk-task T2.1: Парсинг ответа hybrid search (AC-002, AC-003)
func (s *WeaviateStore) parseHybridGraphQLResponse(body []byte) (domain.RetrievalResult, error) {
	var gqlResp struct {
		Data struct {
			Get map[string]json.RawMessage `json:"Get"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return domain.RetrievalResult{}, fmt.Errorf("weaviate graphql error: %s", gqlResp.Errors[0].Message)
	}

	rawCollection, ok := gqlResp.Data.Get[s.collection]
	if !ok || string(rawCollection) == "null" {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}, TotalFound: 0}, nil
	}

	var objects []struct {
		ChunkID       string `json:"chunkId"`
		Content       string `json:"content"`
		ParentID      string `json:"parentId"`
		Position      int    `json:"position"`
		ChunkMetadata string `json:"chunkMetadata"`
		Additional    struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		} `json:"_additional"`
	}

	if err := json.Unmarshal(rawCollection, &objects); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode objects: %w", err)
	}
	if len(objects) == 0 {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}, TotalFound: 0}, nil
	}

	chunks := make([]domain.RetrievedChunk, 0, len(objects))
	for _, obj := range objects {
		chunk := domain.Chunk{
			ID:       obj.ChunkID,
			Content:  obj.Content,
			ParentID: obj.ParentID,
			Position: obj.Position,
		}
		// Восстановление Metadata из JSON-строки chunkMetadata (DEC-003)
		if obj.ChunkMetadata != "" && obj.ChunkMetadata != "null" {
			var meta map[string]string
			if err := json.Unmarshal([]byte(obj.ChunkMetadata), &meta); err == nil {
				chunk.Metadata = meta
			}
		}
		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: obj.Additional.Score, // Fusion score от Weaviate
		})
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		QueryText:  "", // QueryText не заполняется в SearchHybrid
		TotalFound: len(chunks),
	}, nil
}

// searchWithWhere выполняет GraphQL near-vector запрос с опциональным WHERE-фильтром.
// whereClause — готовая строка вида `where: {...}` или пустая строка.
func (s *WeaviateStore) searchWithWhere(ctx context.Context, embedding []float64, topK int, whereClause string) (domain.RetrievalResult, error) {
	vecBytes, err := json.Marshal(embedding)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal vector: %w", err)
	}

	args := fmt.Sprintf("nearVector: {vector: %s}, limit: %d", string(vecBytes), topK)
	if whereClause != "" {
		args += ", " + whereClause
	}

	gqlQuery := fmt.Sprintf(
		`{ Get { %s(%s) { chunkId content parentId position chunkMetadata _additional { id certainty } } } }`,
		s.collection, args,
	)

	reqBody := map[string]string{"query": gqlQuery}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/graphql", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	s.setAuthHeader(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("weaviate request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.RetrievalResult{}, fmt.Errorf("weaviate error: status=%d, body=%s", resp.StatusCode, string(b))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("read response: %w", err)
	}

	return s.parseGraphQLResponse(respBody)
}

// parseGraphQLResponse разбирает ответ Weaviate GraphQL в domain.RetrievalResult.
// Score = certainty (DEC-004).
func (s *WeaviateStore) parseGraphQLResponse(body []byte) (domain.RetrievalResult, error) {
	var gqlResp struct {
		Data struct {
			Get map[string]json.RawMessage `json:"Get"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return domain.RetrievalResult{}, fmt.Errorf("weaviate graphql error: %s", gqlResp.Errors[0].Message)
	}

	rawCollection, ok := gqlResp.Data.Get[s.collection]
	if !ok || string(rawCollection) == "null" {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}, TotalFound: 0}, nil
	}

	var objects []struct {
		ChunkID       string `json:"chunkId"`
		Content       string `json:"content"`
		ParentID      string `json:"parentId"`
		Position      int    `json:"position"`
		ChunkMetadata string `json:"chunkMetadata"`
		Additional    struct {
			ID        string  `json:"id"`
			Certainty float64 `json:"certainty"`
		} `json:"_additional"`
	}

	if err := json.Unmarshal(rawCollection, &objects); err != nil {
		return domain.RetrievalResult{}, fmt.Errorf("decode objects: %w", err)
	}
	if len(objects) == 0 {
		return domain.RetrievalResult{Chunks: []domain.RetrievedChunk{}, TotalFound: 0}, nil
	}

	chunks := make([]domain.RetrievedChunk, 0, len(objects))
	for _, obj := range objects {
		chunk := domain.Chunk{
			ID:       obj.ChunkID,
			Content:  obj.Content,
			ParentID: obj.ParentID,
			Position: obj.Position,
		}
		// Восстановление Metadata из JSON-строки chunkMetadata (DEC-003)
		if obj.ChunkMetadata != "" && obj.ChunkMetadata != "null" {
			var meta map[string]string
			if err := json.Unmarshal([]byte(obj.ChunkMetadata), &meta); err == nil {
				chunk.Metadata = meta
			}
		}
		chunks = append(chunks, domain.RetrievedChunk{
			Chunk: chunk,
			Score: obj.Additional.Certainty, // DEC-004: Score = certainty (0–1 для cosine)
		})
	}

	return domain.RetrievalResult{
		Chunks:     chunks,
		TotalFound: len(chunks),
	}, nil
}

// whereParentIDs форматирует WHERE-клаузу GraphQL для фильтра по parentId.
// При одном ID использует Equal, при нескольких — Or с набором Equal.
//
// @ds-task T2.2: WHERE-блок для SearchWithFilter (AC-002)
func whereParentIDs(parentIDs []string) string {
	if len(parentIDs) == 1 {
		return fmt.Sprintf(`where: {path: ["parentId"], operator: Equal, valueText: %q}`, parentIDs[0])
	}
	operands := make([]string, len(parentIDs))
	for i, id := range parentIDs {
		operands[i] = fmt.Sprintf(`{path: ["parentId"], operator: Equal, valueText: %q}`, id)
	}
	return fmt.Sprintf(`where: {operator: Or, operands: [%s]}`, strings.Join(operands, ", "))
}

// whereMetadataFields форматирует WHERE-клаузу GraphQL для фильтра по meta_* свойствам.
// При одном поле использует Equal напрямую, при нескольких — And с operands.
//
// @ds-task T2.3: WHERE-блок для SearchWithMetadataFilter с meta_-префиксом (AC-003)
func whereMetadataFields(fields map[string]string) string {
	if len(fields) == 1 {
		for k, v := range fields {
			return fmt.Sprintf(`where: {path: ["meta_%s"], operator: Equal, valueText: %q}`, k, v)
		}
	}
	operands := make([]string, 0, len(fields))
	for k, v := range fields {
		operands = append(operands, fmt.Sprintf(`{path: ["meta_%s"], operator: Equal, valueText: %q}`, k, v))
	}
	return fmt.Sprintf(`where: {operator: And, operands: [%s]}`, strings.Join(operands, ", "))
}
