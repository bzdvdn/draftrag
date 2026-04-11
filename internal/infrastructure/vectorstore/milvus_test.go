package vectorstore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// milvusOKResp формирует успешный ответ Milvus REST API v2 с произвольным data-полем.
func milvusOKResp(data any) string {
	b, _ := json.Marshal(data)
	return `{"code":0,"message":"","data":` + string(b) + `}`
}

// milvusErrResp формирует ответ Milvus с ненулевым code.
func milvusErrResp(code int, msg string) string {
	return `{"code":` + itoa(code) + `,"message":"` + msg + `","data":null}`
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// readBody читает и декодирует JSON-тело запроса в map.
func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

// TestMilvusUpsert проверяет, что Upsert отправляет корректное тело (DM-002 Upsert body).
// @sk-task T4.1: тест AC-001
func TestMilvusUpsert(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp(map[string]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	chunk := domain.Chunk{
		ID:        "c1",
		Content:   "hello world",
		ParentID:  "p1",
		Embedding: []float64{0.1, 0.2},
		Metadata:  map[string]string{"source": "wiki"},
	}

	if err := store.Upsert(context.Background(), chunk); err != nil {
		t.Fatalf("Upsert error: %v", err)
	}

	if captured["collectionName"] != "docs" {
		t.Errorf("collectionName = %v, want docs", captured["collectionName"])
	}
	data, ok := captured["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data должен содержать один элемент, got %v", captured["data"])
	}
	item := data[0].(map[string]any)
	if item["id"] != "c1" {
		t.Errorf("id = %v, want c1", item["id"])
	}
	if item["text"] != "hello world" {
		t.Errorf("text = %v, want hello world", item["text"])
	}
	if item["parent_id"] != "p1" {
		t.Errorf("parent_id = %v, want p1", item["parent_id"])
	}
	meta, ok := item["metadata"].(map[string]any)
	if !ok || meta["source"] != "wiki" {
		t.Errorf("metadata[source] = %v, want wiki", meta)
	}
}

// TestMilvusDelete проверяет фильтр-выражение при удалении по ID.
// @sk-task T4.1: тест AC-002
func TestMilvusDelete(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp(map[string]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	if err := store.Delete(context.Background(), "test-id"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	if captured["collectionName"] != "docs" {
		t.Errorf("collectionName = %v, want docs", captured["collectionName"])
	}
	wantFilter := `id == "test-id"`
	if captured["filter"] != wantFilter {
		t.Errorf("filter = %q, want %q", captured["filter"], wantFilter)
	}
}

// TestMilvusSearch проверяет, что Search десериализует ответ в корректное количество чанков.
// @sk-task T4.1: тест AC-003
func TestMilvusSearch(t *testing.T) {
	respData := []map[string]any{
		{"id": "c1", "text": "first", "parent_id": "p1", "metadata": map[string]string{"lang": "ru"}, "distance": 0.9},
		{"id": "c2", "text": "second", "parent_id": "p2", "metadata": nil, "distance": 0.7},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp(respData))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	result, err := store.Search(context.Background(), []float64{0.1, 0.2}, 5)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(result.Chunks) != 2 {
		t.Errorf("chunks count = %d, want 2", len(result.Chunks))
	}
	if result.TotalFound != 2 {
		t.Errorf("TotalFound = %d, want 2", result.TotalFound)
	}
	if result.Chunks[0].Chunk.ID != "c1" {
		t.Errorf("first chunk ID = %v, want c1", result.Chunks[0].Chunk.ID)
	}
	if result.Chunks[0].Score != 0.9 {
		t.Errorf("first chunk Score = %v, want 0.9", result.Chunks[0].Score)
	}
	if result.Chunks[0].Chunk.Metadata["lang"] != "ru" {
		t.Errorf("metadata[lang] = %v, want ru", result.Chunks[0].Chunk.Metadata["lang"])
	}
}

// TestMilvusSearchEmptyResult проверяет, что пустой data возвращает пустой слайс без ошибки.
// @sk-task T4.1: тест AC-003 (edge case)
func TestMilvusSearchEmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp([]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	result, err := store.Search(context.Background(), []float64{0.1}, 5)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(result.Chunks) != 0 {
		t.Errorf("expected empty chunks, got %d", len(result.Chunks))
	}
}

// TestMilvusSearchWithFilter_WithParentIDs проверяет, что при непустых ParentIDs добавляется фильтр.
// @sk-task T4.1: тест AC-004
func TestMilvusSearchWithFilter_WithParentIDs(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp([]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	_, err := store.SearchWithFilter(context.Background(), []float64{0.1}, 5, domain.ParentIDFilter{
		ParentIDs: []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("SearchWithFilter error: %v", err)
	}

	filter, ok := capturedBody["filter"].(string)
	if !ok {
		t.Fatal("filter поле отсутствует в теле запроса")
	}
	if filter != `parent_id in ["a","b"]` {
		t.Errorf("filter = %q, want %q", filter, `parent_id in ["a","b"]`)
	}
}

// TestMilvusSearchWithFilter_EmptyParentIDs проверяет, что при пустых ParentIDs поле filter опускается.
// @sk-task T4.1: тест AC-004 (edge case)
func TestMilvusSearchWithFilter_EmptyParentIDs(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp([]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	_, err := store.SearchWithFilter(context.Background(), []float64{0.1}, 5, domain.ParentIDFilter{})
	if err != nil {
		t.Fatalf("SearchWithFilter error: %v", err)
	}

	if _, hasFilter := capturedBody["filter"]; hasFilter {
		t.Error("поле filter не должно присутствовать при пустом ParentIDs")
	}
}

// TestMilvusSearchWithMetadataFilter_WithFields проверяет точное AND-выражение.
// @sk-task T4.1: тест AC-005
func TestMilvusSearchWithMetadataFilter_WithFields(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp([]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	_, err := store.SearchWithMetadataFilter(context.Background(), []float64{0.1}, 5, domain.MetadataFilter{
		Fields: map[string]string{"source": "wiki"},
	})
	if err != nil {
		t.Fatalf("SearchWithMetadataFilter error: %v", err)
	}

	filter, ok := capturedBody["filter"].(string)
	if !ok {
		t.Fatal("filter поле отсутствует в теле запроса")
	}
	wantFilter := `metadata["source"] == "wiki"`
	if filter != wantFilter {
		t.Errorf("filter = %q, want %q", filter, wantFilter)
	}
}

// TestMilvusSearchWithMetadataFilter_EmptyFields проверяет, что пустой Fields не добавляет filter.
// @sk-task T4.1: тест AC-005 (edge case)
func TestMilvusSearchWithMetadataFilter_EmptyFields(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp([]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	_, err := store.SearchWithMetadataFilter(context.Background(), []float64{0.1}, 5, domain.MetadataFilter{})
	if err != nil {
		t.Fatalf("SearchWithMetadataFilter error: %v", err)
	}

	if _, hasFilter := capturedBody["filter"]; hasFilter {
		t.Error("поле filter не должно присутствовать при пустом Fields")
	}
}

// TestMilvusDeleteByParentID проверяет фильтр parent_id == "<id>" при удалении по ParentID.
// @sk-task T4.1: тест AC-006
func TestMilvusDeleteByParentID(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp(map[string]any{}))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	if err := store.DeleteByParentID(context.Background(), "p1"); err != nil {
		t.Fatalf("DeleteByParentID error: %v", err)
	}

	wantFilter := `parent_id == "p1"`
	if captured["filter"] != wantFilter {
		t.Errorf("filter = %q, want %q", captured["filter"], wantFilter)
	}
}

// TestMilvusDoRequest_CodeError проверяет, что ненулевой code в теле → ошибка с code/msg (AC-008).
// @sk-task T4.1: тест AC-008 (code != 0)
func TestMilvusDoRequest_CodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusErrResp(65535, "collection not found"))
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	err := store.Upsert(context.Background(), domain.Chunk{
		ID: "x", Content: "y", ParentID: "z",
	})
	if err == nil {
		t.Fatal("ожидалась ошибка при code != 0, получен nil")
	}
	if !strings.Contains(err.Error(), "code=65535") {
		t.Errorf("ошибка должна содержать code=65535, got: %v", err)
	}
	if !strings.Contains(err.Error(), "collection not found") {
		t.Errorf("ошибка должна содержать сообщение, got: %v", err)
	}
}

// TestMilvusDoRequest_HTTP5xx проверяет, что HTTP 5xx → ошибка (AC-008).
// @sk-task T4.1: тест AC-008 (HTTP 5xx)
func TestMilvusDoRequest_HTTP5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "internal error")
	}))
	defer srv.Close()

	store := NewMilvusStore(srv.URL, "docs", "")
	err := store.Delete(context.Background(), "any-id")
	if err == nil {
		t.Fatal("ожидалась ошибка при HTTP 500, получен nil")
	}
	if !strings.Contains(err.Error(), "status=500") {
		t.Errorf("ошибка должна содержать status=500, got: %v", err)
	}
}

// TestMilvusBearerToken проверяет, что при непустом token добавляется Authorization заголовок (DEC-002).
// @sk-task T4.1: тест DEC-002
func TestMilvusBearerToken(t *testing.T) {
	var capturedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, milvusOKResp(map[string]any{}))
	}))
	defer srv.Close()

	// С токеном
	store := NewMilvusStore(srv.URL, "docs", "mytoken")
	_ = store.Delete(context.Background(), "id1")
	if capturedHeader != "Bearer mytoken" {
		t.Errorf("Authorization = %q, want %q", capturedHeader, "Bearer mytoken")
	}

	// Без токена
	capturedHeader = ""
	storeNoToken := NewMilvusStore(srv.URL, "docs", "")
	_ = storeNoToken.Delete(context.Background(), "id2")
	if capturedHeader != "" {
		t.Errorf("Authorization заголовок не должен добавляться при пустом token, got %q", capturedHeader)
	}
}
