package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// mockBatchStore имитирует VectorStore для тестов batch-индексации.
type mockBatchStore struct {
	chunks []domain.Chunk
	mu     sync.Mutex
}

func (m *mockBatchStore) Upsert(_ context.Context, chunk domain.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunks = append(m.chunks, chunk)
	return nil
}

func (m *mockBatchStore) Delete(_ context.Context, _ string) error { return nil }
func (m *mockBatchStore) Search(_ context.Context, _ []float64, _ int) (domain.RetrievalResult, error) {
	return domain.RetrievalResult{}, nil
}

// fixedBatchEmbedder возвращает фиксированный embedding.
type fixedBatchEmbedder struct {
	delay time.Duration
}

func (f fixedBatchEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

// countingBatchEmbedder считает количество вызовов Embed.
type countingBatchEmbedder struct {
	fixedBatchEmbedder
	count atomic.Int32
}

func (c *countingBatchEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	c.count.Add(1)
	return c.fixedBatchEmbedder.Embed(ctx, text)
}

// erroringBatchEmbedder возвращает ошибку для определённых текстов.
type erroringBatchEmbedder struct {
	fixedBatchEmbedder
	errorText string
}

func (e erroringBatchEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if text == e.errorText {
		return nil, errors.New("embed failed")
	}
	return e.fixedBatchEmbedder.Embed(ctx, text)
}

// mockChunker разбивает документ на 2 чанка для тестирования chunking.
type mockChunker struct{}

func (m mockChunker) Chunk(_ context.Context, doc domain.Document) ([]domain.Chunk, error) {
	return []domain.Chunk{
		{ID: doc.ID + "#0", Content: doc.Content + " part1", ParentID: doc.ID, Position: 0},
		{ID: doc.ID + "#1", Content: doc.Content + " part2", ParentID: doc.ID, Position: 1},
	}, nil
}

// okBatchLLM для тестов.
type okBatchLLM struct{}

func (okBatchLLM) Generate(_ context.Context, _, _ string) (string, error) {
	return "ok", nil
}

// TestPipeline_IndexBatch_ParallelProcessing проверяет AC-001: параллельная обработка документов.
//
// @sk-task T3.1: Тест на параллельность IndexBatch (AC-001)
func TestPipeline_IndexBatch_ParallelProcessing(t *testing.T) {
	// 10 документов, каждый задерживает embed на 50ms
	// При concurrency=5 и последовательной обработке: 10 * 50ms = 500ms
	// При параллельной обработке с concurrency=5: ~2 * 50ms = 100ms (плюс overhead)

	store := &mockBatchStore{}
	embedder := &countingBatchEmbedder{
		fixedBatchEmbedder: fixedBatchEmbedder{delay: 50 * time.Millisecond},
	}

	// Высокий rate limit чтобы не мешать тесту параллельности
	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{
			IndexConcurrency:    5,
			IndexBatchRateLimit: 1000, // 1000 calls/sec - rate limit не bottleneck
		},
	)

	docs := make([]domain.Document, 10)
	for i := 0; i < 10; i++ {
		docs[i] = domain.Document{ID: "doc" + string(rune('0'+i)), Content: "content " + string(rune('0'+i))}
	}

	start := time.Now()
	result, err := p.IndexBatch(context.Background(), docs, 0)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// При параллельной обработке с concurrency=5 должно занять < 500ms
	// (10 документов / 5 workers * 50ms = 100ms + goroutine overhead)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected parallel processing (< 500ms), got %v", elapsed)
	}

	if result.ProcessedCount != 10 {
		t.Fatalf("expected 10 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 10 {
		t.Fatalf("expected 10 successful docs, got %d", len(result.Successful))
	}

	if len(store.chunks) != 10 {
		t.Fatalf("expected 10 chunks in store, got %d", len(store.chunks))
	}

	// Проверяем что все 10 документов были обработаны
	if embedder.count.Load() != 10 {
		t.Fatalf("expected 10 embed calls, got %d", embedder.count.Load())
	}
}

// TestPipeline_IndexBatch_RateLimiting проверяет AC-002: rate limiting.
//
// @sk-task T3.1: Тест на rate limiting IndexBatch (AC-002)
func TestPipeline_IndexBatch_RateLimiting(t *testing.T) {
	// Rate limit 10/sec, 20 документов
	// Без rate limiting: ~instant
	// С rate limiting 10/sec: минимум 2 секунды

	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{
			IndexConcurrency:    10, // больше rate limit чтобы rate limit был bottleneck
			IndexBatchRateLimit: 10,
		},
	)

	docs := make([]domain.Document, 20)
	for i := 0; i < 20; i++ {
		docs[i] = domain.Document{ID: "doc" + string(rune('0'+i%10)), Content: "content"}
	}

	start := time.Now()
	result, err := p.IndexBatch(context.Background(), docs, 0)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// С rate limit 10/sec, 20 документов должны занять минимум 1.9 секунд
	// (20 токенов / 10 токенов в секунду = 2 секунды минус первая итерация)
	if elapsed < 1900*time.Millisecond {
		t.Fatalf("expected rate limiting (>= 1.9s for 20 docs at 10/sec), got %v", elapsed)
	}

	if result.ProcessedCount != 20 {
		t.Fatalf("expected 20 processed docs, got %d", result.ProcessedCount)
	}
}

// TestPipeline_IndexBatch_PartialErrors проверяет AC-003: обработка частичных ошибок.
//
// @sk-task T3.1: Тест на частичные ошибки IndexBatch (AC-003)
func TestPipeline_IndexBatch_PartialErrors(t *testing.T) {
	store := &mockBatchStore{}
	embedder := erroringBatchEmbedder{
		errorText: "error content",
	}

	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{IndexConcurrency: 2},
	)

	docs := []domain.Document{
		{ID: "doc1", Content: "ok content 1"},
		{ID: "doc2", Content: "error content"}, // будет ошибка
		{ID: "doc3", Content: "ok content 2"},
		{ID: "doc4", Content: "error content"}, // будет ошибка
		{ID: "doc5", Content: "ok content 3"},
	}

	result, err := p.IndexBatch(context.Background(), docs, 0)

	// Ошибки документов не должны возвращать ошибку сверху
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}

	if result.ProcessedCount != 5 {
		t.Fatalf("expected 5 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 3 {
		t.Fatalf("expected 3 successful docs, got %d", len(result.Successful))
	}

	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(result.Errors))
	}

	// Проверяем что ошибки содержат DocumentID
	for _, err := range result.Errors {
		if err.DocumentID == "" {
			t.Fatalf("expected non-empty DocumentID in error")
		}
		if err.Error == nil {
			t.Fatalf("expected non-nil Error in IndexBatchError")
		}
	}

	// Проверяем что успешные документы сохранены
	successfulIDs := make(map[string]bool)
	for _, doc := range result.Successful {
		successfulIDs[doc.ID] = true
	}
	if !successfulIDs["doc1"] || !successfulIDs["doc3"] || !successfulIDs["doc5"] {
		t.Fatalf("expected doc1, doc3, doc5 to be successful")
	}
}

// TestPipeline_IndexBatch_ContextCancellation проверяет AC-004: отмена через контекст.
//
// @sk-task T3.1: Тест на отмену контекста в IndexBatch (AC-004)
func TestPipeline_IndexBatch_ContextCancellation(t *testing.T) {
	store := &mockBatchStore{}
	// Embedder с большой задержкой чтобы таймаут точно сработал
	embedder := fixedBatchEmbedder{delay: 200 * time.Millisecond}

	// Высокий rate limit чтобы таймаут был bottleneck
	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{
			IndexConcurrency:    5,
			IndexBatchRateLimit: 1000, // без rate limiting
		},
	)

	// Контекст с коротким таймаутом 100ms
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Много документов чтобы обработка точно не успела
	docs := make([]domain.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = domain.Document{ID: "doc" + string(rune('0'+i%10)), Content: "content"}
	}

	result, err := p.IndexBatch(ctx, docs, 0)

	// Должна быть ошибка контекста
	if err == nil {
		t.Fatalf("expected error from cancelled context, got nil")
	}

	// Проверяем что это ошибка контекста (DeadlineExceeded или Canceled)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error, got %v", err)
	}

	// Должен быть partial result
	if result == nil {
		t.Fatalf("expected partial result, got nil")
	}

	// ProcessedCount должен быть консистентен с Successful + Errors
	if result.ProcessedCount != len(result.Successful)+len(result.Errors) {
		t.Fatalf("invariant violated: ProcessedCount=%d, Successful=%d, Errors=%d",
			result.ProcessedCount, len(result.Successful), len(result.Errors))
	}
}

// TestPipeline_IndexBatch_WithChunker проверяет AC-005: интеграция с Chunker.
//
// @sk-task T3.1: Тест на интеграцию с Chunker в IndexBatch (AC-005)
func TestPipeline_IndexBatch_WithChunker(t *testing.T) {
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{
			IndexConcurrency: 2,
			Chunker:          mockChunker{},
		},
	)

	// 3 документа, каждый разбивается на 2 чанка
	docs := []domain.Document{
		{ID: "doc1", Content: "content1"},
		{ID: "doc2", Content: "content2"},
		{ID: "doc3", Content: "content3"},
	}

	result, err := p.IndexBatch(context.Background(), docs, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcessedCount != 3 {
		t.Fatalf("expected 3 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 3 {
		t.Fatalf("expected 3 successful docs, got %d", len(result.Successful))
	}

	// В store должно быть 6 чанков (3 документа * 2 чанка каждый)
	if len(store.chunks) != 6 {
		t.Fatalf("expected 6 chunks (3 docs * 2 chunks each), got %d", len(store.chunks))
	}

	// Проверяем что чанки имеют правильную структуру
	for _, chunk := range store.chunks {
		if chunk.ID == "" {
			t.Fatalf("expected non-empty chunk ID")
		}
		if chunk.ParentID == "" {
			t.Fatalf("expected non-empty chunk ParentID")
		}
		if chunk.Embedding == nil {
			t.Fatalf("expected non-nil chunk Embedding")
		}
	}
}

// TestPipeline_IndexBatch_EmptyDocs проверяет edge case: пустой слайс документов.
//
// @sk-task T3.1: Тест на edge case — пустой batch (краевой случай)
func TestPipeline_IndexBatch_EmptyDocs(t *testing.T) {
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{IndexConcurrency: 4},
	)

	result, err := p.IndexBatch(context.Background(), []domain.Document{}, 0)

	if err != nil {
		t.Fatalf("unexpected error for empty docs: %v", err)
	}

	if result.ProcessedCount != 0 {
		t.Fatalf("expected 0 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 0 {
		t.Fatalf("expected 0 successful docs, got %d", len(result.Successful))
	}

	if len(result.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(result.Errors))
	}
}

// TestPipeline_IndexBatch_InvalidDocument проверяет edge case: невалидный документ.
//
// @sk-task T3.1: Тест на edge case — невалидный документ (краевой случай)
func TestPipeline_IndexBatch_InvalidDocument(t *testing.T) {
	store := &mockBatchStore{}
	embedder := fixedBatchEmbedder{}

	p := NewPipelineWithConfig(
		store,
		okBatchLLM{},
		embedder,
		PipelineConfig{IndexConcurrency: 2},
	)

	docs := []domain.Document{
		{ID: "doc1", Content: "valid content"},
		{ID: "", Content: "invalid doc - no ID"}, // невалидный
		{ID: "doc3", Content: "valid content"},
	}

	result, err := p.IndexBatch(context.Background(), docs, 0)

	// Ошибки валидации документов не должны прерывать batch
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}

	if result.ProcessedCount != 3 {
		t.Fatalf("expected 3 processed docs, got %d", result.ProcessedCount)
	}

	if len(result.Successful) != 2 {
		t.Fatalf("expected 2 successful docs, got %d", len(result.Successful))
	}

	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}

	// Проверяем что ошибка - это ошибка валидации
	if result.Errors[0].Error == nil {
		t.Fatalf("expected validation error, got nil")
	}
}
